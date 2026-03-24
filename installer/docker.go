package main

import (
	"fmt"
	"os/exec"
	"strings"
)

type DockerClient struct{}

func NewDockerClient() *DockerClient {
	return &DockerClient{}
}

func (d *DockerClient) IsInstalled() bool {
	cmd := exec.Command("docker", "--version")
	err := cmd.Run()
	return err == nil
}

func (d *DockerClient) IsRunning() bool {
	cmd := exec.Command("docker", "info")
	err := cmd.Run()
	return err == nil
}

func (d *DockerClient) Start() error {
	cmd := exec.Command("systemctl", "start", "docker")
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("service", "docker", "start")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("无法启动 Docker 服务")
		}
	}
	return nil
}

func (d *DockerClient) PullImage(image string) error {
	cmd := exec.Command("docker", "pull", image)
	cmd.Stdout = &LogWriter{}
	cmd.Stderr = &LogWriter{}
	return cmd.Run()
}

func (d *DockerClient) Images() ([]string, error) {
	cmd := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	images := strings.Split(strings.TrimSpace(string(output)), "\n")
	return images, nil
}

func (d *DockerClient) Containers() ([]string, error) {
	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	return containers, nil
}

type LogWriter struct{}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
