package orchestrator

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/cengebretson/orc/internal/config"
	"github.com/cengebretson/orc/internal/runner"
	"github.com/cengebretson/orc/internal/state"
	"github.com/cengebretson/orc/internal/tmux"
)

type LaunchMode string

const (
	LaunchModeTmux       LaunchMode = "tmux"
	LaunchModeForeground LaunchMode = "foreground"
)

type LaunchResult struct {
	Mode            LaunchMode
	Session         string
	Window          string
	AttachHint      string
	Fallbacks       []string
	HistoryWarnings []string
}

type LaunchOptions struct {
	Root       string
	FeatureDir string
	State      *state.State
	Plan       *runner.Plan
	Window     string
	In         io.Reader
	Out        io.Writer
	Err        io.Writer

	DisableTmux         bool
	RequireExistingTmux bool

	OnFallback       func(message string)
	OnHistoryWarning func(message string)
	OnTmuxSend       func(session, window string)
	OnForeground     func()
}

type Launcher struct {
	TmuxAvailable func() bool
	SessionExists func(string) bool
	CreateSession func(slug, featureDir string, workflows []string) error
	SendCommand   func(session, window, featureDir, runDir string, argv []string) error
	SetRuntime    func(featureDir, tmuxSession string) error
	AppendHistory func(featureDir, stage, workerID, result string) error
	RunForeground func(opts LaunchOptions) error
	AttachHint    func(session, window string) string
}

func NewLauncher() Launcher {
	return Launcher{
		TmuxAvailable: tmux.Available,
		SessionExists: tmux.SessionExists,
		CreateSession: tmux.CreateSession,
		SendCommand:   tmux.SendCommand,
		SetRuntime:    state.SetRuntime,
		AppendHistory: state.AppendHistory,
		RunForeground: runForeground,
		AttachHint:    tmux.AttachHint,
	}
}

func Launch(opts LaunchOptions) (*LaunchResult, error) {
	launcher := NewLauncher()
	return launcher.Launch(opts)
}

func (l Launcher) Launch(opts LaunchOptions) (*LaunchResult, error) {
	if opts.State == nil {
		return nil, fmt.Errorf("state is required")
	}
	if opts.Plan == nil {
		return nil, fmt.Errorf("plan is required")
	}

	if l.TmuxAvailable == nil {
		l.TmuxAvailable = tmux.Available
	}
	if l.SessionExists == nil {
		l.SessionExists = tmux.SessionExists
	}
	if l.CreateSession == nil {
		l.CreateSession = tmux.CreateSession
	}
	if l.SendCommand == nil {
		l.SendCommand = tmux.SendCommand
	}
	if l.SetRuntime == nil {
		l.SetRuntime = state.SetRuntime
	}
	if l.AppendHistory == nil {
		l.AppendHistory = state.AppendHistory
	}
	if l.RunForeground == nil {
		l.RunForeground = runForeground
	}
	if l.AttachHint == nil {
		l.AttachHint = tmux.AttachHint
	}

	result := &LaunchResult{
		Mode:   LaunchModeForeground,
		Window: opts.State.Stage.Name,
	}
	if opts.Window != "" {
		result.Window = opts.Window
	}

	if !opts.DisableTmux && l.TmuxAvailable() {
		session := opts.State.Slug
		if opts.State.Runtime.Tmux != nil {
			session = opts.State.Runtime.Tmux.Session
		}
		result.Session = session
		tmuxReady := false

		if opts.RequireExistingTmux {
			if opts.State.Runtime.Tmux != nil && l.SessionExists(session) {
				tmuxReady = true
			}
		} else if opts.State.Runtime.Tmux == nil {
			if err := l.CreateSession(session, opts.FeatureDir, stageNamesForTicket(opts.Root, opts.State)); err != nil {
				result.addFallback(opts, fmt.Sprintf("tmux session create failed (%v)", err))
			} else if err := l.SetRuntime(opts.FeatureDir, session); err != nil {
				result.addFallback(opts, fmt.Sprintf("warning: could not write runtime to STATE.yaml: %v", err))
				tmuxReady = true
			} else {
				tmuxReady = true
			}
		} else if !l.SessionExists(session) {
			if err := l.CreateSession(session, opts.FeatureDir, stageNamesForTicket(opts.Root, opts.State)); err != nil {
				result.addFallback(opts, fmt.Sprintf("tmux session recreate failed (%v)", err))
			} else {
				tmuxReady = true
			}
		} else {
			tmuxReady = true
		}

		if tmuxReady {
			if opts.OnTmuxSend != nil {
				opts.OnTmuxSend(session, result.Window)
			}
			if err := l.SendCommand(session, result.Window, opts.FeatureDir, opts.Plan.CWD, opts.Plan.LaunchArgv); err != nil {
				result.addFallback(opts, fmt.Sprintf("tmux send failed (%v)", err))
			} else {
				result.Mode = LaunchModeTmux
				result.AttachHint = l.AttachHint(session, result.Window)
				l.recordLaunch(opts, result, fmt.Sprintf("launched in tmux session %s:%s", session, result.Window))
				return result, nil
			}
		}
	}

	if opts.OnForeground != nil {
		opts.OnForeground()
	}
	if err := l.RunForeground(opts); err != nil {
		l.recordLaunch(opts, result, fmt.Sprintf("launch failed in foreground: %v", err))
		return result, err
	}
	l.recordLaunch(opts, result, "launched in foreground")
	return result, nil
}

func (r *LaunchResult) addFallback(opts LaunchOptions, message string) {
	r.Fallbacks = append(r.Fallbacks, message)
	if opts.OnFallback != nil {
		opts.OnFallback(message)
	}
}

func (l Launcher) recordLaunch(opts LaunchOptions, result *LaunchResult, message string) {
	if opts.FeatureDir == "" {
		return
	}
	stage := opts.Plan.Stage
	if stage == "" {
		stage = result.Window
	}
	workerID := ""
	if opts.Plan.Worker != nil {
		workerID = opts.Plan.Worker.ID
	}
	if err := l.AppendHistory(opts.FeatureDir, stage, workerID, message); err != nil {
		warning := fmt.Sprintf("warning: could not record launch history: %v", err)
		result.HistoryWarnings = append(result.HistoryWarnings, warning)
		if opts.OnHistoryWarning != nil {
			opts.OnHistoryWarning(warning)
		}
	}
}

func stageNamesForTicket(root string, s *state.State) []string {
	workflowCfg, err := config.Load(root)
	if err != nil {
		return nil
	}
	workflow := s.Workflow
	if workflow == "" {
		workflow = workflowCfg.DefaultWorkflow()
	}
	return workflowCfg.StageNames(workflow)
}

func runForeground(opts LaunchOptions) error {
	c := exec.Command(opts.Plan.LaunchArgv[0], opts.Plan.LaunchArgv[1:]...)
	c.Stdin = opts.In
	if c.Stdin == nil {
		c.Stdin = os.Stdin
	}
	c.Stdout = opts.Out
	if c.Stdout == nil {
		c.Stdout = os.Stdout
	}
	c.Stderr = opts.Err
	if c.Stderr == nil {
		c.Stderr = os.Stderr
	}
	c.Dir = opts.Plan.CWD
	return c.Run()
}
