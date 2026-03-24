package portforward

import (
	"context"
	"log/slog"
	"testing"
)

func TestNewManager(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	mgr := NewManager(logger)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.forwarder == nil {
		t.Error("forwarder is nil")
	}

	if mgr.forwards == nil {
		t.Error("forwards map is nil")
	}

	if mgr.Count() != 0 {
		t.Errorf("expected count 0, got %d", mgr.Count())
	}
}

func TestNewForwarder(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	fwd := NewForwarder(logger)

	if fwd == nil {
		t.Fatal("NewForwarder returned nil")
	}
}

func TestProtocolForNet(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	fwd := NewForwarder(logger)

	tests := []struct {
		name     string
		protocol string
		expected string
	}{
		{
			name:     "tcp protocol",
			protocol: "tcp",
			expected: "tcp",
		},
		{
			name:     "udp protocol",
			protocol: "udp",
			expected: "udp",
		},
		{
			name:     "unknown protocol defaults to tcp",
			protocol: "unknown",
			expected: "tcp",
		},
		{
			name:     "empty protocol defaults to tcp",
			protocol: "",
			expected: "tcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fwd.protocolForNet(tt.protocol)
			if result != tt.expected {
				t.Errorf("protocolForNet(%s) = %s; want %s", tt.protocol, result, tt.expected)
			}
		})
	}
}

func TestForwarderStartValidation(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	fwd := NewForwarder(logger)

	ctx := context.Background()

	_, err := fwd.Start(ctx, "", 9000, 8080, "tcp")
	if err == nil {
		t.Error("expected error for empty container ID")
	}
}
