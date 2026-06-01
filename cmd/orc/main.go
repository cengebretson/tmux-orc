package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

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

var globalWorkspace string

var rootCmd = &cobra.Command{
	Use:               "orc",
	Short:             "orc — agentic workspace orchestrator",
	Long:              banner,
	CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the orc version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			fmt.Println(info.Main.Version)
		} else {
			fmt.Println("dev")
		}
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a new orc workspace — asks questions interactively when run in a terminal",
	RunE:  runInit,
}

var (
	initWithSampleWorkers bool
	initDryRun            bool
	initForce             bool
)

var healthCmd = &cobra.Command{
	Use:   "health [ticket]",
	Short: "Check workspace health, or validate a ticket's state when a ticket ID is given",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runHealth,
}

var nextCmd = &cobra.Command{
	Use:   "next <ticket>",
	Short: "Launch the next agent for a ticket — use --dry to preview without running",
	Args:  cobra.ExactArgs(1),
	RunE:  runNext,
}

var (
	nextJSON   bool
	nextDry    bool
	nextWorker string
)

var statusCmd = &cobra.Command{
	Use:   "status [ticket]",
	Short: "Show all features and their current stage, or full details for a specific ticket",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runStatus,
}

var statusJSON bool

var workCmd = &cobra.Command{
	Use:   "work <ticket>",
	Short: "Start work on a ticket — creates the feature folder and STATE.yaml",
	Args:  cobra.ExactArgs(1),
	RunE:  runWork,
}

var (
	workSlug     string
	workTmux     bool
	workNext     bool
	workWorkflow string
)

var markCmd = &cobra.Command{
	Use:   "mark <ticket> <start|next|pause|done> [reason]",
	Short: "Update ticket state — start | next [--result] [--stage] [--worker] | pause <reason> | done [--result]",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runMark,

	Hidden: true,
}

var (
	markWorker string
	markResult string
	markStage  string
)

var archiveCmd = &cobra.Command{
	Use:   "archive <ticket>",
	Short: "Archive a completed feature — removes worktrees and moves folder to features/_archive/",
	Args:  cobra.ExactArgs(1),
	RunE:  runArchive,
}

var attachCmd = &cobra.Command{
	Use:   "attach <ticket>",
	Short: "Attach to the tmux session for a ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runAttach,
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive dashboard",
	Args:  cobra.NoArgs,
	RunE:  runTui,
}

