package runner

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cengebretson/orc/internal/config"
	"github.com/cengebretson/orc/internal/state"
	"github.com/cengebretson/orc/internal/workers"
)

// Plan is the fully resolved next action for a ticket.
type Plan struct {
	Ticket         string
	Workflow       string
	Stage          string
	Worker         *workers.Worker
	WorkerReason   string // "flag override" | "stage owner" | "workflow default"
	Prompt         string
	LaunchCommand  string   // shell string for --dry display
	LaunchArgv     []string // for exec.Command
	CWD            string
	EndInstruction string // the orc advance/wait command the agent should run
}

// Compute resolves the next action plan for a ticket.
// workerOverride is the --worker flag value; pass "" to use the default resolution order.
func Compute(root, featureDir, workerOverride string) (*Plan, error) {
	s, err := state.Load(featureDir)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load(root)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	allWorkers, err := workers.Load(filepath.Join(root, "workers"))
	if err != nil {
		return nil, fmt.Errorf("loading workers: %w", err)
	}

	workflow, err := resolveWorkflow(cfg, s.Workflow)
	if err != nil {
		return nil, err
	}
	stageCfg, _ := cfg.StageConfig(workflow, s.Stage.Name)
	nextStage := cfg.NextStage(workflow, s.Stage.Name)

	worker, reason, err := resolveWorker(allWorkers, workerOverride, s.Stage.Owner, stageCfg.Worker, s.Stage.Name)
	if err != nil {
		return nil, err
	}

	cwd := s.ResolveCWD(root, featureDir)
	prompt := buildPrompt(s, nextStage, stageCfg.Advance)

	plan := &Plan{
		Ticket:         s.Ticket,
		Workflow:       workflow,
		Stage:          s.Stage.Name,
		Worker:         worker,
		WorkerReason:   reason,
		Prompt:         prompt,
		LaunchCommand:  workers.LaunchCommand(worker, root, cwd, prompt),
		LaunchArgv:     workers.LaunchArgs(worker, root, cwd, prompt),
		CWD:            cwd,
		EndInstruction: endInstruction(s.Ticket, nextStage, stageCfg.Advance),
	}
	return plan, nil
}

// ResolveWorkflow returns the ticket's workflow name, using the workspace default if unset.
// Returns an error if the resolved workflow is not defined in orc.yaml.
// Exported so callers that already have a config can avoid a second load.
func ResolveWorkflow(cfg *config.Config, ticketWorkflow string) (string, error) {
	return resolveWorkflow(cfg, ticketWorkflow)
}

func resolveWorkflow(cfg *config.Config, ticketWorkflow string) (string, error) {
	name := ticketWorkflow
	if name == "" {
		name = cfg.DefaultWorkflow()
	}
	if name == "" {
		known := cfg.Names()
		if len(known) > 0 {
			return "", fmt.Errorf("no default_workflow set in orc.yaml (available: %s)", strings.Join(known, ", "))
		}
		return "", fmt.Errorf("no default_workflow set in orc.yaml")
	}
	if _, ok := cfg.Workflows[name]; !ok {
		known := cfg.Names()
		if len(known) > 0 {
			return "", fmt.Errorf("workflow %q not found in orc.yaml (available: %s)", name, strings.Join(known, ", "))
		}
		return "", fmt.Errorf("workflow %q not found in orc.yaml", name)
	}
	return name, nil
}

func resolveWorker(allWorkers []*workers.Worker, flagOverride, stageOwner, configWorker, stageName string) (*workers.Worker, string, error) {
	if flagOverride != "" {
		if w := workers.FindByID(allWorkers, flagOverride); w != nil {
			return w, "flag override", nil
		}
	}
	if stageOwner != "" {
		if w := workers.FindByID(allWorkers, stageOwner); w != nil {
			return w, "stage owner", nil
		}
	}
	if configWorker != "" {
		if w := workers.FindByID(allWorkers, configWorker); w != nil {
			return w, "workflow default", nil
		}
		return nil, "", fmt.Errorf("worker %q assigned to stage %q in orc.yaml not found in workers/", configWorker, stageName)
	}
	return nil, "", fmt.Errorf("no worker assigned for stage %q — set worker: in orc.yaml", stageName)
}

func buildPrompt(s *state.State, nextStage, advanceMode string) string {
	prompt := s.NextAction.Prompt
	if prompt == "" {
		prompt = fmt.Sprintf(
			"Continue %s — stage: %s\n\nFeature context: features/%s/STATE.yaml\nStage: stages/%s.md",
			s.Ticket, s.Stage.Name, s.Slug, s.Stage.Name,
		)
	}

	preamble := fmt.Sprintf(
		"Before starting: read AGENTS.md and ORC.md. Run `orc start %s` to mark in_progress.\n\n",
		s.Ticket,
	)

	return preamble + prompt + endInstruction(s.Ticket, nextStage, advanceMode)
}

func endInstruction(ticket, nextStage, advanceMode string) string {
	if nextStage == "" {
		return ""
	}
	if advanceMode == "manual" {
		return fmt.Sprintf(
			"\n\nWhen this stage is complete, run:\n  orc wait %s \"<summary — human will review before advancing to %s>\"",
			ticket, nextStage,
		)
	}
	return fmt.Sprintf(
		"\n\nWhen this stage is complete, run:\n  orc advance %s --owner <worker-id> --result \"<summary>\"",
		ticket,
	)
}
