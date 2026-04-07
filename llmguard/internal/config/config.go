package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Upstream UpstreamConfig `yaml:"upstream"`
	Scanners ScannersConfig `yaml:"scanners"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type UpstreamConfig struct {
	URL string `yaml:"url"`
}

type ScannersConfig struct {
	Input  []ScannerConfig `yaml:"input"`
	Output []ScannerConfig `yaml:"output"`
}

type ScannerConfig struct {
	Name      string  `yaml:"name"`
	MaxTokens int     `yaml:"max_tokens,omitempty"`
	Threshold float64 `yaml:"threshold,omitempty"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:         8080,
			ReadTimeout:  120 * time.Second,
			WriteTimeout: 120 * time.Second,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Upstream.URL == "" {
		return nil, fmt.Errorf("upstream.url is required")
	}

	return cfg, nil
}
