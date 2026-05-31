package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

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

// WorkflowDef is a named sequence of stages.
type WorkflowDef struct {
	Stages []StageDef `yaml:"stages"`
}

// StageDef is one step in a workflow.
type StageDef struct {
	Name    string `yaml:"name"`
	Worker  string `yaml:"worker"`
	Advance string `yaml:"advance"`
}

// RepairStageDef defines a repair loop stage.
type RepairStageDef struct {
	Repairs    string `yaml:"repairs"`
	Worker     string `yaml:"worker"`
	Advance    string `yaml:"advance"`
	MaxRetries int    `yaml:"max_retries"`
}

type Config struct {
	Repos        []Repo                    `yaml:"repos"`
	Settings     Settings                  `yaml:"settings"`
	Workflows    map[string]WorkflowDef    `yaml:"workflows"`
	RepairStages map[string]RepairStageDef `yaml:"repair_stages"`
}

// DefaultWorkflow returns the configured default workflow name, falling back to "default".
func (c *Config) DefaultWorkflow() string {
	if c.Settings.DefaultWorkflow != "" {
		return c.Settings.DefaultWorkflow
	}
	return "default"
}

// Names returns all workflow names, sorted.
func (c *Config) Names() []string {
	names := make([]string, 0, len(c.Workflows))
	for k := range c.Workflows {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Stages returns the ordered StageDefs for the named workflow.
func (c *Config) Stages(name string) []StageDef {
	return c.Workflows[name].Stages
}

// StageNames returns just the stage names for the named workflow.
func (c *Config) StageNames(name string) []string {
	stages := c.Workflows[name].Stages
	names := make([]string, len(stages))
	for i, s := range stages {
		names[i] = s.Name
	}
	return names
}

// NextStage returns the stage that follows current in the named workflow.
// Returns "" if current is the last stage or not found.
func (c *Config) NextStage(workflowName, current string) string {
	stages := c.Workflows[workflowName].Stages
	for i, s := range stages {
		if s.Name == current && i+1 < len(stages) {
			return stages[i+1].Name
		}
	}
	return ""
}

// StageConfig returns the StageDef for a named stage in a named workflow.
// Also checks repair stages if not found in the workflow.
func (c *Config) StageConfig(workflowName, stageName string) (StageDef, bool) {
	for _, s := range c.Workflows[workflowName].Stages {
		if s.Name == stageName {
			return s, true
		}
	}
	if rs, ok := c.RepairStages[stageName]; ok {
		return StageDef{Name: stageName, Worker: rs.Worker, Advance: rs.Advance}, true
	}
	return StageDef{}, false
}

// IsRepairStage returns true if the named stage is a repair stage.
func (c *Config) IsRepairStage(name string) bool {
	_, ok := c.RepairStages[name]
	return ok
}

// RepairStage returns the RepairStageDef for the given name, if it exists.
func (c *Config) RepairStage(name string) (RepairStageDef, bool) {
	rs, ok := c.RepairStages[name]
	return rs, ok
}

// Load reads orc.yaml from the workspace root.
// Returns an empty Config (not an error) if the file does not exist.
func Load(root string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(root, Filename))
	if os.IsNotExist(err) {
		return &Config{
			Workflows:    map[string]WorkflowDef{},
			RepairStages: map[string]RepairStageDef{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", Filename, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", Filename, err)
	}
	if cfg.Workflows == nil {
		cfg.Workflows = map[string]WorkflowDef{}
	}
	if cfg.RepairStages == nil {
		cfg.RepairStages = map[string]RepairStageDef{}
	}
	return &cfg, nil
}
