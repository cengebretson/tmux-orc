package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cengebretson/orc/internal/health"
	"github.com/cengebretson/orc/internal/state"
	"github.com/cengebretson/orc/internal/tmux"
	"github.com/cengebretson/orc/internal/workflow"
	"github.com/cengebretson/orc/internal/workers"
	"github.com/cengebretson/orc/internal/workspace"
	"github.com/spf13/cobra"
)

const banner = `
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⣿⣿⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣤⣶⣧⣄⣉⣉⣠⣼⣶⣤⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⢰⣿⣿⣿⣿⡿⣿⣿⣿⣿⢿⣿⣿⣿⣿⡆⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⣼⣤⣤⣈⠙⠳⢄⣉⣋⡡⠞⠋⣁⣤⣤⣧⠀⠀⠀⠀⠀⠀⠀
⠀⢲⣶⣤⣄⡀⢀⣿⣄⠙⠿⣿⣦⣤⡿⢿⣤⣴⣿⠿⠋⣠⣿⠀⢀⣠⣤⣶⡖⠀
⠀⠀⠙⣿⠛⠇⢸⣿⣿⡟⠀⡄⢉⠉⢀⡀⠉⡉⢠⠀⢻⣿⣿⡇⠸⠛⣿⠋⠀⠀
⠀⠀⠀⠘⣷⠀⢸⡏⠻⣿⣤⣤⠂⣠⣿⣿⣄⠑⣤⣤⣿⠟⢹⡇⠀⣾⠃⠀⠀⠀
⠀⠀⠀⠀⠘⠀⢸⣿⡀⢀⠙⠻⢦⣌⣉⣉⣡⡴⠟⠋⡀⢀⣿⡇⠀⠃⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⢸⣿⣧⠈⠛⠂⠀⠉⠛⠛⠉⠀⠐⠛⠁⣼⣿⡇⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠸⣏⠀⣤⡶⠖⠛⠋⠉⠉⠙⠛⠲⢶⣤⠀⣹⠇⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⢹⣿⣶⣿⣿⣿⣿⣿⣿⣶⣿⡏⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠉⠉⠛⠛⠛⠛⠉⠉⠉⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀

orc · workspace orchestrator
`

var rootCmd = &cobra.Command{
	Use:   "orc",
	Short: "orc — agentic workspace orchestrator",
	Long:  banner,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a new orc workspace",
	RunE:  runInit,
}

var (
	initWorkspace       string
	initWithSampleWorkers bool
	initDryRun          bool
	initForce           bool
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check workspace and integration health",
	RunE:  runHealth,
}

var healthWorkspace string

var nextCmd = &cobra.Command{
	Use:   "next <ticket>",
	Short: "Launch the next agent for a ticket — use --dry to preview without running",
	Args:  cobra.ExactArgs(1),
	RunE:  runNext,
}

var (
	nextWorkspace string
	nextJSON      bool
	nextDry       bool
	nextWorker    string
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show all features and their current stage",
	RunE:  runStatus,
}

var (
	statusWorkspace string
	statusJSON      bool
)

var workCmd = &cobra.Command{
	Use:   "work <ticket>",
	Short: "Start work on a ticket — creates the feature folder and STATE.yaml",
	Args:  cobra.ExactArgs(1),
	RunE:  runWork,
}

var (
	workWorkspace string
	workSlug      string
)

var showCmd = &cobra.Command{
	Use:   "show <slug>",
	Short: "Show full details for a feature ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

var (
	showWorkspace string
	showJSON      bool
)

var startCmd = &cobra.Command{
	Use:    "start <ticket>",
	Short:  "Mark a ticket as in_progress",
	Args:   cobra.ExactArgs(1),
	RunE:   runStart,
	Hidden: true,
}

var startWorkspace string

var waitCmd = &cobra.Command{
	Use:    "wait <ticket> <reason>",
	Short:  "Mark a ticket as waiting for human input or approval",
	Args:   cobra.MinimumNArgs(2),
	RunE:   runWait,
	Hidden: true,
}

var waitWorkspace string

var blockCmd = &cobra.Command{
	Use:    "block <ticket> <reason>",
	Short:  "Mark a ticket as blocked with a reason",
	Args:   cobra.MinimumNArgs(2),
	RunE:   runBlock,
	Hidden: true,
}

var blockWorkspace string

var advanceCmd = &cobra.Command{
	Use:    "advance <ticket>",
	Short:  "Mark current workflow complete and move to the next — writes STATE.yaml",
	Args:   cobra.ExactArgs(1),
	RunE:   runAdvance,
	Hidden: true,
}

var (
	advanceWorkspace string
	advanceOwner     string
	advanceResult    string
	advanceWorkflow  string
)

var archiveCmd = &cobra.Command{
	Use:   "archive <ticket>",
	Short: "Archive a completed feature — removes worktrees and moves folder to features/_archive/",
	Args:  cobra.ExactArgs(1),
	RunE:  runArchive,
}

var archiveWorkspace string

var tmuxCmd = &cobra.Command{
	Use:   "tmux",
	Short: "Manage tmux sessions for feature tickets",
}

var tmuxCreateCmd = &cobra.Command{
	Use:   "create <ticket>",
	Short: "Create a tmux session for a ticket with workflow windows",
	Args:  cobra.ExactArgs(1),
	RunE:  runTmuxCreate,
}

var tmuxAttachCmd = &cobra.Command{
	Use:   "attach <ticket>",
	Short: "Attach to the tmux session for a ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runTmuxAttach,
}

var tmuxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active tmux sessions",
	Args:  cobra.NoArgs,
	RunE:  runTmuxList,
}

