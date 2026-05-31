package health

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cengebretson/orc/internal/config"
	"github.com/cengebretson/orc/internal/workflow"
)

type Status int

const (
	OK      Status = iota
	Missing        // expected but not found
	Empty          // exists but has no content worth noting
)

func (s Status) String() string {
	switch s {
	case OK:
		return "✓"
	case Empty:
		return "⚠"
	default:
		return "✗"
	}
}

type Result struct {
	Name    string
	Status  Status
	Detail  string
}

type Report struct {
	Root    string
	Results []Result
}

func (r *Report) OK() bool {
	for _, res := range r.Results {
		if res.Status == Missing {
			return false
		}
	}
	return true
}

// Run checks the workspace filesystem state at root.
func Run(root string) *Report {
	report := &Report{Root: root}

	// root-level docs
	for _, f := range []string{"AGENTS.md", "TOOLS.md", "RULES.md", "ROUTER.md"} {
		report.Results = append(report.Results, checkFile(root, f))
	}

	// setup completion
	report.Results = append(report.Results, checkSetup(root))

	// features/
	report.Results = append(report.Results, checkFeatures(root))

	// workers/
	report.Results = append(report.Results, checkDirWithCount(root, "workers", "*.md", "worker"))

	// orc.yaml
	report.Results = append(report.Results, checkOrcConfig(root))

	// workflows.yaml + stages/
	report.Results = append(report.Results, checkWorkflowsFile(root))
	report.Results = append(report.Results, checkDirWithCount(root, "stages", "*.md", "stage"))

	// optional dirs — note presence but don't fail if missing
	report.Results = append(report.Results, checkOptionalDir(root, "worktrees"))

	return report
}

// Print renders the report to stdout.
func Print(r *Report) {
	fmt.Printf("Workspace: %s\n\n", r.Root)
	for _, res := range r.Results {
		if res.Detail != "" {
			fmt.Printf("  %s  %-20s %s\n", res.Status, res.Name, res.Detail)
		} else {
			fmt.Printf("  %s  %s\n", res.Status, res.Name)
		}
	}
}

func checkSetup(root string) Result {
	path := filepath.Join(root, "SETUP.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{Name: "SETUP.md", Status: Missing, Detail: "missing — run `orc init`"}
	}
	content := string(data)

	shared := strings.Contains(content, "shared: complete")
	claude := strings.Contains(content, "claude: complete")
	codex := strings.Contains(content, "codex:  complete")

	if shared && (claude || codex) {
		var done []string
		if claude {
			done = append(done, "claude")
		}
		if codex {
			done = append(done, "codex")
		}
		return Result{Name: "SETUP.md", Status: OK, Detail: "complete (" + strings.Join(done, ", ") + ")"}
	}

	var pending []string
	if !shared {
		pending = append(pending, "shared")
	}
	if !claude {
		pending = append(pending, "claude")
	}
	if !codex {
		pending = append(pending, "codex")
	}
	return Result{Name: "SETUP.md", Status: Empty, Detail: "pending: " + strings.Join(pending, ", ") + " — run an agent on SETUP.md"}
}

func checkFile(root, name string) Result {
	path := filepath.Join(root, name)
	if _, err := os.Stat(path); err == nil {
		return Result{Name: name, Status: OK}
	}
	return Result{Name: name, Status: Missing, Detail: "missing — run `orc init`"}
}

func checkDir(root, name string) Result {
	path := filepath.Join(root, name)
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return Result{Name: name + "/", Status: OK}
	}
	return Result{Name: name + "/", Status: Missing, Detail: "missing"}
}

func checkFeatures(root string) Result {
	path := filepath.Join(root, "features")
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return Result{Name: "features/", Status: Missing, Detail: "missing — run `orc init`"}
	}
	entries, _ := os.ReadDir(path)
	var active, archived int
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "_template" {
			continue
		}
		if e.Name() == "_archive" {
			archiveEntries, _ := os.ReadDir(filepath.Join(path, "_archive"))
			for _, a := range archiveEntries {
				if a.IsDir() {
					archived++
				}
			}
			continue
		}
		active++
	}

	if active == 0 && archived == 0 {
		return Result{Name: "features/", Status: Empty, Detail: "no features yet — start one with `orc work <ticket>`"}
	}

	detail := fmt.Sprintf("%d active", active)
	if archived > 0 {
		detail += fmt.Sprintf(", %d archived", archived)
	}
	return Result{Name: "features/", Status: OK, Detail: detail}
}

func checkOptionalDir(root, name string) Result {
	path := filepath.Join(root, name)
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return Result{Name: name + "/", Status: OK}
	}
	return Result{Name: name + "/", Status: Empty, Detail: "not created yet"}
}

func checkOrcConfig(root string) Result {
	cfg, err := config.Load(root)
	if err != nil {
		return Result{Name: config.Filename, Status: Missing, Detail: "missing — run `orc init`"}
	}
	if len(cfg.Repos) == 0 {
		return Result{Name: config.Filename, Status: Empty, Detail: "no repos defined — edit orc.yaml to add repos"}
	}
	names := make([]string, len(cfg.Repos))
	for i, r := range cfg.Repos {
		names[i] = r.Name
	}
	return Result{Name: config.Filename, Status: OK, Detail: fmt.Sprintf("%d repo(s): %s", len(names), strings.Join(names, ", "))}
}

func checkWorkflowsFile(root string) Result {
	cfg, err := workflow.Load(root)
	if err != nil {
		return Result{Name: "workflows.yaml", Status: Missing, Detail: "missing — run `orc init`"}
	}
	names := cfg.Names()
	if len(names) == 0 {
		return Result{Name: "workflows.yaml", Status: Empty, Detail: "no workflows defined"}
	}
	return Result{Name: "workflows.yaml", Status: OK, Detail: fmt.Sprintf("%d workflow(s): %s", len(names), strings.Join(names, ", "))}
}

func checkDirWithCount(root, dir, pattern, label string) Result {
	path := filepath.Join(root, dir)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return Result{Name: dir + "/", Status: Missing, Detail: "missing — run `orc init`"}
	}

	matches, _ := filepath.Glob(filepath.Join(path, pattern))

	switch len(matches) {
	case 0:
		return Result{Name: dir + "/", Status: Empty, Detail: fmt.Sprintf("no %ss defined", label)}
	case 1:
		return Result{Name: dir + "/", Status: OK, Detail: fmt.Sprintf("1 %s", label)}
	default:
		return Result{Name: dir + "/", Status: OK, Detail: fmt.Sprintf("%d %ss", len(matches), label)}
	}
}
