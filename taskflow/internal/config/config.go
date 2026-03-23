package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Redis   RedisConfig   `yaml:"redis"`
	Log     LogConfig     `yaml:"log"`
	Backend BackendConfig `yaml:"backend"`
	Runner  RunnerConfig  `yaml:"runner"`
}

type ServerConfig struct {
	HTTPAddr string `yaml:"http_addr"`
	GRPCAddr string `yaml:"grpc_addr"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

type BackendConfig struct {
	Addr string `yaml:"addr"`
}

type RunnerConfig struct {
	Token           string `yaml:"token"`
	HeartbeatTimout int    `yaml:"heartbeat_timeout"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	setDefaults(&cfg)
	return &cfg, nil
}

func setDefaults(cfg *Config) {
	if cfg.Server.HTTPAddr == "" {
		cfg.Server.HTTPAddr = ":8888"
	}
	if cfg.Server.GRPCAddr == "" {
		cfg.Server.GRPCAddr = ":50051"
	}
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "localhost:6379"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Backend.Addr == "" {
		cfg.Backend.Addr = "http://localhost:8888"
	}
	if cfg.Runner.HeartbeatTimout == 0 {
		cfg.Runner.HeartbeatTimout = 60
	}
}
