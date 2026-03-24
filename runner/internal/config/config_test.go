package config

import (
	"os"
	"testing"
)

func TestLoadDefault(t *testing.T) {
	os.Unsetenv("TOKEN")
	os.Unsetenv("GRPC_URL")

	cfg := Load()

	if cfg.Token != "" {
		t.Errorf("expected Token '', got %s", cfg.Token)
	}

	if cfg.GRPCAddr != "localhost:50051" {
		t.Errorf("expected GRPCAddr 'localhost:50051', got %s", cfg.GRPCAddr)
	}
}

func TestLoadWithEnv(t *testing.T) {
	os.Setenv("TOKEN", "test-token-123")
	defer os.Unsetenv("TOKEN")

	os.Setenv("GRPC_URL", "grpc.example.com:50052")
	defer os.Unsetenv("GRPC_URL")

	cfg := Load()

	if cfg.Token != "test-token-123" {
		t.Errorf("expected Token 'test-token-123', got %s", cfg.Token)
	}

	if cfg.GRPCAddr != "grpc.example.com:50052" {
		t.Errorf("expected GRPCAddr 'grpc.example.com:50052', got %s", cfg.GRPCAddr)
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		setValue     string
		expected     string
	}{
		{
			name:         "returns value when set",
			key:          "TEST_KEY_1",
			defaultValue: "default",
			setValue:     "value",
			expected:     "value",
		},
		{
			name:         "returns default when not set",
			key:          "TEST_KEY_2",
			defaultValue: "default",
			setValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setValue != "" {
				os.Setenv(tt.key, tt.setValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv(%s, %s) = %s; want %s", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		setValue     string
		expected     int
	}{
		{
			name:         "returns int value when set",
			key:          "TEST_INT_1",
			defaultValue: 10,
			setValue:     "42",
			expected:     42,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_INT_2",
			defaultValue: 10,
			setValue:     "",
			expected:     10,
		},
		{
			name:         "returns default when value is invalid",
			key:          "TEST_INT_3",
			defaultValue: 10,
			setValue:     "invalid",
			expected:     10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setValue != "" {
				os.Setenv(tt.key, tt.setValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvInt(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvInt(%s, %d) = %d; want %d", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}
