package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cengebretson/orc/internal/config"
	"github.com/cengebretson/orc/internal/health"
	"github.com/cengebretson/orc/internal/resume"
	"github.com/cengebretson/orc/internal/runner"
	"github.com/cengebretson/orc/internal/state"
	"github.com/cengebretson/orc/internal/tmux"
	"github.com/cengebretson/orc/internal/tui"
	"github.com/cengebretson/orc/internal/validate"
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
	Short: "Scaffold a new orc workspace — asks questions interactively when run in a terminal",
	RunE:  runInit,
}

var (
	initWorkspace         string
	initWithSampleWorkers bool
	initDryRun            bool
	initForce             bool
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
	workTmux      bool
	workNext      bool
	workWorkflow  string
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
	advanceStage     string
)

var archiveCmd = &cobra.Command{
	Use:   "archive <ticket>",
	Short: "Archive a completed feature — removes worktrees and moves folder to features/_archive/",
	Args:  cobra.ExactArgs(1),
	RunE:  runArchive,
}

var archiveWorkspace string

var attachCmd = &cobra.Command{
	Use:   "attach <ticket>",
	Short: "Attach to the tmux session for a ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runAttach,
}

var attachWorkspace string

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive dashboard",
	Args:  cobra.NoArgs,
	RunE:  runTui,
}

var tuiWorkspace string

var validateCmd = &cobra.Command{
	Use:   "validate <ticket>",
	Short: "Validate a ticket's state against the workspace — checks workflow, stage, worker, and worktrees",
	Args:  cobra.ExactArgs(1),
	RunE:  runValidate,
}

var validateWorkspace string

var resumeCmd = &cobra.Command{
	Use:   "resume <ticket>",
	Short: "Generate a recovery prompt for a stuck or interrupted ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runResume,
}

