package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	installDir := flag.String("dir", "", "Installation directory")
	token := flag.String("token", "", "Authentication token")
	grpcHost := flag.String("grpc-host", "", "gRPC host")
	grpcPort := flag.String("grpc-port", "", "gRPC port")
	grpcURL := flag.String("grpc-url", "", "gRPC URL")
	runnerImage := flag.String("runner-image", "", "Runner image (default: ghcr.io/chaitin/monkeycode-runner:latest)")
	flag.Parse()

	if *installDir == "" {
		reader := &InteractiveReader{}

		fmt.Println("")
		fmt.Println("╔══════════════════════════════════════════════════════════════════════════════════════╗")
		fmt.Println("║                                                                                      ║")
		fmt.Println("║   ███    ███  ██████  ███    ██ ██   ██ ███████ ██    ██  ██████  ██████  ██████  ║")
		fmt.Println("║   ████  ████ ██    ██ ████   ██ ██  ██  ██       ██  ██  ██      ██    ██ ██   ██ ║")
		fmt.Println("║   ██ ████ ██ ██    ██ ██ ██  ██ █████   █████     ████   ██      ██    ██ ██   ██ ║")
		fmt.Println("║   ██  ██  ██ ██    ██ ██  ██ ██ ██  ██  ██         ██    ██      ██    ██ ██   ██ ║")
		fmt.Println("║   ██      ██  ██████  ██   ████ ██   ██ ███████    ██     ██████  ██████  ██████  ║")
		fmt.Println("║                                                                                      ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════════════════════════════╝")
		fmt.Println("")

		dir, err := reader.ReadString("请输入安装目录", "/data/monkeycode_runner")
		if err != nil {
			log.Fatalf("读取安装目录失败: %v", err)
		}
		installDir = &dir
	}

	installer := NewInstaller(*installDir, *runnerImage)

	if err := installer.CheckSystem(); err != nil {
		log.Fatalf("系统检查失败: %v", err)
	}

	envVars := map[string]string{}
	if *token != "" {
		envVars["TOKEN"] = *token
	}
	if *grpcHost != "" {
		envVars["GRPC_HOST"] = *grpcHost
	}
	if *grpcPort != "" {
		envVars["GRPC_PORT"] = *grpcPort
	}
	if *grpcURL != "" {
		envVars["GRPC_URL"] = *grpcURL
	}

	if err := installer.Install(envVars); err != nil {
		log.Fatalf("安装失败: %v", err)
	}

	fmt.Println("")
	fmt.Println("✅ 安装完成!")
	fmt.Println("")
}
