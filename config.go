package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type NodeConfig struct {
	Name  string   `yaml:"name"`
	Path  string   `yaml:"path"`
	Build string   `yaml:"build"`
	Deps  []string `yaml:"deps"`
}

type Config struct {
	Exec  string       `yaml:"exec"`
	Nodes []NodeConfig `yaml:"nodes"`
}

func loadConfig(path string) (*Config, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(f, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
