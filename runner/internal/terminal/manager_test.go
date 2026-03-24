package terminal

import (
	"log/slog"
	"testing"

	"github.com/docker/docker/client"
)

func TestNewManager(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	mgr := NewManager(logger, cli)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.terminals == nil {
		t.Error("terminals map is nil")
	}

	if mgr.logger == nil {
		t.Error("logger is nil")
	}

	if mgr.dockerCli == nil {
		t.Error("dockerCli is nil")
	}

	if mgr.Count() != 0 {
		t.Errorf("expected count 0, got %d", mgr.Count())
	}
}

func TestNewTerminal(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	term := New("term-1", "vm-1", "container-123", cli)

	if term.ID != "term-1" {
		t.Errorf("ID = %s; want %s", term.ID, "term-1")
	}

	if term.VMID != "vm-1" {
		t.Errorf("VMID = %s; want %s", term.VMID, "vm-1")
	}

	if term.ContainerID != "container-123" {
		t.Errorf("ContainerID = %s; want %s", term.ContainerID, "container-123")
	}

	if term.resizeCh == nil {
		t.Error("resizeCh is nil")
	}

	if term.done == nil {
		t.Error("done is nil")
	}
}

func TestTerminalClose(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	term := New("term-1", "vm-1", "container-123", cli)

	err = term.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	err = term.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

func TestTerminalDone(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	term := New("term-1", "vm-1", "container-123", cli)

	done := term.Done()
	if done == nil {
		t.Error("Done() returned nil channel")
	}

	select {
	case <-done:
		t.Error("Done() should not be closed initially")
	default:
	}

	term.Close()

	select {
	case <-done:
	default:
		t.Error("Done() should be closed after Close()")
	}
}

func TestTerminalWriteBeforeStart(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	term := New("term-1", "vm-1", "container-123", cli)

	_, err = term.Write([]byte("test"))
	if err == nil {
		t.Error("Write() should fail before Start()")
	}
}

func TestTerminalReadBeforeStart(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}

	term := New("term-1", "vm-1", "container-123", cli)

	_, err = term.Read(make([]byte, 10))
	if err == nil {
		t.Error("Read() should fail before Start()")
	}
}

func TestResizeStruct(t *testing.T) {
	resize := Resize{Cols: 100, Rows: 50}

	if resize.Cols != 100 {
		t.Errorf("Cols = %d; want 100", resize.Cols)
	}

	if resize.Rows != 50 {
		t.Errorf("Rows = %d; want 50", resize.Rows)
	}
}
