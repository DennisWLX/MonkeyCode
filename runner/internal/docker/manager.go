package docker

import (
	"context"
	"io"
	"log/slog"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Manager struct {
	client *client.Client
	logger *slog.Logger
}

func NewManager(logger *slog.Logger) (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &Manager{
		client: cli,
		logger: logger,
	}, nil
}

func (m *Manager) Close() error {
	return m.client.Close()
}

func (m *Manager) Ping(ctx context.Context) error {
	_, err := m.client.Ping(ctx)
	return err
}

func (m *Manager) ListContainers(ctx context.Context) ([]container.Summary, error) {
	return m.client.ContainerList(ctx, container.ListOptions{All: true})
}

func (m *Manager) GetContainer(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return m.client.ContainerInspect(ctx, containerID)
}

func (m *Manager) ExecInContainer(ctx context.Context, containerID string, cmd []string, stdin io.Reader, stdout io.Writer) (int, error) {
	execConfig := container.ExecOptions{
		AttachStdin:  stdin != nil,
		AttachStdout: stdout != nil,
		AttachStderr: true,
		Cmd:          cmd,
	}

	execResp, err := m.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return -1, err
	}

	attachResp, err := m.client.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return -1, err
	}
	defer attachResp.Close()

	if stdout != nil {
		go io.Copy(stdout, attachResp.Reader)
	}

	inspectResp, err := m.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return -1, err
	}

	return inspectResp.ExitCode, nil
}