var helpAllCmd = &cobra.Command{
	Use:   "help-all",
	Short: "List all commands with human and agent commands separated",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var human, agent []*cobra.Command
		for _, c := range rootCmd.Commands() {
			if c.Name() == "help" || c.Name() == "help-all" || c.Name() == "completion" {
				continue
			}
			if c.Hidden {
				agent = append(agent, c)
			} else {
				human = append(human, c)
			}
		}

		colWidth := func(cmds []*cobra.Command) int {
			w := len("COMMAND")
			for _, c := range cmds {
				if n := len(c.UseLine()); n > w {
					w = n
				}
			}
			return w
		}
		printSection := func(title string, cmds []*cobra.Command) {
			w := colWidth(cmds)
			fmt.Println(title)
			fmt.Println()
			fmt.Printf("  %-*s  %s\n", w, "COMMAND", "DESCRIPTION")
			fmt.Printf("  %-*s  %s\n", w, strings.Repeat("-", w), strings.Repeat("-", len("DESCRIPTION")))
			for _, c := range cmds {
				fmt.Printf("  %-*s  %s\n", w, c.UseLine(), c.Short)
			}
		}

		printSection("Human commands:", human)
		fmt.Println()
		printSection("Agent commands  (called by agents, hidden from orc --help):", agent)
		fmt.Println()
		fmt.Println("Read commands  (human commands agents also use):")
		fmt.Println()
		fmt.Println("  orc status <ticket> --json    read current state as JSON")
		fmt.Println()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&globalWorkspace, "workspace", ".", "Workspace root (default: current directory)")

	initCmd.Flags().BoolVar(&initWithSampleWorkers, "with-sample-workers", false, "Include sample worker files (skips the interactive prompt)")
	initCmd.Flags().BoolVar(&initDryRun, "dry-run", false, "Print what would be created without writing files")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing generated files")

	nextCmd.Flags().BoolVar(&nextJSON, "json", false, "Output as JSON")
	nextCmd.Flags().BoolVar(&nextDry, "dry", false, "Print the launch command without executing it")
	nextCmd.Flags().StringVar(&nextWorker, "worker", "", "Override the workflow's default worker (worker ID)")
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON")
	workCmd.Flags().StringVar(&workSlug, "slug", "", "Optional slug suffix (e.g. add-user-export → TICKET-123-add-user-export)")
	workCmd.Flags().BoolVar(&workTmux, "tmux", false, "Enable tmux session for this ticket — session created automatically on first orc next")
	workCmd.Flags().BoolVar(&workNext, "next", false, "Immediately launch the first stage after creating the feature")
	workCmd.Flags().StringVar(&workWorkflow, "workflow", "", "Workflow to use (default: settings.default_workflow in orc.yaml)")
	markCmd.Flags().StringVar(&markWorker, "worker", "", "Worker ID that owns the new stage (next only)")
	markCmd.Flags().StringVar(&markResult, "result", "", "Summary of what was accomplished (next/done only)")
	markCmd.Flags().StringVar(&markStage, "stage", "", "New stage name (next only — required when crossing workflow boundaries)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(workCmd)
	rootCmd.AddCommand(markCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(helpAllCmd)
	rootCmd.AddCommand(versionCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Print(banner)

	interactive := isTTY()

	// Workspace path — prompt if not explicitly set and running interactively.
	if !cmd.Root().PersistentFlags().Changed("workspace") && interactive {
		cwd, _ := os.Getwd()
		ans := promptLine(fmt.Sprintf("Workspace path [%s]: ", cwd))
		if ans == "" {
			globalWorkspace = cwd
		} else {
			globalWorkspace = ans
		}
	}

	// Sample workers — prompt if not explicitly set and running interactively.
	if !cmd.Flags().Changed("with-sample-workers") && interactive {
		ans := promptLine("Include sample workers? [y/N]: ")
		ans = strings.ToLower(strings.TrimSpace(ans))
		initWithSampleWorkers = ans == "y" || ans == "yes"
	}

	opts := workspace.InitOptions{
		Root:              globalWorkspace,
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
	root, err := resolveRoot(globalWorkspace)
	if err != nil {
		return err
	}

	if len(args) == 1 {
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

	report := health.Run(root)
	health.Print(report)
	return nil
}

func runNext(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(globalWorkspace)
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

	interactive := isTTY()
	useResume := false

	switch s.Status {
	case "pending":
		fmt.Println()
		fmt.Println("Intake has not run yet. Launching intake agent:")

	case "active":
		sessionActive := s.Runtime.Tmux != nil && tmux.Available() && tmux.SessionExists(s.Runtime.Tmux.Session)
		if sessionActive {
			fmt.Println()
			fmt.Printf("⚠ tmux session %q is already running.\n", s.Runtime.Tmux.Session)
			if interactive {
				ans := promptLine("  Attach to existing session? [Y/n]: ")
				ans = strings.ToLower(strings.TrimSpace(ans))
				if ans == "" || ans == "y" || ans == "yes" {
					return tmux.Attach(s.Runtime.Tmux.Session)
				}
				fmt.Println("Cancelled.")
				return nil
			}
			return tmux.Attach(s.Runtime.Tmux.Session)
		} else {
			fmt.Println()
			fmt.Println("⚠ Ticket is active but no session found — likely interrupted.")
			if interactive {
				ans := promptLine("  Launch with recovery context? [Y/n]: ")
				ans = strings.ToLower(strings.TrimSpace(ans))
				if ans == "" || ans == "y" || ans == "yes" {
					useResume = true
				}
			} else {
				useResume = true
			}
		}

	case "paused":
		reason := s.NextAction.Prompt
		if len(s.History) > 0 {
			reason = s.History[len(s.History)-1].Result
		}
		fmt.Println()
		fmt.Printf("⚠ Ticket is paused:\n  %s\n", reason)
		if interactive {
			ans := promptLine("  Launch with recovery context? [Y/n]: ")
			ans = strings.ToLower(strings.TrimSpace(ans))
			if ans != "" && ans != "y" && ans != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}
		useResume = true
	}
	fmt.Println()

	plan, err := runner.Compute(root, featureDir, nextWorker)
	if err != nil {
		return err
	}

	if useResume {
		ctx, err := resume.Build(root, featureDir)
		if err != nil {
			return fmt.Errorf("building resume prompt: %w", err)
		}
		plan.Prompt = ctx.Prompt
		fmt.Println("Using recovery context.")
		fmt.Println()
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
	root, err := resolveRoot(globalWorkspace)
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
	root, err := resolveRoot(globalWorkspace)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		featureDir, err := state.FindFeatureDir(root, args[0])
		if err != nil {
			return err
		}
		s, err := state.Load(featureDir)
		if err != nil {
			return err
		}
		if statusJSON {
			return printJSON(s)
		}
		return printShow(root, featureDir, s)
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

func printShow(root, featureDir string, s *state.State) error {
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
	workflow := resolveWorkflow(root, s.Workflow)
	fmt.Printf("  Stage:     %s · %s\n", workflow, s.Stage.Name)
	fmt.Printf("  Owner:     %s\n", s.Stage.Owner)
	if wfCfg, err := config.Load(root); err == nil {
		if next := wfCfg.NextStage(workflow, s.Stage.Name); next != "" {
			sc, _ := wfCfg.StageConfig(workflow, next)
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
	case "paused":
		reason := ""
		if len(s.History) > 0 {
			reason = s.History[len(s.History)-1].Result
		}
		if reason == "" {
			reason = s.NextAction.Prompt
		}
		fmt.Printf("  Paused:  %s\n", reason)
		fmt.Println("  Run `orc next` after resolving to continue.")
	default:
		allWorkers, _ := workers.Load(filepath.Join(root, "workers"))
		wfCfg, _ := config.Load(root)
		sc, _ := wfCfg.StageConfig(workflow, s.Stage.Name)
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
			ts := h.At
			if t, err := time.Parse(time.RFC3339, h.At); err == nil {
				ts = t.Format("2006-01-02 15:04")
			}
			fmt.Printf("  %-16s  %-20s  %-20s  %s\n", ts, h.Stage, h.Owner, h.Result)
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

func runMark(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(globalWorkspace)
	if err != nil {
		return err
	}

	ticket := args[0]
	action := strings.ToLower(args[1])

	featureDir, err := state.FindFeatureDir(root, ticket)
	if err != nil {
		return err
	}

	if action == "pause" && len(args) < 3 {
		return fmt.Errorf("orc mark %s pause requires a reason — e.g. orc mark %s pause \"<reason>\"", ticket, ticket)
	}

	switch action {
	case "start":
		s, err := state.Load(featureDir)
		if err != nil {
			return err
		}
		if err := state.Start(featureDir); err != nil {
			return err
		}
		workflow := resolveWorkflow(root, s.Workflow)
		fmt.Printf("Ticket:   %s\n", s.Ticket)
		fmt.Printf("Status:   active\n")
		fmt.Printf("Stage:    %s · %s\n", workflow, s.Stage.Name)
		fmt.Printf("Owner:    %s\n", s.Stage.Owner)
		return nil

	case "pause":
		s, err := state.Load(featureDir)
		if err != nil {
			return err
		}
		if err := state.ValidateRepos(s, root); err != nil {
			return err
		}
		reason := strings.Join(args[2:], " ")
		if err := state.Pause(featureDir, reason); err != nil {
			return err
		}
		fmt.Printf("Ticket:  %s\n", s.Ticket)
		fmt.Printf("Status:  paused\n")
		fmt.Printf("Reason:  %s\n", reason)
		fmt.Printf("\nRun `orc next %s` to resume once resolved.\n", s.Ticket)
		return nil

	case "next":
		return runMarkNext(root, featureDir)

	case "done":
		s, err := state.Load(featureDir)
		if err != nil {
			return err
		}
		result := markResult
		if result == "" {
			result = "done"
		}
		if err := state.Done(featureDir, result); err != nil {
			return err
		}
		fmt.Printf("Ticket:  %s\n", s.Ticket)
		fmt.Printf("Status:  done\n")
		return nil

	default:
		return fmt.Errorf("unknown action %q — use: start | next [--result] [--stage] [--worker] | pause <reason> | done [--result]", action)
	}
}

func runMarkNext(root, featureDir string) error {
	s, err := state.Load(featureDir)
	if err != nil {
		return err
	}

	// Guard: archived or done tickets cannot be advanced.
	if s.Status == "archived" || s.Status == "done" {
		return fmt.Errorf("ticket %s is %s — cannot advance", s.Ticket, s.Status)
	}

	if err := state.ValidateRepos(s, root); err != nil {
		return err
	}

	workflowCfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	workflow := resolveWorkflow(root, s.Workflow)
	prevStage := s.Stage.Name

	// If --stage is given manually, validate it exists in the workflow or repair stages.
	if markStage != "" {
		if _, ok := workflowCfg.StageConfig(workflow, markStage); !ok {
			return fmt.Errorf("stage %q not found in workflow %q — check orc.yaml", markStage, workflow)
		}
	}

	// Auto-advance to next stage in the feature's pipeline.
	nextStage := markStage
	if nextStage == "" {
		nextStage = workflowCfg.NextStage(workflow, prevStage)
	}

	// Guard: manual gate — agent must call orc mark pause, not orc mark next.
	if nextStage != "" {
		if sc, ok := workflowCfg.StageConfig(workflow, prevStage); ok && sc.Advance == "manual" {
			return fmt.Errorf(
				"stage %q has advance: manual — use `orc mark %s pause \"<reason>\"` so a human can review before continuing",
				prevStage, s.Ticket,
			)
		}
	}

	// Guard: repair stage max_retries.
	if rs, ok := workflowCfg.RepairStage(prevStage); ok && rs.MaxRetries > 0 {
		count := s.StageCounts[prevStage]
		if count >= rs.MaxRetries {
			return fmt.Errorf(
				"repair stage %q has reached max_retries (%d) — resolve manually or use `orc mark %s pause`",
				prevStage, rs.MaxRetries, s.Ticket,
			)
		}
	}

	result := markResult
	if result == "" {
		if nextStage != "" && nextStage != prevStage {
			result = fmt.Sprintf("advanced from %s to %s", prevStage, nextStage)
		} else {
			result = fmt.Sprintf("completed %s", prevStage)
		}
	}

	if err := state.Next(featureDir, nextStage, markWorker, result); err != nil {
		return err
	}

	fmt.Printf("Ticket:   %s\n", s.Ticket)
	if nextStage != "" && nextStage != prevStage {
		fmt.Printf("Stage:    %s → %s\n", prevStage, nextStage)
	} else if nextStage == "" {
		fmt.Printf("Stage:    %s  (final)\n", prevStage)
		fmt.Printf("Status:   done\n")
	} else {
		fmt.Printf("Stage:    %s  (unchanged)\n", prevStage)
	}
	if markWorker != "" {
		fmt.Printf("Worker:   %s\n", markWorker)
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
		return nil
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
	root, err := resolveRoot(globalWorkspace)
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

func runTui(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(globalWorkspace)
	if err != nil {
		return err
	}
	return tui.Run(root)
}

func runAttach(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(globalWorkspace)
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
