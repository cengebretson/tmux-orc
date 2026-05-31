package workers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Worker struct {
	ID      string `yaml:"id"`
	Name    string `yaml:"name"`
	Product string `yaml:"product"` // claude | codex | cursor
	Kind    string `yaml:"kind"`
	Model   string `yaml:"model"`

	ReasoningEffort string `yaml:"reasoning_effort"`
	Thinking        string `yaml:"thinking"`
	CostTier        string `yaml:"cost_tier"`

	Workflows         []string `yaml:"workflows"`
	Stages            []string `yaml:"stages"`
	DefaultTmuxWindow string   `yaml:"default_tmux_window"`
	LaunchMode        string   `yaml:"launch_mode"`
}

// Load parses all worker markdown files in the given directory.
func Load(workersDir string) ([]*Worker, error) {
	entries, err := filepath.Glob(filepath.Join(workersDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("scanning workers/: %w", err)
	}

	var workers []*Worker
	for _, path := range entries {
		if filepath.Base(path) == "_template.md" {
			continue
		}
		w, err := parseWorkerFile(path)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		workers = append(workers, w)
	}
	return workers, nil
}

// Match returns workers that support the given workflow.
func Match(workers []*Worker, workflow string) []*Worker {
	var matched []*Worker
	for _, w := range workers {
		if supportsWorkflow(w, workflow) {
			matched = append(matched, w)
		}
	}
	return matched
}

// FindByID returns the worker with the given ID, or nil.
func FindByID(workers []*Worker, id string) *Worker {
	for _, w := range workers {
		if w.ID == id {
			return w
		}
	}
	return nil
}

// Preferred returns the worker whose ID matches the owner, or nil.
// Deprecated: use FindByID. Kept for fallback match paths.
func Preferred(workers []*Worker, ownerID string) *Worker {
	return FindByID(workers, ownerID)
}

// LaunchCommand renders the launch command string for display.
func LaunchCommand(w *Worker, workspaceRoot, cwd, prompt string) string {
	args := LaunchArgs(w, workspaceRoot, cwd, prompt)
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t\n") {
			parts[i] = fmt.Sprintf("%q", a)
		} else {
			parts[i] = a
		}
	}
	return strings.Join(parts, " ")
}

// LaunchArgs returns the argv slice for executing a worker's launch command.
// workspaceRoot is always included so agents start with full context.
// cwd is where repo commands should run (the worktree).
// prompt is the instruction string.
func LaunchArgs(w *Worker, workspaceRoot, cwd, prompt string) []string {
	switch strings.ToLower(w.Product) {
	case "codex":
		model := w.Model
		if model == "" {
			model = "default"
		}
		return []string{"codex", "--model", model, "--cd", cwd, prompt}
	case "cursor":
		return []string{"cursor", cwd}
	default: // claude
		return []string{"claude", "--add-dir", workspaceRoot, prompt}
	}
}

// parseWorkerFile reads a markdown file and extracts YAML frontmatter.
func parseWorkerFile(path string) (*Worker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fm, err := extractFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("no valid frontmatter: %w", err)
	}

	var w Worker
	if err := yaml.Unmarshal([]byte(fm), &w); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	// fall back to filename stem as id if not set
	if w.ID == "" {
		base := filepath.Base(path)
		w.ID = strings.TrimSuffix(base, filepath.Ext(base))
	}

	return &w, nil
}

func extractFrontmatter(content string) (string, error) {
	if !strings.HasPrefix(content, "---") {
		return "", fmt.Errorf("missing opening ---")
	}
	rest := content[3:]
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return "", fmt.Errorf("missing closing ---")
	}
	return strings.TrimSpace(rest[:end]), nil
}

func supportsWorkflow(w *Worker, stage string) bool {
	if len(w.Workflows) == 0 && len(w.Stages) == 0 {
		return true // no restriction
	}
	for _, wf := range w.Workflows {
		if wf == stage {
			return true
		}
	}
	for _, s := range w.Stages {
		if s == stage {
			return true
		}
	}
	return false
}

