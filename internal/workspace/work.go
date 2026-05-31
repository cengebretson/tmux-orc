package workspace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cengebretson/orc/internal/workflow"
	"gopkg.in/yaml.v3"
)

type WorkOptions struct {
	Root     string
	Ticket   string // e.g. FLYWL-123
	Slug     string // optional override suffix, e.g. "add-user-export"
	Workflow string // pipeline name, defaults to "default"
}

type WorkResult struct {
	FeatureDir string
	Slug       string
}

func Work(opts WorkOptions) (*WorkResult, error) {
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, fmt.Errorf("resolving workspace path: %w", err)
	}

	ticket := strings.ToUpper(strings.TrimSpace(opts.Ticket))
	if ticket == "" {
		return nil, fmt.Errorf("ticket ID is required")
	}

	slug := buildSlug(ticket, opts.Slug)
	featureDir := filepath.Join(root, "features", slug)

	// check for any existing feature with this ticket prefix (e.g. FLYWL-123-some-slug)
	if existing, err := findExistingFeature(root, ticket); err == nil {
		return nil, fmt.Errorf("feature for %q already exists: features/%s\nUse `orc next %s` to continue working on it", ticket, existing, ticket)
	}

	// also guard the exact path in case the slug matches directly
	if _, err := os.Stat(featureDir); err == nil {
		return nil, fmt.Errorf("feature %q already exists\nUse `orc next %s` to continue working on it", slug, ticket)
	}

	templateDir := filepath.Join(root, "features", "_template")
	if _, err := os.Stat(templateDir); err != nil {
		return nil, fmt.Errorf("features/_template not found — run `orc init` first")
	}

	if err := copyDir(templateDir, featureDir); err != nil {
		return nil, fmt.Errorf("creating feature folder: %w", err)
	}

	workflowName := opts.Workflow
	if workflowName == "" {
		workflowName = "default"
	}
	workflowCfg, _ := workflow.Load(root)
	firstStage := "intake"
	if stages := workflowCfg.StageNames(workflowName); len(stages) > 0 {
		firstStage = stages[0]
	}

	if err := writeStateYAML(featureDir, ticket, slug, workflowName, firstStage); err != nil {
		return nil, fmt.Errorf("writing STATE.yaml: %w", err)
	}

	return &WorkResult{FeatureDir: featureDir, Slug: slug}, nil
}

// findExistingFeature returns the name of any feature folder that starts with ticket.
func findExistingFeature(root, ticket string) (string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "features"))
	if err != nil {
		return "", err
	}
	upper := strings.ToUpper(ticket)
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "_template" {
			continue
		}
		name := strings.ToUpper(e.Name())
		if name == upper || strings.HasPrefix(name, upper+"-") {
			return e.Name(), nil
		}
	}
	return "", fmt.Errorf("not found")
}

func buildSlug(ticket, suffix string) string {
	if suffix == "" {
		return ticket
	}
	// normalise suffix: lowercase, spaces to hyphens
	suffix = strings.ToLower(strings.TrimSpace(suffix))
	suffix = strings.ReplaceAll(suffix, " ", "-")
	return ticket + "-" + suffix
}

// writeStateYAML stamps STATE.yaml with real ticket/slug values.
func writeStateYAML(featureDir, ticket, slug, workflowName, firstStage string) error {
	type stateStage struct {
		Owner string `yaml:"owner"`
		Name  string `yaml:"name"`
	}
	type stateNextAction struct {
		Worker string `yaml:"worker"`
		Prompt string `yaml:"prompt"`
		CWD    string `yaml:"cwd"`
	}
	type stateHistory struct {
		At     string `yaml:"at"`
		Stage  string `yaml:"stage"`
		Owner  string `yaml:"owner"`
		Result string `yaml:"result"`
	}
	type stateFile struct {
		Ticket   string `yaml:"ticket"`
		Slug     string `yaml:"slug"`
		Status   string `yaml:"status"`
		Workflow string `yaml:"workflow,omitempty"`

		Stage stateStage `yaml:"stage"`

		NextAction stateNextAction `yaml:"next_action"`

		History []stateHistory `yaml:"history"`
	}

	s := stateFile{
		Ticket:   ticket,
		Slug:     slug,
		Status:   "pending",
		Workflow: workflowName,
		Stage: stateStage{
			Owner: "agent",
			Name:  firstStage,
		},
		NextAction: stateNextAction{
			Worker: firstStage,
			Prompt: fmt.Sprintf("Load ticket %s and populate TICKET.md, SPEC.md, and PLAN.md. Update STATE.yaml when complete.", ticket),
			CWD:    ".",
		},
		History: []stateHistory{
			{
				At:    time.Now().Format(time.RFC3339),
				Stage: firstStage,
				Owner: "agent",
				Result:   "feature context created by orc work",
			},
		},
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(featureDir, "STATE.yaml"), data, 0644)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
