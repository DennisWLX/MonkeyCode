package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
)

type Installer struct {
	installDir string
	docker     *DockerClient
}

func NewInstaller(installDir string) *Installer {
	return &Installer{
		installDir: installDir,
		docker:     NewDockerClient(),
	}
}

func (i *Installer) CheckSystem() error {
	fmt.Println("📋 正在检查系统要求...")

	if !i.docker.IsInstalled() {
		fmt.Println("⚠️  Docker 未安装，正在安装 Docker...")
		if err := i.installDocker(); err != nil {
			return fmt.Errorf("安装 Docker 失败: %w", err)
		}
		fmt.Println("✅ Docker 安装成功")
	}

	if !i.docker.IsRunning() {
		fmt.Println("⚠️  Docker 未运行，正在启动 Docker...")
		if err := i.docker.Start(); err != nil {
			return fmt.Errorf("启动 Docker 失败: %w", err)
		}
		fmt.Println("✅ Docker 启动成功")
	}

	fmt.Println("✅ 系统要求检查通过")
	return nil
}

func (i *Installer) Install(envVars map[string]string) error {
	fmt.Println("")
	fmt.Println("📦 开始安装...")

	if err := os.MkdirAll(i.installDir, 0755); err != nil {
		return fmt.Errorf("创建安装目录失败: %w", err)
	}
	fmt.Printf("✅ 安装目录创建成功: %s\n", i.installDir)

	composeURL := "https://raw.githubusercontent.com/chaitin/MonkeyCode/main/docker-compose.yml"
	envTemplateURL := "https://raw.githubusercontent.com/chaitin/MonkeyCode/main/env.template"

	composePath := filepath.Join(i.installDir, "docker-compose.yml")
	envTemplatePath := filepath.Join(i.installDir, "env.template")

	fmt.Printf("📥 正在下载 docker-compose.yml...\n")
	if err := i.downloadFile(composeURL, composePath); err != nil {
		return fmt.Errorf("下载 docker-compose.yml 失败: %w", err)
	}
	fmt.Printf("✅ docker-compose.yml 下载成功\n")

	fmt.Printf("📥 正在下载 env.template...\n")
	if err := i.downloadFile(envTemplateURL, envTemplatePath); err != nil {
		return fmt.Errorf("下载 env.template 失败: %w", err)
	}
	fmt.Printf("✅ env.template 下载成功\n")

	fmt.Println("🔧 正在生成变量...")
	vars, err := i.generateVars(envVars)
	if err != nil {
		return fmt.Errorf("生成变量失败: %w", err)
	}
	fmt.Printf("✅ 变量生成成功\n")

	envPath := filepath.Join(i.installDir, ".env")
	fmt.Println("📝 正在渲染 .env 文件...")
	if err := i.renderEnvFile(envTemplatePath, envPath, vars); err != nil {
		return fmt.Errorf("渲染 .env 文件失败: %w", err)
	}
	fmt.Printf("✅ .env 文件渲染成功\n")

	fmt.Println("🐳 正在拉取镜像...")
	if err := i.pullImages(); err != nil {
		return fmt.Errorf("拉取镜像失败: %w", err)
	}
	fmt.Printf("✅ 镜像拉取成功\n")

	fmt.Println("🚀 正在启动容器...")
	if err := i.startContainers(); err != nil {
		return fmt.Errorf("启动容器失败: %w", err)
	}
	fmt.Printf("✅ 容器启动成功\n")

	return nil
}

func (i *Installer) downloadFile(url, destPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	content, err := os.ReadFile(destPath)
	if err == nil && len(content) > 0 {
		return nil
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return err
	}

	return os.WriteFile(destPath, buf.Bytes(), 0644)
}

func (i *Installer) generateVars(userVars map[string]string) (map[string]interface{}, error) {
	hostUUID := uuid.New().String()

	vars := map[string]interface{}{
		"HOST_ID":  hostUUID,
		"HOST_UUID": hostUUID,
	}

	for k, v := range userVars {
		vars[k] = v
	}

	if _, ok := vars["TOKEN"]; !ok {
		vars["TOKEN"] = uuid.New().String()
	}

	if _, ok := vars["GRPC_HOST"]; !ok {
		vars["GRPC_HOST"] = "localhost"
	}

	if _, ok := vars["GRPC_PORT"]; !ok {
		vars["GRPC_PORT"] = "50051"
	}

	if _, ok := vars["GRPC_URL"]; !ok {
		vars["GRPC_URL"] = "localhost:50051"
	}

	return vars, nil
}

func (i *Installer) renderEnvFile(templatePath, destPath string, vars map[string]interface{}) error {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, vars); err != nil {
		return err
	}

	return os.WriteFile(destPath, buf.Bytes(), 0644)
}

func (i *Installer) pullImages() error {
	images := []string{
		"ghcr.io/chaitin/monkeycode/watchtower:latest",
		"ghcr.io/chaitin/monkeycode/orchestrator:latest",
	}

	for _, image := range images {
		fmt.Printf("   正在拉取: %s\n", image)
		if err := i.docker.PullImage(image); err != nil {
			return fmt.Errorf("拉取镜像 %s 失败: %w", image, err)
		}
	}

	return nil
}

func (i *Installer) startContainers() error {
	cmd := exec.Command("docker-compose", "-f", filepath.Join(i.installDir, "docker-compose.yml"), "up", "-d")
	cmd.Dir = i.installDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启动容器失败: %w", err)
	}

	time.Sleep(2 * time.Second)

	statusCmd := exec.Command("docker", "ps", "--filter", "name=monkeycode", "--format", "{{.Names}}")
	statusCmd.Dir = i.installDir
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("检查容器状态失败: %w", err)
	}

	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, name := range containers {
		if name != "" {
			fmt.Printf("   ✅ 容器已启动: %s\n", name)
		}
	}

	return nil
}

func (i *Installer) installDocker() error {
	fmt.Println("   正在检测操作系统...")

	osRelease, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return fmt.Errorf("无法读取系统信息: %w", err)
	}

	isUbuntu := strings.Contains(string(osRelease), "Ubuntu")
	isDebian := strings.Contains(string(osRelease), "Debian")
	isCentOS := strings.Contains(string(osRelease), "CentOS")
	isRHEL := strings.Contains(string(osRelease), "Red Hat")

	var installCmd []string

	if isUbuntu || isDebian {
		installCmd = []string{
			"apt-get", "update",
		}
		if err := exec.Command(installCmd[0], installCmd[1:]...).Run(); err != nil {
			return err
		}

		installCmd = []string{
			"apt-get", "install", "-y",
			"ca-certificates", "curl", "gnupg",
		}
		if err := exec.Command(installCmd[0], installCmd[1:]...).Run(); err != nil {
			return err
		}

		cmd := exec.Command("bash", "-c",
			`install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/${ID}/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${ID} $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin`)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()

	} else if isCentOS || isRHEL {
		cmd := exec.Command("bash", "-c",
			`yum install -y yum-utils
yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin`)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return fmt.Errorf("不支持的操作系统")
}
