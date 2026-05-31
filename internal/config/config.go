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

type Config struct {
	Repos []Repo `yaml:"repos"`
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
