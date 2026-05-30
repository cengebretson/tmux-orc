package workflow

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the machine-readable frontmatter from a WORKFLOW.md file.
type Config struct {
	// NextWorkflow is the workflow name to transition to after this one completes.
	NextWorkflow string `yaml:"next_workflow"`
	// NextStage is the stage name within NextWorkflow to advance to.
	NextStage string `yaml:"next_stage"`
	// Advance controls how the transition happens.
	// "auto"   — agent calls orc advance when the stage is done.
	// "manual" — agent calls orc wait; a human approves before advancing.
	Advance string `yaml:"advance"`
	// Worker is the default worker ID assigned to this workflow.
	// Overridden by stage.owner in STATE.yaml or the --worker flag on orc next.
	Worker string `yaml:"worker"`
}

// Load reads the YAML frontmatter from workflows/<name>/WORKFLOW.md.
// Returns an empty Config (no error) if the file has no frontmatter.
func Load(workflowsDir, name string) (*Config, error) {
	path := filepath.Join(workflowsDir, name, "WORKFLOW.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return &Config{}, nil
	}

	content := strings.TrimSpace(string(data))
	if !strings.HasPrefix(content, "---") {
		return &Config{}, nil
	}

	// Extract the block between the first and second "---"
	parts := strings.SplitN(content[3:], "---", 2)
	if len(parts) < 2 {
		return &Config{}, nil
	}

	var c Config
	if err := yaml.Unmarshal([]byte(parts[0]), &c); err != nil {
		return &Config{}, nil
	}
	return &c, nil
}
