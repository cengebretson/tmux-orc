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

const SchemaVersion = 1

type State struct {
	SchemaVersion int    `yaml:"schema_version,omitempty"`
	Ticket        string `yaml:"ticket"`
	Slug          string `yaml:"slug"`
	Status        string `yaml:"status"`
	Workflow      string `yaml:"workflow,omitempty"`

	Stage Stage `yaml:"stage"`

	StageCounts map[string]int `yaml:"stage_counts,omitempty"`

	Runtime Runtime `yaml:"runtime,omitempty"`

	Repos map[string]Repo `yaml:"repos"`

	Inputs  IOSet `yaml:"inputs"`
	Outputs IOSet `yaml:"outputs"`

	NextAction NextAction `yaml:"next_action"`

	History []HistoryEntry `yaml:"history"`
}

type Runtime struct {
	Tmux *TmuxRuntime `yaml:"tmux,omitempty"`
}

type TmuxRuntime struct {
	Session string `yaml:"session"`
}

type Stage struct {
	Owner string `yaml:"owner"`
	Name  string `yaml:"name"`
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
	At    string `yaml:"at"`
	Stage string `yaml:"stage"`
	Owner string `yaml:"owner"`
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
		Stage:   s.Stage.Name,
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
		Stage:   s.Stage.Name,
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
		Stage:   s.Stage.Name,
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
func Advance(featureDir, stageName, owner, result string) error {
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
		Stage:   s.Stage.Name,
		Owner:  s.Stage.Owner,
		Result: result,
	})

	if stageName != "" {
		s.Stage.Name = stageName
		if s.StageCounts == nil {
			s.StageCounts = map[string]int{}
		}
		s.StageCounts[stageName]++
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
// SetRuntime writes the runtime.tmux session name to STATE.yaml.
func SetRuntime(featureDir, tmuxSession string) error {
	path := filepath.Join(featureDir, Filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	s.Runtime.Tmux = &TmuxRuntime{Session: tmuxSession}
	out, err := yaml.Marshal(&s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

// ClearRuntime removes the runtime block from STATE.yaml.
func ClearRuntime(featureDir string) error {
	path := filepath.Join(featureDir, Filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	s.Runtime = Runtime{}
	out, err := yaml.Marshal(&s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

// ValidateRepos checks that repo fields in STATE.yaml are internally consistent
// and point to paths that make sense within the workspace. Returns a non-nil
// error listing all problems found. Only validates fields that are set — a repo
// with no worktree recorded is not an error.
func ValidateRepos(s *State, workspaceRoot string) error {
	worktreesRoot := filepath.Join(workspaceRoot, "worktrees")
	var errs []string

	for name, r := range s.Repos {
		// main must exist if set
		if r.Main != "" {
			if _, err := os.Stat(r.Main); os.IsNotExist(err) {
				errs = append(errs, fmt.Sprintf("repos.%s.main %q does not exist", name, r.Main))
			}
		}

		if r.Worktree != "" {
			// worktree must be under worktrees/ in the workspace
			abs := r.Worktree
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(workspaceRoot, abs)
			}
			rel, err := filepath.Rel(worktreesRoot, abs)
			if err != nil || strings.HasPrefix(rel, "..") {
				errs = append(errs, fmt.Sprintf("repos.%s.worktree %q is not under worktrees/ in the workspace", name, r.Worktree))
			}

			// branch must be non-empty when a worktree is recorded
			if r.Branch == "" {
				errs = append(errs, fmt.Sprintf("repos.%s.branch is empty but worktree is set", name))
			}
		}
	}

	// next_action.cwd must be under a recorded worktree when any worktree is set
	hasWorktrees := false
	for _, r := range s.Repos {
		if r.Worktree != "" {
			hasWorktrees = true
			break
		}
	}
	if hasWorktrees && s.NextAction.CWD != "" {
		cwd := s.NextAction.CWD
		if !filepath.IsAbs(cwd) {
			cwd = filepath.Join(workspaceRoot, cwd)
		}
		matched := false
		for _, r := range s.Repos {
			if r.Worktree == "" {
				continue
			}
			wt := r.Worktree
			if !filepath.IsAbs(wt) {
				wt = filepath.Join(workspaceRoot, wt)
			}
			rel, err := filepath.Rel(wt, cwd)
			if err == nil && !strings.HasPrefix(rel, "..") {
				matched = true
				break
			}
		}
		if !matched {
			errs = append(errs, fmt.Sprintf("next_action.cwd %q does not match any recorded worktree", s.NextAction.CWD))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("STATE.yaml repo validation failed:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

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
