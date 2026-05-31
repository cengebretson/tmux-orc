package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const Filename = "orc.yaml"

type Repo struct {
	Name    string `yaml:"name"`
	Path    string `yaml:"path"`
	Purpose string `yaml:"purpose"`
}

type Settings struct {
	DefaultWorkflow string `yaml:"default_workflow"`
	AutoArchive     bool   `yaml:"auto_archive"`
}

type Config struct {
	Repos    []Repo   `yaml:"repos"`
	Settings Settings `yaml:"settings"`
}

// DefaultWorkflow returns the configured default workflow name, falling back to "default".
func (c *Config) DefaultWorkflow() string {
	if c.Settings.DefaultWorkflow != "" {
		return c.Settings.DefaultWorkflow
	}
	return "default"
}

// Load reads orc.yaml from the workspace root.
// Returns an empty Config (not an error) if the file does not exist.
func Load(root string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(root, Filename))
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", Filename, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", Filename, err)
	}
	return &cfg, nil
}
