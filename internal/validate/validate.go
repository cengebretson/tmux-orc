package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cengebretson/orc/internal/config"
	"github.com/cengebretson/orc/internal/state"
	"github.com/cengebretson/orc/internal/tmux"
	"github.com/cengebretson/orc/internal/workers"
)

type Status int

const (
	OK      Status = iota
	Warning        // non-blocking issue
	Fail           // blocking problem
)

func (s Status) String() string {
	switch s {
	case OK:
		return "✓"
	case Warning:
		return "⚠"
	default:
		return "✗"
	}
}

type Check struct {
	Name   string
	Status Status
	Detail string
}

type Report struct {
	Ticket     string
	FeatureDir string
	Checks     []Check
}

func (r *Report) OK() bool {
	for _, c := range r.Checks {
		if c.Status == Fail {
			return false
		}
	}
	return true
}

func ok(name string) Check      { return Check{Name: name, Status: OK} }
func okd(name, d string) Check  { return Check{Name: name, Status: OK, Detail: d} }
func warn(name, d string) Check { return Check{Name: name, Status: Warning, Detail: d} }
func fail(name, d string) Check { return Check{Name: name, Status: Fail, Detail: d} }

// Run validates a ticket's STATE.yaml against the workspace.
func Run(root, featureDir string) *Report {
	r := &Report{FeatureDir: featureDir}

	s, err := state.Load(featureDir)
	if err != nil {
		r.Checks = append(r.Checks, fail("STATE.yaml", fmt.Sprintf("cannot load: %v", err)))
		return r
	}
	r.Ticket = s.Ticket
	r.Checks = append(r.Checks, ok("STATE.yaml"))

	// Load orc.yaml — fail early if unreadable.
	cfg, err := config.Load(root)
	if err != nil {
		r.Checks = append(r.Checks, fail("orc.yaml", fmt.Sprintf("cannot load: %v", err)))
		return r
	}
	r.Checks = append(r.Checks, ok("orc.yaml"))

	// Resolve workflow name and verify it exists in orc.yaml.
	pname := s.Workflow
	if pname == "" {
		pname = cfg.DefaultWorkflow()
	}
	if _, ok := cfg.Workflows[pname]; !ok {
		known := cfg.Names()
		detail := fmt.Sprintf("%q not found in orc.yaml", pname)
		if len(known) > 0 {
			detail += fmt.Sprintf(" (available: %s)", strings.Join(known, ", "))
		}
		r.Checks = append(r.Checks, fail("workflow", detail))
		return r
	}
	r.Checks = append(r.Checks, okd("workflow", pname))
	stageNames := cfg.StageNames(pname)

	// Current stage exists in the workflow.
	stageName := s.Stage.Name
	stageInWorkflow := false
	for _, sn := range stageNames {
		if sn == stageName {
			stageInWorkflow = true
			break
		}
	}
	// Also check repair stages.
	if !stageInWorkflow {
		if _, ok := cfg.RepairStages[stageName]; ok {
			stageInWorkflow = true
		}
	}
	if !stageInWorkflow && len(stageNames) > 0 {
		r.Checks = append(r.Checks, fail("stage in workflow", fmt.Sprintf("%q not found in %q pipeline", stageName, pname)))
	} else if stageInWorkflow {
		r.Checks = append(r.Checks, okd("stage in workflow", stageName))
	}

	// Stage markdown file exists.
	stageMD := filepath.Join(root, "stages", stageName+".md")
	if _, err := os.Stat(stageMD); err != nil {
		r.Checks = append(r.Checks, fail("stage file", fmt.Sprintf("stages/%s.md missing", stageName)))
	} else {
		r.Checks = append(r.Checks, okd("stage file", fmt.Sprintf("stages/%s.md", stageName)))
	}

	// Worker exists.
	sc, _ := cfg.StageConfig(pname, stageName)
	workerID := s.Stage.Owner
	if workerID == "" {
		workerID = sc.Worker
	}
	if workerID != "" {
		allWorkers, _ := workers.Load(filepath.Join(root, "workers"))
		if workers.FindByID(allWorkers, workerID) == nil {
			r.Checks = append(r.Checks, fail("worker", fmt.Sprintf("%q not found in workers/", workerID)))
		} else {
			r.Checks = append(r.Checks, okd("worker", workerID))
		}
	} else {
		r.Checks = append(r.Checks, warn("worker", "no worker assigned for this stage"))
	}

	// Repo worktrees exist.
	if len(s.Repos) > 0 {
		var missing []string
		for name, repo := range s.Repos {
			if repo.Worktree == "" {
				continue
			}
			p := repo.Worktree
			if !filepath.IsAbs(p) {
				p = filepath.Join(root, p)
			}
			if _, err := os.Stat(p); err != nil {
				missing = append(missing, fmt.Sprintf("%s (%s)", name, repo.Worktree))
			}
		}
		if len(missing) > 0 {
			for _, m := range missing {
				r.Checks = append(r.Checks, fail("worktree", "not found: "+m))
			}
		} else {
			r.Checks = append(r.Checks, ok("worktrees"))
		}
	}

	// Stage output folder exists (warn if missing — agent may not have written it yet).
	stageOutputDir := filepath.Join(featureDir, stageName)
	if _, err := os.Stat(stageOutputDir); err != nil {
		r.Checks = append(r.Checks, warn("stage outputs", fmt.Sprintf("%s/ not yet written", stageName)))
	} else {
		entries, _ := os.ReadDir(stageOutputDir)
		r.Checks = append(r.Checks, okd("stage outputs", fmt.Sprintf("%s/ (%d file(s))", stageName, len(entries))))
	}

	// Tmux session alive (if configured).
	if s.Runtime.Tmux != nil {
		session := s.Runtime.Tmux.Session
		if tmux.Available() {
			if tmux.SessionExists(session) {
				r.Checks = append(r.Checks, okd("tmux", fmt.Sprintf("session %q active", session)))
			} else {
				r.Checks = append(r.Checks, warn("tmux", fmt.Sprintf("session %q configured but not running", session)))
			}
		} else {
			r.Checks = append(r.Checks, warn("tmux", "configured but tmux not available"))
		}
	}

	return r
}

// Print renders the validation report to stdout.
func Print(r *Report) {
	fmt.Printf("Ticket: %s\n\n", r.Ticket)
	for _, c := range r.Checks {
		if c.Detail != "" {
			fmt.Printf("  %s  %-20s %s\n", c.Status, c.Name, c.Detail)
		} else {
			fmt.Printf("  %s  %s\n", c.Status, c.Name)
		}
	}
	fmt.Println()
	if r.OK() {
		fmt.Println("All checks passed.")
	} else {
		fmt.Println("Some checks failed — ticket may not advance cleanly.")
	}
}