var resumeWorkspace string

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
	initCmd.Flags().StringVar(&initWorkspace, "workspace", ".", "Workspace root directory (skips the interactive path prompt)")
	initCmd.Flags().BoolVar(&initWithSampleWorkers, "with-sample-workers", false, "Include sample worker files (skips the interactive prompt)")
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
	workCmd.Flags().BoolVar(&workTmux, "tmux", false, "Enable tmux session for this ticket — session created automatically on first orc next")
	workCmd.Flags().BoolVar(&workNext, "next", false, "Immediately launch the first stage after creating the feature")
	workCmd.Flags().StringVar(&workWorkflow, "workflow", "", "Workflow to use (default: settings.default_workflow in orc.yaml, or \"default\")")
	showCmd.Flags().StringVar(&showWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output as JSON")
	startCmd.Flags().StringVar(&startWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	waitCmd.Flags().StringVar(&waitWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	blockCmd.Flags().StringVar(&blockWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	advanceCmd.Flags().StringVar(&advanceWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	advanceCmd.Flags().StringVar(&advanceOwner, "owner", "", "Worker or role that owns the new stage")
	advanceCmd.Flags().StringVar(&advanceResult, "result", "", "Summary of what was accomplished in the previous stage")
	advanceCmd.Flags().StringVar(&advanceStage, "stage", "", "New stage name (required when crossing workflow boundaries)")
	archiveCmd.Flags().StringVar(&archiveWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	attachCmd.Flags().StringVar(&attachWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	tuiCmd.Flags().StringVar(&tuiWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	validateCmd.Flags().StringVar(&validateWorkspace, "workspace", ".", "Workspace root (default: current directory)")
	resumeCmd.Flags().StringVar(&resumeWorkspace, "workspace", ".", "Workspace root (default: current directory)")

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
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(helpAllCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Print(banner)

	interactive := isTTY()

	// Workspace path — prompt if not explicitly set and running interactively.
	if !cmd.Flags().Changed("workspace") && interactive {
		cwd, _ := os.Getwd()
		ans := promptLine(fmt.Sprintf("Workspace path [%s]: ", cwd))
		if ans == "" {
			initWorkspace = cwd
		} else {
			initWorkspace = ans
		}
	}

	// Sample workers — prompt if not explicitly set and running interactively.
	if !cmd.Flags().Changed("with-sample-workers") && interactive {
		ans := promptLine("Include sample workers? [y/N]: ")
		ans = strings.ToLower(strings.TrimSpace(ans))
		initWithSampleWorkers = ans == "y" || ans == "yes"
	}

	opts := workspace.InitOptions{
		Root:              initWorkspace,
		WithSampleWorkers: initWithSampleWorkers,
		DryRun:            initDryRun,
		Force:             initForce,
	}

	return workspace.Init(opts)
}

// isTTY returns true when stdin is an interactive terminal.
func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// promptLine prints the prompt and reads one line from stdin.
func promptLine(prompt string) string {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
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
		plan, err := runner.Compute(root, featureDir, nextWorker)
		if err != nil {
			return err
		}
		return printJSON(map[string]any{
			"ticket":   plan.Ticket,
			"status":   s.Status,
			"workflow": plan.Workflow,
			"stage":    plan.Stage,
			"owner":    s.Stage.Owner,
			"cwd":      plan.CWD,
			"prompt":   plan.Prompt,
			"worker":   plan.Worker.ID,
			"product":  plan.Worker.Engine,
			"model":    plan.Worker.Model,
			"launch":   plan.LaunchCommand,
		})
	}

	fmt.Printf("Ticket:   %s\n", s.Ticket)
	fmt.Printf("Status:   %s\n", s.Status)
	fmt.Printf("Workflow: %s\n", resolveWorkflow(root, s.Workflow))
	fmt.Printf("Stage:    %s\n", s.Stage.Name)
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

	plan, err := runner.Compute(root, featureDir, nextWorker)
	if err != nil {
		return err
	}

	if nextDry {
		fmt.Printf("Worker:  %s  (%s)\n", plan.Worker.Name, plan.WorkerReason)
		fmt.Printf("Engine: %s\n", plan.Worker.Engine)
		if plan.Worker.Model != "" {
			fmt.Printf("Model:   %s\n", plan.Worker.Model)
		}
		fmt.Printf("cwd:     %s\n", plan.CWD)
		fmt.Println()
		fmt.Println("Would run:")
		fmt.Printf("  %s\n", plan.LaunchCommand)
		fmt.Println()
		fmt.Printf("Override worker: orc next %s --worker <worker-id>\n", s.Ticket)
		return nil
	}

	return launchPlan(root, featureDir, s, plan)
}

func launchPlan(root, featureDir string, s *state.State, plan *runner.Plan) error {
	// Auto-tmux: create a session if available, fall through to foreground on failure.
	if tmux.Available() {
		session := s.Slug
		window := s.Stage.Name

		if s.Runtime.Tmux == nil {
			stages := stageNamesForTicket(root, s)
			if err := tmux.CreateSession(session, featureDir, stages); err != nil {
				fmt.Printf("tmux session create failed (%v) — running in foreground\n", err)
				goto runForeground
			}
			if err := state.SetRuntime(featureDir, session); err != nil {
				fmt.Printf("warning: could not write runtime to STATE.yaml: %v\n", err)
			}
		} else {
			session = s.Runtime.Tmux.Session
			if !tmux.SessionExists(session) {
				stages := stageNamesForTicket(root, s)
				if err := tmux.CreateSession(session, featureDir, stages); err != nil {
					fmt.Printf("tmux session recreate failed (%v) — running in foreground\n", err)
					goto runForeground
				}
			}
		}

		fmt.Printf("Sending to tmux session %s:%s...\n", session, window)
		if err := tmux.SendCommand(session, window, featureDir, plan.CWD, plan.LaunchArgv); err != nil {
			fmt.Printf("tmux send failed (%v) — running in foreground\n", err)
		} else {
			fmt.Printf("Agent launched in background.\n")
			fmt.Printf("Attach:  %s\n", tmux.AttachHint(session, window))
			return nil
		}
	}
runForeground:
	fmt.Printf("Launching %s (%s)...\n", plan.Worker.Name, plan.Worker.Engine)
	c := exec.Command(plan.LaunchArgv[0], plan.LaunchArgv[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Dir = plan.CWD
	return c.Run()
}

func runWork(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(workWorkspace)
	if err != nil {
		return err
	}

	result, err := workspace.Work(workspace.WorkOptions{
		Root:     root,
		Ticket:   args[0],
		Slug:     workSlug,
		Workflow: workWorkflow,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Created: features/%s/\n\n", result.Slug)

	cfg, _ := config.Load(root)

	useTmux := workTmux
	if !useTmux && cfg != nil {
		useTmux = cfg.Settings.AutoTmux
	}
	if useTmux {
		if err := state.SetRuntime(result.FeatureDir, result.Slug); err != nil {
			fmt.Printf("warning: could not write tmux runtime to STATE.yaml: %v\n", err)
		}
	}

	plan, err := runner.Compute(root, result.FeatureDir, "")
	if err != nil {
		return err
	}

	useNext := workNext || (cfg != nil && cfg.Settings.AutoNext)
	if useNext {
		s, err := state.Load(result.FeatureDir)
		if err != nil {
			return err
		}
		return launchPlan(root, result.FeatureDir, s, plan)
	}

	printDryRun(plan, result.Slug)
	return nil
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
			rowPname := resolveWorkflow(root, s.Workflow)
			rows = append(rows, row{
				ticket:   s.Ticket,
				status:   s.Status,
				workflow: rowPname + " · " + s.Stage.Name,
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
			fmt.Printf("Session:  %s\n", tmux.AttachHint(session, s.Stage.Name))
		} else {
			fmt.Printf("Session:  %s  (not running — run `orc next %s` to restart)\n", session, s.Ticket)
		}
	}

	fmt.Println()
	fmt.Println("Stage")
	pname := resolveWorkflow(root, s.Workflow)
	fmt.Printf("  Workflow:  %s\n", pname)
	fmt.Printf("  Name:      %s\n", s.Stage.Name)
	fmt.Printf("  Owner:     %s\n", s.Stage.Owner)
	if wfCfg, err := config.Load(root); err == nil {
		if next := wfCfg.NextStage(pname, s.Stage.Name); next != "" {
			sc, _ := wfCfg.StageConfig(pname, next)
			advance := sc.Advance
			if advance == "" {
				advance = "auto"
			}
			fmt.Printf("  Next:      %s  (%s)\n", next, advance)
		}
	}

	if len(s.Repos) > 0 {
		fmt.Println()
		fmt.Println("Repos")
		for name, r := range s.Repos {
			fmt.Printf("  %s\n", name)
			if r.Main != "" {
				fmt.Printf("    main:     %s\n", r.Main)
			}
			if r.Worktree != "" {
				fmt.Printf("    worktree: %s\n", r.Worktree)
				fmt.Printf("    branch:   %s\n", r.Branch)
			}
		}
	}

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
	switch s.Status {
	case "waiting_for_human", "blocked":
		label := "Waiting"
		if s.Status == "blocked" {
			label = "Blocked"
		}
		reason := ""
		if len(s.History) > 0 {
			reason = s.History[len(s.History)-1].Result
		}
		if reason == "" {
			reason = s.NextAction.Prompt
		}
		fmt.Printf("  %s: %s\n", label, reason)
		fmt.Println("  Run `orc next` after resolving to continue.")
	default:
		allWorkers, _ := workers.Load(filepath.Join(root, "workers"))
		wfCfg, _ := config.Load(root)
		sc, _ := wfCfg.StageConfig(pname, s.Stage.Name)
		workerID := s.Stage.Owner
		if workerID == "" {
			workerID = sc.Worker
		}
		if workerID != "" {
			preferred := workers.FindByID(allWorkers, workerID)
			if preferred != nil {
				fmt.Printf("  Worker:  %s (%s)\n", preferred.Name, preferred.Engine)
				if preferred.Model != "" {
					fmt.Printf("  Model:   %s\n", preferred.Model)
				}
			} else {
				fmt.Printf("  Worker:  %s (not found in workers/)\n", workerID)
			}
		} else {
			fmt.Println("  Worker:  none assigned — set worker: in orc.yaml")
		}
		fmt.Println("  Run `orc next` to launch.")
	}

	if len(s.History) > 0 {
		fmt.Println()
		fmt.Println("History")
		for _, h := range s.History {
			fmt.Printf("  %s  %-24s  %-20s  %s\n", h.At, h.Stage, h.Owner, h.Result)
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

	startPname := resolveWorkflow(root, s.Workflow)
	fmt.Printf("Ticket:   %s\n", s.Ticket)
	fmt.Printf("Status:   in_progress\n")
	fmt.Printf("Workflow: %s\n", startPname)
	fmt.Printf("Stage:    %s\n", s.Stage.Name)
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

	s, err := state.Load(featureDir)
	if err != nil {
		return err
	}
	if err := state.ValidateRepos(s, root); err != nil {
		return err
	}

	reason := strings.Join(args[1:], " ")
	if err := state.WaitForHuman(featureDir, reason); err != nil {
		return err
	}

	fmt.Printf("Ticket:  %s\n", s.Ticket)
	fmt.Printf("Status:  waiting_for_human\n")
	fmt.Printf("Needs:   %s\n", reason)
	fmt.Printf("\nRun `orc advance %s` to continue once resolved.\n", s.Ticket)
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
	fmt.Printf("\nRun `orc advance %s` to unblock when resolved.\n", s.Ticket)
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

	// Guard: archived tickets cannot be advanced.
	if s.Status == "archived" {
		return fmt.Errorf("ticket %s is archived — cannot advance", s.Ticket)
	}

	if err := state.ValidateRepos(s, root); err != nil {
		return err
	}

	workflowCfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	pname := resolveWorkflow(root, s.Workflow)
	prevStage := s.Stage.Name

	// If --stage is given manually, validate it exists in the workflow or repair stages.
	if advanceStage != "" {
		if _, ok := workflowCfg.StageConfig(pname, advanceStage); !ok {
			return fmt.Errorf("stage %q not found in workflow %q — check orc.yaml", advanceStage, pname)
		}
	}

	// Auto-advance to next stage in the feature's pipeline.
	nextStage := advanceStage
	if nextStage == "" {
		nextStage = workflowCfg.NextStage(pname, prevStage)
	}

	// Guard: manual gate — agent must call orc wait, not orc advance.
	if nextStage != "" {
		if sc, ok := workflowCfg.StageConfig(pname, prevStage); ok && sc.Advance == "manual" {
			return fmt.Errorf(
				"stage %q has advance: manual — use `orc wait %s \"<reason>\"` so a human can review before continuing",
				prevStage, s.Ticket,
			)
		}
	}

	// Guard: repair stage max_retries.
	if rs, ok := workflowCfg.RepairStage(prevStage); ok && rs.MaxRetries > 0 {
		count := s.StageCounts[prevStage]
		if count >= rs.MaxRetries {
			return fmt.Errorf(
				"repair stage %q has reached max_retries (%d) — resolve manually or use `orc block %s`",
				prevStage, rs.MaxRetries, s.Ticket,
			)
		}
	}

	result := advanceResult
	if result == "" {
		if nextStage != "" && nextStage != prevStage {
			result = fmt.Sprintf("advanced from %s to %s", prevStage, nextStage)
		} else {
			result = fmt.Sprintf("completed %s", prevStage)
		}
	}

	if err := state.Advance(featureDir, nextStage, advanceOwner, result); err != nil {
		return err
	}

	fmt.Printf("Ticket:   %s\n", s.Ticket)
	if nextStage != "" && nextStage != prevStage {
		fmt.Printf("Stage:    %s → %s\n", prevStage, nextStage)
	} else {
		fmt.Printf("Stage:    %s  (unchanged)\n", prevStage)
	}
	if advanceOwner != "" {
		fmt.Printf("Owner:    %s\n", advanceOwner)
	}
	// Auto-archive if the pipeline is complete and the workspace opts in.
	if nextStage == "" {
		cfg, _ := config.Load(root)
		if cfg != nil && cfg.Settings.AutoArchive {
			fmt.Println()
			s, err = state.Load(featureDir)
			if err != nil {
				return err
			}
			return archiveFeature(root, featureDir, s)
		}
	}

	fmt.Printf("\nRun `orc next %s` to launch the next worker.\n", s.Ticket)
	fmt.Println()

	plan, err := runner.Compute(root, featureDir, "")
	if err != nil {
		return err
	}
	printDryRun(plan, s.Ticket)
	return nil
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

	return archiveFeature(root, featureDir, s)
}

func archiveFeature(root, featureDir string, s *state.State) error {
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

func runValidate(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(validateWorkspace)
	if err != nil {
		return err
	}

	featureDir, err := state.FindFeatureDir(root, args[0])
	if err != nil {
		return err
	}

	report := validate.Run(root, featureDir)
	validate.Print(report)
	if !report.OK() {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func runResume(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(resumeWorkspace)
	if err != nil {
		return err
	}

	featureDir, err := state.FindFeatureDir(root, args[0])
	if err != nil {
		return err
	}

	ctx, err := resume.Build(root, featureDir)
	if err != nil {
		return err
	}

	fmt.Println(ctx.Prompt)
	return nil
}

func runTui(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(tuiWorkspace)
	if err != nil {
		return err
	}
	return tui.Run(root)
}

func runAttach(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(attachWorkspace)
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
		return fmt.Errorf("no tmux session for %s — run `orc next %s` to start one", s.Ticket, s.Ticket)
	}

	return tmux.Attach(s.Slug + ":" + s.Stage.Name)
}

// resolveWorkflow returns the ticket's workflow name for display purposes.
func resolveWorkflow(root, ticketWorkflow string) string {
	if ticketWorkflow != "" {
		return ticketWorkflow
	}
	cfg, _ := config.Load(root)
	if cfg != nil && cfg.Settings.DefaultWorkflow != "" {
		return cfg.Settings.DefaultWorkflow
	}
	return ""
}

// stageNamesForTicket returns the ordered stage names for the ticket's workflow pipeline.
func stageNamesForTicket(root string, s *state.State) []string {
	workflowCfg, _ := config.Load(root)
	return workflowCfg.StageNames(resolveWorkflow(root, s.Workflow))
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

func printDryRun(plan *runner.Plan, ticket string) {
	fmt.Printf("Worker:  %s  (%s)\n", plan.Worker.Name, plan.WorkerReason)
	fmt.Printf("Engine: %s\n", plan.Worker.Engine)
	if plan.Worker.Model != "" {
		fmt.Printf("Model:   %s\n", plan.Worker.Model)
	}
	fmt.Printf("cwd:     %s\n", plan.CWD)
	fmt.Println()
	fmt.Println("Would run:")
	fmt.Printf("  %s\n", plan.LaunchCommand)
	fmt.Println()
	fmt.Printf("Override worker: orc next %s --worker <worker-id>\n", ticket)
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