var tmuxKillCmd = &cobra.Command{
	Use:   "kill <ticket>",
	Short: "Kill the tmux session for a ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runTmuxKill,
}

var tmuxWorkspace string

var helpAllCmd = &cobra.Command{
	Use:   "help-all",
	Short: "List all commands, including agent-only hidden commands",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("All orc commands:")
		fmt.Println()
		fmt.Printf("  %-30s  %s\n", "COMMAND", "DESCRIPTION")
		fmt.Printf("  %-30s  %s\n", "-------", "-----------")
		for _, c := range rootCmd.Commands() {
			if c.Name() == "help" || c.Name() == "help-all" {
				continue
			}
			tag := ""
			if c.Hidden {
				tag = "  [agent]"
			}
			fmt.Printf("  %-30s  %s%s\n", c.UseLine(), c.Short, tag)
		}
		fmt.Println()
		fmt.Println("[agent] commands are called by agents, not humans. They are hidden from normal help.")
	},
}

func init() {
	initCmd.Flags().StringVar(&initWorkspace, "workspace", ".", "Workspace root directory (default: current directory)")
	initCmd.Flags().BoolVar(&initWithSampleWorkers, "with-sample-workers", false, "Include sample worker files")
	initCmd.Flags().BoolVar(&initDryRun, "dry-run", false, "Print what would be created without writing files")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing generated files")

	healthCmd.Flags().StringVar(&healthWorkspace, "workspace", ".", "Workspace root to check (default: current directory)")
	nextCmd.Flags().StringVar(&nextWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	nextCmd.Flags().BoolVar(&nextJSON, "json", false, "Output as JSON")
	nextCmd.Flags().BoolVar(&nextDry, "dry", false, "Print the launch command without executing it")
	nextCmd.Flags().StringVar(&nextWorker, "worker", "", "Override the workflow's default worker (worker ID)")
	statusCmd.Flags().StringVar(&statusWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON")
	workCmd.Flags().StringVar(&workWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	workCmd.Flags().StringVar(&workSlug, "slug", "", "Optional slug suffix (e.g. add-user-export → TICKET-123-add-user-export)")
	showCmd.Flags().StringVar(&showWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")
	startCmd.Flags().StringVar(&startWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	waitCmd.Flags().StringVar(&waitWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	blockCmd.Flags().StringVar(&blockWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	advanceCmd.Flags().StringVar(&advanceWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	advanceCmd.Flags().StringVar(&advanceOwner, "owner", "", "Worker or role that owns the new stage")
	advanceCmd.Flags().StringVar(&advanceResult, "result", "", "Summary of what was accomplished in the previous stage")
	advanceCmd.Flags().StringVar(&advanceWorkflow, "workflow", "", "New workflow name (required when crossing workflow boundaries)")
	archiveCmd.Flags().StringVar(&archiveWorkspace, "workspace", ".", "Workspace root (default: current directory)")

	tmuxCmd.PersistentFlags().StringVar(&tmuxWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	tmuxCmd.AddCommand(tmuxCreateCmd)
	tmuxCmd.AddCommand(tmuxAttachCmd)
	tmuxCmd.AddCommand(tmuxListCmd)
	tmuxCmd.AddCommand(tmuxKillCmd)

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(workCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(waitCmd)
	rootCmd.AddCommand(blockCmd)
	rootCmd.AddCommand(advanceCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(tmuxCmd)
	rootCmd.AddCommand(helpAllCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if cmd.Parent() != nil {
		fmt.Print(banner)
	}

	opts := workspace.InitOptions{
		Root:              initWorkspace,
		WithSampleWorkers: initWithSampleWorkers,
		DryRun:            initDryRun,
		Force:             initForce,
	}

	return workspace.Init(opts)
}

func runHealth(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(healthWorkspace)
	if err != nil {
		return err
	}

	report := health.Run(root)
	health.Print(report)
	return nil
}

func runNext(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(nextWorkspace)
	if err != nil {
		return err
	}

	featureDir, err := state.FindFeatureDir(root, args[0])
	if err != nil {
		return err
	}

	s, err := state.Load(featureDir)
	if err != nil {
		return err
	}

	if nextJSON {
		allWorkers, err := workers.Load(filepath.Join(root, "workers"))
		if err != nil {
			return err
		}
		cwd := s.ResolveCWD(root, featureDir)
		wfCfg, _ := workflow.Load(filepath.Join(root, "workflows"), s.Stage.Workflow)
		jsonPrompt := s.NextAction.Prompt
		if jsonPrompt == "" {
			jsonPrompt = fmt.Sprintf("Continue %s — workflow: %s\n\nFeature context: features/%s/STATE.yaml\nWorkflow: workflows/%s/WORKFLOW.md",
				s.Ticket, s.Stage.Workflow, s.Slug, s.Stage.Workflow)
		}
		jsonPreamble := fmt.Sprintf("Before starting: read AGENTS.md and workflows/REQUIREMENTS.md. Run `orc start %s` to mark in_progress.\n\n", s.Ticket)
		jsonPrompt = jsonPreamble + jsonPrompt
		if wfCfg != nil && wfCfg.NextWorkflow != "" {
			if wfCfg.Advance == "auto" {
				jsonPrompt += fmt.Sprintf("\n\nWhen this workflow is complete, run:\n  orc advance %s --workflow %s --owner <worker-id> --result \"<summary>\"",
					s.Ticket, wfCfg.NextWorkflow)
			} else {
				jsonPrompt += fmt.Sprintf("\n\nWhen this workflow is complete, run:\n  orc wait %s \"<summary — human will review before advancing to %s>\"",
					s.Ticket, wfCfg.NextWorkflow)
			}
		}
		// Resolve worker using same priority order as runNextAction.
		var preferred *workers.Worker
		if nextWorker != "" {
			preferred = workers.FindByID(allWorkers, nextWorker)
		}
		if preferred == nil && s.Stage.Owner != "" {
			preferred = workers.FindByID(allWorkers, s.Stage.Owner)
		}
		if preferred == nil && wfCfg != nil && wfCfg.Worker != "" {
			preferred = workers.FindByID(allWorkers, wfCfg.Worker)
		}
		if preferred == nil {
			matched := workers.Match(allWorkers, s.Stage.Workflow)
			if len(matched) > 0 {
				preferred = matched[0]
			}
		}
		out := map[string]any{
			"ticket":   s.Ticket,
			"status":   s.Status,
			"workflow": s.Stage.Workflow,
			"owner":    s.Stage.Owner,
			"cwd":      cwd,
			"prompt":   jsonPrompt,
		}
		if preferred != nil {
			out["worker"] = preferred.ID
			out["product"] = preferred.Product
			out["model"] = preferred.Model
			out["launch"] = workers.LaunchCommand(preferred, root, cwd, jsonPrompt)
		}
		return printJSON(out)
	}

	fmt.Printf("Ticket:   %s\n", s.Ticket)
	fmt.Printf("Status:   %s\n", s.Status)
	fmt.Printf("Workflow: %s\n", s.Stage.Workflow)
	fmt.Printf("Owner:    %s\n", s.Stage.Owner)

	switch s.Status {
	case "pending":
		fmt.Println()
		fmt.Println("Intake has not run yet. Launching intake agent:")
	case "in_progress":
		fmt.Println()
		fmt.Println("⚠ This ticket is already in_progress — an agent session may be active.")
		fmt.Println("  Check for partial work before launching a new session.")
	case "waiting_for_human":
		fmt.Println()
		fmt.Printf("Needs your input: %s\n", s.NextAction.Prompt)
		fmt.Println()
		fmt.Println("Resolve then run `orc next` again to continue:")
		return nil
	case "blocked":
		fmt.Println()
		fmt.Printf("Blocked: %s\n", s.NextAction.Prompt)
		fmt.Println()
		fmt.Println("Resolve the external issue then run `orc next` to continue:")
		return nil
	}
	fmt.Println()

	return runNextAction(root, featureDir, s, nextDry)
}

// runNextAction resolves the recommended worker and either executes or prints the launch command.
// When dry is true, it prints the command without executing (preview mode).
func runNextAction(root, featureDir string, s *state.State, dry bool) error {
	allWorkers, err := workers.Load(filepath.Join(root, "workers"))
	if err != nil {
		return err
	}

	cwd := s.ResolveCWD(root, featureDir)
	prompt := s.NextAction.Prompt
	if prompt == "" {
		prompt = fmt.Sprintf("Continue %s — workflow: %s\n\nFeature context: features/%s/STATE.yaml\nWorkflow: workflows/%s/WORKFLOW.md",
			s.Ticket, s.Stage.Workflow, s.Slug, s.Stage.Workflow)
	}
	preamble := fmt.Sprintf("Before starting: read AGENTS.md and workflows/REQUIREMENTS.md. Run `orc start %s` to mark in_progress.\n\n", s.Ticket)
	prompt = preamble + prompt

	wfCfg, _ := workflow.Load(filepath.Join(root, "workflows"), s.Stage.Workflow)
	if wfCfg.NextWorkflow != "" {
		var suffix string
		if wfCfg.Advance == "auto" {
			suffix = fmt.Sprintf(
				"\n\nWhen this workflow is complete, run:\n  orc advance %s --workflow %s --owner <worker-id> --result \"<summary>\"",
				s.Ticket, wfCfg.NextWorkflow,
			)
		} else {
			suffix = fmt.Sprintf(
				"\n\nWhen this workflow is complete, run:\n  orc wait %s \"<summary — human will review before advancing to %s>\"",
				s.Ticket, wfCfg.NextWorkflow,
			)
		}
		prompt += suffix
	}

	// Resolve worker: --worker flag > stage.owner > workflow default > fallback match
	var worker *workers.Worker
	var matchReason string
	if nextWorker != "" {
		worker = workers.FindByID(allWorkers, nextWorker)
		matchReason = "flag override"
	}
	if worker == nil && s.Stage.Owner != "" {
		worker = workers.FindByID(allWorkers, s.Stage.Owner)
		matchReason = "stage owner"
	}
	if worker == nil && wfCfg.Worker != "" {
		worker = workers.FindByID(allWorkers, wfCfg.Worker)
		matchReason = "workflow default"
	}
	if worker == nil {
		matched := workers.Match(allWorkers, s.Stage.Workflow)
		if len(matched) > 0 {
			worker = matched[0]
			matchReason = "best match for workflow"
		}
	}

	if worker == nil {
		fmt.Printf("No worker found for workflow %q\n", s.Stage.Workflow)
		if wfCfg.Worker != "" {
			fmt.Printf("Workflow default worker %q not found in workers/\n", wfCfg.Worker)
		}
		fmt.Println("Set worker: in the workflow's WORKFLOW.md or add a matching worker file.")
		return nil
	}

	argv := workers.LaunchArgs(worker, root, cwd, prompt)

	// Auto-detect tmux: if a session is recorded in state and is alive, send the command there.
	if !dry && tmux.Available() && s.Runtime.Tmux != nil {
		session := s.Runtime.Tmux.Session
		window := s.Stage.Workflow
		if tmux.SessionExists(session) {
			fmt.Printf("Sending to tmux session %s:%s...\n", session, window)
			if err := tmux.SendCommand(session, window, featureDir, cwd, argv); err != nil {
				fmt.Printf("tmux send failed (%v) — running in foreground\n", err)
			} else {
				fmt.Printf("Agent launched in background.\n")
				fmt.Printf("Attach:  %s\n", tmux.AttachHint(session, window))
				return nil
			}
		} else {
			fmt.Printf("tmux session %q not running — use `orc tmux create %s` to restart, or proceeding in foreground\n", session, s.Ticket)
		}
	}

	if dry {
		fmt.Printf("Worker:  %s  (%s)\n", worker.Name, matchReason)
		fmt.Printf("Product: %s\n", worker.Product)
		if worker.Model != "" {
			fmt.Printf("Model:   %s\n", worker.Model)
		}
		fmt.Printf("cwd:     %s\n", cwd)
		fmt.Println()
		fmt.Println("Would run:")
		fmt.Printf("  %s\n", workers.LaunchCommand(worker, root, cwd, prompt))
		fmt.Println()
		fmt.Printf("Override worker: orc next %s --worker <worker-id>\n", s.Ticket)
		return nil
	}

	fmt.Printf("Launching %s (%s)...\n", worker.Name, worker.Product)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = cwd
	return cmd.Run()
}



func runWork(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(workWorkspace)
	if err != nil {
		return err
	}

	result, err := workspace.Work(workspace.WorkOptions{
		Root:   root,
		Ticket: args[0],
		Slug:   workSlug,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Created: features/%s/\n\n", result.Slug)

	s, err := state.Load(result.FeatureDir)
	if err != nil {
		return err
	}

	return runNextAction(root, result.FeatureDir, s, true)
}

func runStatus(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(statusWorkspace)
	if err != nil {
		return err
	}

	type row struct {
		ticket   string
		status   string
		workflow string
		owner    string
		next     string
		session  string
	}

	// Fetch active tmux sessions once for cross-referencing.
	activeSessions := make(map[string]bool)
	showTmux := tmux.Available()
	if showTmux {
		for _, name := range tmux.ListSessions() {
			activeSessions[name] = true
		}
	}

	collectRows := func(dir string) []row {
		entries, _ := os.ReadDir(dir)
		var rows []row
		for _, e := range entries {
			if !e.IsDir() || e.Name() == "_template" || e.Name() == "_archive" {
				continue
			}
			featureDir := filepath.Join(dir, e.Name())
			s, err := state.Load(featureDir)
			if err != nil {
				rows = append(rows, row{ticket: e.Name(), status: "error", next: err.Error()})
				continue
			}
			next := s.NextAction.Prompt
			if len(next) > 40 {
				next = next[:40] + "…"
			}
			session := "-"
			if s.Runtime.Tmux != nil {
				if activeSessions[s.Runtime.Tmux.Session] {
					session = "✓"
				} else {
					session = "✗" // configured but not running
				}
			}
			rows = append(rows, row{
				ticket:   s.Ticket,
				status:   s.Status,
				workflow: s.Stage.Workflow,
				owner:    s.Stage.Owner,
				next:     next,
				session:  session,
			})
		}
		return rows
	}

	printTable := func(rows []row) {
		if showTmux {
			fmt.Printf("%-16s  %-16s  %-28s  %-20s  %-6s  %s\n", "Ticket", "Status", "Workflow", "Owner", "Tmux", "Next")
			fmt.Printf("%-16s  %-16s  %-28s  %-20s  %-6s  %s\n", "------", "------", "--------", "-----", "----", "----")
			for _, r := range rows {
				fmt.Printf("%-16s  %-16s  %-28s  %-20s  %-6s  %s\n", r.ticket, r.status, r.workflow, r.owner, r.session, r.next)
			}
		} else {
			fmt.Printf("%-16s  %-16s  %-28s  %-20s  %s\n", "Ticket", "Status", "Workflow", "Owner", "Next")
			fmt.Printf("%-16s  %-16s  %-28s  %-20s  %s\n", "------", "------", "--------", "-----", "----")
			for _, r := range rows {
				fmt.Printf("%-16s  %-16s  %-28s  %-20s  %s\n", r.ticket, r.status, r.workflow, r.owner, r.next)
			}
		}
	}

	featuresDir := filepath.Join(root, "features")

	if statusJSON {
		collectStates := func(dir string) []*state.State {
			entries, _ := os.ReadDir(dir)
			var out []*state.State
			for _, e := range entries {
				if !e.IsDir() || e.Name() == "_template" || e.Name() == "_archive" {
					continue
				}
				s, err := state.Load(filepath.Join(dir, e.Name()))
				if err == nil {
					out = append(out, s)
				}
			}
			return out
		}
		return printJSON(map[string]any{
			"active":   collectStates(featuresDir),
			"archived": collectStates(filepath.Join(featuresDir, "_archive")),
		})
	}

	active := collectRows(featuresDir)
	archived := collectRows(filepath.Join(featuresDir, "_archive"))

	if len(active) == 0 && len(archived) == 0 {
		fmt.Println("No features found. Start one with `orc work <ticket>`.")
		return nil
	}

	if len(active) > 0 {
		fmt.Printf("Active (%d)\n\n", len(active))
		printTable(active)
	}

	if len(archived) > 0 {
		if len(active) > 0 {
			fmt.Println()
		}
		fmt.Printf("Archived (%d)\n\n", len(archived))
		printTable(archived)
	}

	return nil
}

func runShow(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(showWorkspace)
	if err != nil {
		return err
	}

	featureDir, err := state.FindFeatureDir(root, args[0])
	if err != nil {
		return err
	}

	s, err := state.Load(featureDir)
	if err != nil {
		return err
	}

	if showJSON {
		return printJSON(s)
	}

	fmt.Printf("Ticket:   %s\n", s.Ticket)
	fmt.Printf("Slug:     %s\n", s.Slug)
	fmt.Printf("Status:   %s\n", s.Status)
	if s.Runtime.Tmux != nil {
		session := s.Runtime.Tmux.Session
		if tmux.Available() && tmux.SessionExists(session) {
			fmt.Printf("Session:  %s\n", tmux.AttachHint(session, s.Stage.Workflow))
		} else {
			fmt.Printf("Session:  %s  (not running — use `orc tmux create %s` to restart)\n", session, s.Ticket)
		}
	}

	fmt.Println()
	fmt.Println("Stage")
	fmt.Printf("  Workflow:  %s\n", s.Stage.Workflow)
	fmt.Printf("  Owner:     %s\n", s.Stage.Owner)

	if len(s.Inputs.Ready)+len(s.Inputs.Required)+len(s.Inputs.Completed) > 0 {
		fmt.Println()
		fmt.Println("Inputs")
		for _, f := range s.Inputs.Ready {
			fmt.Printf("  %s  %s\n", fileCheck(featureDir, f), f)
		}
		for _, f := range s.Inputs.Required {
			fmt.Printf("  %s  %s\n", fileCheck(featureDir, f), f)
		}
		for _, f := range s.Inputs.Completed {
			fmt.Printf("  %s  %s\n", fileCheck(featureDir, f), f)
		}
	}

	if len(s.Outputs.Ready)+len(s.Outputs.Required)+len(s.Outputs.Completed) > 0 {
		fmt.Println()
		fmt.Println("Outputs")
		for _, f := range s.Outputs.Ready {
			fmt.Printf("  %s  %s\n", fileCheck(featureDir, f), f)
		}
		for _, f := range s.Outputs.Required {
			fmt.Printf("  %s  %s\n", fileCheck(featureDir, f), f)
		}
		for _, f := range s.Outputs.Completed {
			fmt.Printf("  %s  %s\n", fileCheck(featureDir, f), f)
		}
	}

	fmt.Println()
	fmt.Println("Next")
	if s.Status == "waiting_for_human" {
		fmt.Printf("  Waiting: %s\n", s.NextAction.Prompt)
		fmt.Println("  Run `orc next` after resolving to continue.")
	} else if s.Status == "blocked" {
		fmt.Printf("  Blocked: %s\n", s.NextAction.Prompt)
		fmt.Println("  Run `orc next` after resolving to continue.")
	} else {
		allWorkers, _ := workers.Load(filepath.Join(root, "workers"))
		matched := workers.Match(allWorkers, s.Stage.Workflow)
		preferred := workers.Preferred(matched, s.Stage.Owner)
		if preferred == nil && len(matched) > 0 {
			preferred = matched[0]
		}
		if preferred != nil {
			fmt.Printf("  Worker:  %s (%s)\n", preferred.Name, preferred.Product)
			if preferred.Model != "" {
				fmt.Printf("  Model:   %s\n", preferred.Model)
			}
		}
		fmt.Println("  Run `orc next` to launch.")
	}

	if len(s.History) > 0 {
		fmt.Println()
		fmt.Println("History")
		for _, h := range s.History {
			fmt.Printf("  %s  %-24s  %-20s  %s\n", h.At, h.Workflow, h.Owner, h.Result)
		}
	}

	return nil
}

func fileCheck(featureDir, relPath string) string {
	if _, err := os.Stat(filepath.Join(featureDir, relPath)); err == nil {
		return "✓"
	}
	return "✗"
}

func runStart(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(startWorkspace)
	if err != nil {
		return err
	}

	featureDir, err := state.FindFeatureDir(root, args[0])
	if err != nil {
		return err
	}

	s, err := state.Load(featureDir)
	if err != nil {
		return err
	}

	if err := state.Start(featureDir); err != nil {
		return err
	}

	fmt.Printf("Ticket:   %s\n", s.Ticket)
	fmt.Printf("Status:   in_progress\n")
	fmt.Printf("Workflow: %s\n", s.Stage.Workflow)
	fmt.Printf("Owner:    %s\n", s.Stage.Owner)
	return nil
}

func runWait(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(waitWorkspace)
	if err != nil {
		return err
	}

	featureDir, err := state.FindFeatureDir(root, args[0])
	if err != nil {
		return err
	}

	reason := strings.Join(args[1:], " ")
	if err := state.WaitForHuman(featureDir, reason); err != nil {
		return err
	}

	s, err := state.Load(featureDir)
	if err != nil {
		return err
	}

	fmt.Printf("Ticket:  %s\n", s.Ticket)
	fmt.Printf("Status:  waiting_for_human\n")
	fmt.Printf("Needs:   %s\n", reason)
	fmt.Printf("\nRun `orc advance %s --workflow <next-workflow>` to continue once resolved.\n", s.Ticket)
	return nil
}

func runBlock(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(blockWorkspace)
	if err != nil {
		return err
	}

	featureDir, err := state.FindFeatureDir(root, args[0])
	if err != nil {
		return err
	}

	reason := strings.Join(args[1:], " ")
	if err := state.Block(featureDir, reason); err != nil {
		return err
	}

	s, err := state.Load(featureDir)
	if err != nil {
		return err
	}

	fmt.Printf("Ticket:  %s\n", s.Ticket)
	fmt.Printf("Status:  blocked\n")
	fmt.Printf("Reason:  %s\n", reason)
	fmt.Printf("\nRun `orc advance %s --workflow <next-workflow>` to unblock when resolved.\n", s.Ticket)
	return nil
}

func runAdvance(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(advanceWorkspace)
	if err != nil {
		return err
	}

	featureDir, err := state.FindFeatureDir(root, args[0])
	if err != nil {
		return err
	}

	s, err := state.Load(featureDir)
	if err != nil {
		return err
	}

	prevWorkflow := s.Stage.Workflow
	result := advanceResult
	if result == "" {
		if advanceWorkflow != "" {
			result = fmt.Sprintf("advanced from %s to %s", prevWorkflow, advanceWorkflow)
		} else {
			result = fmt.Sprintf("completed %s", prevWorkflow)
		}
	}

	if err := state.Advance(featureDir, advanceWorkflow, advanceOwner, result); err != nil {
		return err
	}

	fmt.Printf("Ticket:   %s\n", s.Ticket)
	if advanceWorkflow != "" {
		fmt.Printf("Workflow: %s → %s\n", prevWorkflow, advanceWorkflow)
	} else {
		fmt.Printf("Workflow: %s  (unchanged)\n", prevWorkflow)
	}
	if advanceOwner != "" {
		fmt.Printf("Owner:    %s\n", advanceOwner)
	}
	fmt.Printf("\nRun `orc next %s` to launch the next worker.\n", s.Ticket)
	fmt.Println()

	s, err = state.Load(featureDir)
	if err != nil {
		return err
	}
	return runNextAction(root, featureDir, s, true)
}

func runArchive(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(archiveWorkspace)
	if err != nil {
		return err
	}

	featureDir, err := state.FindFeatureDir(root, args[0])
	if err != nil {
		return err
	}

	s, err := state.Load(featureDir)
	if err != nil {
		return err
	}

	// remove git worktrees for any write repos
	for name, repo := range s.Repos {
		if repo.Worktree == "" {
			continue
		}
		worktreePath := filepath.Join(root, repo.Worktree)
		fmt.Printf("Removing worktree: %s\n", repo.Worktree)
		if err := removeWorktree(repo.Main, worktreePath); err != nil {
			fmt.Printf("  warning: %v\n", err)
			fmt.Printf("  you may need to run: git -C %q worktree remove %q --force\n", repo.Main, worktreePath)
		} else {
			fmt.Printf("  removed %s (%s)\n", name, repo.Branch)
		}
	}

	// stamp status: archived in STATE.yaml before moving
	if err := state.SetStatus(featureDir, "archived"); err != nil {
		return fmt.Errorf("updating status: %w", err)
	}

	// move to features/_archive/
	archiveDir := filepath.Join(root, "features", "_archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("creating _archive dir: %w", err)
	}

	slug := filepath.Base(featureDir)
	dest := filepath.Join(archiveDir, slug)
	if err := os.Rename(featureDir, dest); err != nil {
		return fmt.Errorf("moving feature folder: %w", err)
	}

	// Kill tmux session if one is running for this ticket.
	if tmux.Available() && tmux.SessionExists(s.Slug) {
		if err := tmux.KillSession(s.Slug); err != nil {
			fmt.Printf("warning: could not kill tmux session %s: %v\n", s.Slug, err)
		} else {
			fmt.Printf("Killed tmux session: %s\n", s.Slug)
		}
	}
	// Clear runtime from the archived STATE.yaml regardless of whether the session existed.
	archiveDest := filepath.Join(root, "features", "_archive", filepath.Base(featureDir))
	if err := state.ClearRuntime(archiveDest); err != nil {
		fmt.Printf("warning: could not clear runtime from STATE.yaml: %v\n", err)
	}

	fmt.Printf("Archived: features/_archive/%s/\n", slug)
	return nil
}

func runTmuxCreate(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(tmuxWorkspace)
	if err != nil {
		return err
	}
	if !tmux.Available() {
		return fmt.Errorf("tmux is not installed or not in PATH")
	}

	featureDir, err := state.FindFeatureDir(root, args[0])
	if err != nil {
		return err
	}
	s, err := state.Load(featureDir)
	if err != nil {
		return err
	}

	if tmux.SessionExists(s.Slug) {
		fmt.Printf("Session already exists: %s\n", s.Slug)
		fmt.Printf("Attach:  %s\n", tmux.AttachHint(s.Slug, s.Stage.Workflow))
		return nil
	}

	workflows := listWorkflowNames(root)
	if err := tmux.CreateSession(s.Slug, featureDir, workflows); err != nil {
		return err
	}

	if err := state.SetRuntime(featureDir, s.Slug); err != nil {
		fmt.Printf("warning: could not write runtime to STATE.yaml: %v\n", err)
	}

	fmt.Printf("Session: %s\n", s.Slug)
	fmt.Printf("Windows: %s\n", strings.Join(workflows, ", "))
	fmt.Println()
	fmt.Printf("Attach:  %s\n", tmux.AttachHint(s.Slug, s.Stage.Workflow))
	return nil
}

func runTmuxAttach(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(tmuxWorkspace)
	if err != nil {
		return err
	}
	if !tmux.Available() {
		return fmt.Errorf("tmux is not installed or not in PATH")
	}

	featureDir, err := state.FindFeatureDir(root, args[0])
	if err != nil {
		return err
	}
	s, err := state.Load(featureDir)
	if err != nil {
		return err
	}

	if !tmux.SessionExists(s.Slug) {
		return fmt.Errorf("no tmux session for %s — run `orc tmux create %s` first", s.Ticket, s.Ticket)
	}

	return tmux.Attach(s.Slug + ":" + s.Stage.Workflow)
}

func runTmuxList(cmd *cobra.Command, args []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux is not installed or not in PATH")
	}
	sessions := tmux.ListSessions()
	if len(sessions) == 0 {
		fmt.Println("No active tmux sessions.")
		return nil
	}
	fmt.Println("Active tmux sessions:")
	for _, s := range sessions {
		fmt.Printf("  %s\n", s)
	}
	return nil
}

func runTmuxKill(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(tmuxWorkspace)
	if err != nil {
		return err
	}
	if !tmux.Available() {
		return fmt.Errorf("tmux is not installed or not in PATH")
	}

	featureDir, err := state.FindFeatureDir(root, args[0])
	if err != nil {
		return err
	}
	s, err := state.Load(featureDir)
	if err != nil {
		return err
	}

	if !tmux.SessionExists(s.Slug) {
		return fmt.Errorf("no tmux session for %s", s.Ticket)
	}

	if err := tmux.KillSession(s.Slug); err != nil {
		return err
	}

	if err := state.ClearRuntime(featureDir); err != nil {
		fmt.Printf("warning: could not clear runtime from STATE.yaml: %v\n", err)
	}

	fmt.Printf("Killed session: %s\n", s.Slug)
	return nil
}

// listWorkflowNames returns workflow directory names ordered by the next_workflow
// chain defined in each WORKFLOW.md frontmatter. Workflows not reachable from
// a chain start (e.g. repair branches) are appended at the end.
func listWorkflowNames(root string) []string {
	workflowsDir := filepath.Join(root, "workflows")
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil
	}

	var all []string
	for _, e := range entries {
		if e.IsDir() {
			all = append(all, e.Name())
		}
	}

	// Build next_workflow map and track which names are referenced.
	next := make(map[string]string)
	referenced := make(map[string]bool)
	for _, name := range all {
		cfg, _ := workflow.Load(workflowsDir, name)
		if cfg != nil && cfg.NextWorkflow != "" {
			next[name] = cfg.NextWorkflow
			referenced[cfg.NextWorkflow] = true
		}
	}

	// Starting points: workflows not referenced as anyone's next_workflow.
	var starts []string
	for _, name := range all {
		if !referenced[name] {
			starts = append(starts, name)
		}
	}

	// Walk each chain from its start. Skip starts whose next_workflow target is
	// already visited — those are branch workflows (e.g. pr-repair → pr-open)
	// and will be inserted adjacent to their target in the next pass.
	visited := make(map[string]bool)
	var ordered []string
	var walk func(name string)
	walk = func(name string) {
		if name == "" || visited[name] {
			return
		}
		visited[name] = true
		ordered = append(ordered, name)
		walk(next[name])
	}
	for _, s := range starts {
		if visited[next[s]] {
			continue // branch start — handle in insertion pass
		}
		walk(s)
	}

	// Insert unvisited workflows immediately after the workflow they point to,
	// so pr-repair appears right after pr-open rather than at the end.
	for _, name := range all {
		if visited[name] {
			continue
		}
		target := next[name]
		pos := -1
		for i, n := range ordered {
			if n == target {
				pos = i
				break
			}
		}
		if pos >= 0 {
			tail := make([]string, len(ordered[pos+1:]))
			copy(tail, ordered[pos+1:])
			ordered = append(ordered[:pos+1], append([]string{name}, tail...)...)
		} else {
			ordered = append(ordered, name)
		}
		visited[name] = true
	}

	return ordered
}

func removeWorktree(repoMain, worktreePath string) error {
	out, err := exec.Command("git", "-C", repoMain, "worktree", "remove", worktreePath, "--force").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func withoutWorker(list []*workers.Worker, id string) []*workers.Worker {
	var out []*workers.Worker
	for _, w := range list {
		if w.ID != id {
			out = append(out, w)
		}
	}
	return out
}

func resolveRoot(path string) (string, error) {
	if path == "." {
		return os.Getwd()
	}
	return path, nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
