package doctor

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cengebretson/orc/internal/health"
	"github.com/cengebretson/orc/internal/workers"
)

type Status int

const (
	OK Status = iota
	Warning
	Fail
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
	Group  string
	Name   string
	Status Status
	Detail string
}

type Report struct {
	Root   string
	Checks []Check
}

func (r *Report) OK() bool {
	for _, c := range r.Checks {
		if c.Status == Fail {
			return false
		}
	}
	return true
}

type Options struct {
	LookPath func(string) (string, error)
}

func Run(root string) *Report {
	return RunWithOptions(root, Options{})
}

func RunWithOptions(root string, opts Options) *Report {
	if opts.LookPath == nil {
		opts.LookPath = exec.LookPath
	}

	report := &Report{Root: root}
	appendHealth(report, health.Run(root))
	appendToolChecks(report, root, opts.LookPath)
	return report
}

func Print(r *Report) {
	fmt.Printf("Workspace: %s\n\n", r.Root)
	var currentGroup string
	for _, c := range r.Checks {
		if c.Group != currentGroup {
			currentGroup = c.Group
			if currentGroup != "" {
				fmt.Printf("  %s\n", currentGroup)
			}
		}
		indent := "  "
		if c.Group != "" {
			indent = "    "
		}
		if c.Detail != "" {
			fmt.Printf("%s%s  %-20s %s\n", indent, c.Status, c.Name, c.Detail)
		} else {
			fmt.Printf("%s%s  %s\n", indent, c.Status, c.Name)
		}
	}
}

func appendHealth(report *Report, h *health.Report) {
	for _, r := range h.Results {
		status := OK
		switch r.Status {
		case health.Missing:
			status = Fail
		case health.Empty:
			status = Warning
		}
		group := r.Group
		if group == "" {
			group = "workspace"
		}
		report.Checks = append(report.Checks, Check{
			Group:  group,
			Name:   r.Name,
			Status: status,
			Detail: r.Detail,
		})
	}
}

func appendToolChecks(report *Report, root string, lookPath func(string) (string, error)) {
	report.Checks = append(report.Checks, executableCheck("tools", "tmux", "tmux", lookPath, true))

	engineNames, err := workerEngines(root)
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Group:  "tools",
			Name:   "workers",
			Status: Fail,
			Detail: fmt.Sprintf("cannot load workers/: %v", err),
		})
		return
	}
	for _, engine := range engineNames {
		required := engine != "cursor"
		report.Checks = append(report.Checks, executableCheck("tools", engine, engine, lookPath, !required))
	}
}

func executableCheck(group, name, command string, lookPath func(string) (string, error), optional bool) Check {
	path, err := lookPath(command)
	if err == nil {
		return Check{Group: group, Name: name, Status: OK, Detail: path}
	}
	status := Fail
	detail := "not found in PATH"
	if optional {
		status = Warning
		detail += " (optional)"
	}
	return Check{Group: group, Name: name, Status: status, Detail: detail}
}

func workerEngines(root string) ([]string, error) {
	allWorkers, err := workers.Load(filepath.Join(root, "workers"))
	if err != nil {
		return nil, err
	}
	engines := map[string]bool{}
	for _, w := range allWorkers {
		engine := strings.ToLower(strings.TrimSpace(w.Engine))
		if engine == "" {
			engine = "claude"
		}
		engines[engine] = true
	}
	names := make([]string, 0, len(engines))
	for name := range engines {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}
