package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func timeNow() string {
	return time.Now().Format(time.RFC3339)
}

const Filename = "STATE.yaml"

type State struct {
	Ticket string `yaml:"ticket"`
	Slug   string `yaml:"slug"`
	Status string `yaml:"status"`

	Stage Stage `yaml:"stage"`

	Repos map[string]Repo `yaml:"repos"`

	Inputs  IOSet `yaml:"inputs"`
	Outputs IOSet `yaml:"outputs"`

	NextAction NextAction `yaml:"next_action"`

	History []HistoryEntry `yaml:"history"`
}

type Stage struct {
	Owner    string `yaml:"owner"`
	Workflow string `yaml:"workflow"`
}

type Repo struct {
	Main     string `yaml:"main"`
	Worktree string `yaml:"worktree"`
	Branch   string `yaml:"branch"`
}

type IOSet struct {
	Ready     []string `yaml:"ready"`
	Required  []string `yaml:"required"`
	Completed []string `yaml:"completed"`
}

type NextAction struct {
	Worker string `yaml:"worker"`
	Prompt string `yaml:"prompt"`
	CWD    string `yaml:"cwd"`
}

type HistoryEntry struct {
	At     string `yaml:"at"`
	Stage  string `yaml:"stage"`
	Owner  string `yaml:"owner"`
	Result string `yaml:"result"`
}

// Load reads STATE.yaml from the given feature directory.
func Load(featureDir string) (*State, error) {
	path := filepath.Join(featureDir, Filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return &s, nil
}

// Start marks the feature as in_progress and records a history entry.
func Start(featureDir string) error {
	path := filepath.Join(featureDir, Filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	s.History = append(s.History, HistoryEntry{
		At:     timeNow(),
		Stage:  s.Stage.Workflow,
		Owner:  s.Stage.Owner,
		Result: "started",
	})
	s.Status = "in_progress"

	out, err := yaml.Marshal(&s)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}

// WaitForHuman marks the feature as waiting_for_human and records a history entry.
func WaitForHuman(featureDir, reason string) error {
	path := filepath.Join(featureDir, Filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	s.History = append(s.History, HistoryEntry{
		At:     timeNow(),
		Stage:  s.Stage.Workflow,
		Owner:  s.Stage.Owner,
		Result: "waiting_for_human — " + reason,
	})

	s.Status = "waiting_for_human"
	s.NextAction.Worker = "human"
	s.NextAction.Prompt = reason

	out, err := yaml.Marshal(&s)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}

// Block marks the feature as blocked with a reason and records a history entry.
func Block(featureDir, reason string) error {
	path := filepath.Join(featureDir, Filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	s.History = append(s.History, HistoryEntry{
		At:     timeNow(),
		Stage:  s.Stage.Workflow,
		Owner:  s.Stage.Owner,
		Result: "blocked — " + reason,
	})

	s.Status = "blocked"
	s.NextAction.Worker = "human"
	s.NextAction.Prompt = reason

	out, err := yaml.Marshal(&s)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}

// Advance moves the feature to the next workflow, records a history entry, and saves STATE.yaml.
func Advance(featureDir, workflow, owner, result string) error {
	path := filepath.Join(featureDir, Filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	s.History = append(s.History, HistoryEntry{
		At:     timeNow(),
		Stage:  s.Stage.Workflow,
		Owner:  s.Stage.Owner,
		Result: result,
	})

	if workflow != "" {
		s.Stage.Workflow = workflow
	}
	if owner != "" {
		s.Stage.Owner = owner
	}
	s.Status = "ready"
	s.NextAction = NextAction{}

	out, err := yaml.Marshal(&s)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}

// SetStatus updates only the status field in STATE.yaml, preserving all other content.
func SetStatus(featureDir, status string) error {
	path := filepath.Join(featureDir, Filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	s.Status = status

	out, err := yaml.Marshal(&s)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}

// FindFeatureDir locates the feature directory for the given slug or ticket ID.
// Supports full slug match or prefix match on ticket ID (e.g. "FLYWL-123").
func FindFeatureDir(workspaceRoot, query string) (string, error) {
	featuresDir := filepath.Join(workspaceRoot, "features")

	entries, err := os.ReadDir(featuresDir)
	if err != nil {
		return "", fmt.Errorf("reading features/: %w", err)
	}

	query = strings.ToUpper(strings.TrimSpace(query))

	var matches []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "_template" {
			continue
		}
		name := e.Name()
		upper := strings.ToUpper(name)
		if upper == query || strings.HasPrefix(upper, query) {
			matches = append(matches, filepath.Join(featuresDir, name))
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no feature found matching %q — create one with `orc work %s`", query, query)
	case 1:
		return matches[0], nil
	default:
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = filepath.Base(m)
		}
		return "", fmt.Errorf("ambiguous slug %q matches multiple features:\n  %s\nUse the full slug", query, strings.Join(names, "\n  "))
	}
}

// ResolveCWD returns an absolute path for the next action cwd,
// resolving relative paths against the feature directory.
func (s *State) ResolveCWD(workspaceRoot, featureDir string) string {
	cwd := s.NextAction.CWD
	if cwd == "" || cwd == "." {
		return workspaceRoot
	}
	if filepath.IsAbs(cwd) {
		return cwd
	}
	return filepath.Clean(filepath.Join(featureDir, cwd))
}
