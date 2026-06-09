package orchestrator

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cengebretson/orc/internal/runner"
	"github.com/cengebretson/orc/internal/state"
	"github.com/cengebretson/orc/internal/workers"
)

func TestLauncherLaunchesInTmux(t *testing.T) {
	s := &state.State{
		Slug: "TICKET-1",
		Stage: state.Stage{
			Name: "develop",
		},
	}
	plan := &runner.Plan{
		CWD:        "/workspace",
		LaunchArgv: []string{"codex", "do it"},
		Worker:     &workers.Worker{Name: "Dev", Engine: "codex"},
	}

	var createdSession string
	var sent []string
	var sendEvent []string
	var history []string

	launcher := Launcher{
		TmuxAvailable: func() bool { return true },
		SessionExists: func(string) bool { return false },
		CreateSession: func(slug, featureDir string, workflows []string) error {
			createdSession = slug
			return nil
		},
		SetRuntime: func(featureDir, tmuxSession string) error { return nil },
		SendCommand: func(session, window, featureDir, runDir string, argv []string) error {
			sent = []string{session, window, runDir}
			return nil
		},
		AppendHistory: func(featureDir, stage, workerID, result string) error {
			history = []string{stage, workerID, result}
			return nil
		},
		RunForeground: func(opts LaunchOptions) error {
			t.Fatal("foreground should not run")
			return nil
		},
		AttachHint: func(session, window string) string { return session + ":" + window },
	}

	result, err := launcher.Launch(LaunchOptions{
		FeatureDir: "/feature",
		State:      s,
		Plan:       plan,
		OnTmuxSend: func(session, window string) {
			sendEvent = []string{session, window}
		},
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	if result.Mode != LaunchModeTmux {
		t.Fatalf("Mode = %q, want %q", result.Mode, LaunchModeTmux)
	}
	if createdSession != "TICKET-1" {
		t.Errorf("createdSession = %q, want TICKET-1", createdSession)
	}
	if !reflect.DeepEqual(sent, []string{"TICKET-1", "develop", "/workspace"}) {
		t.Errorf("sent = %#v", sent)
	}
	if !reflect.DeepEqual(sendEvent, []string{"TICKET-1", "develop"}) {
		t.Errorf("sendEvent = %#v", sendEvent)
	}
	if result.AttachHint != "TICKET-1:develop" {
		t.Errorf("AttachHint = %q", result.AttachHint)
	}
	if !reflect.DeepEqual(history, []string{"develop", "", "launched in tmux session TICKET-1:develop"}) {
		t.Errorf("history = %#v", history)
	}
}

func TestLauncherFallsBackToForegroundWhenTmuxCreateFails(t *testing.T) {
	s := &state.State{
		Slug:  "TICKET-1",
		Stage: state.Stage{Name: "develop"},
	}
	plan := &runner.Plan{
		CWD:        "/workspace",
		LaunchArgv: []string{"codex", "do it"},
		Worker:     &workers.Worker{Name: "Dev", Engine: "codex"},
	}

	var foregroundRan bool
	var fallbackMessages []string
	var history []string
	createErr := errors.New("no tmux")

	launcher := Launcher{
		TmuxAvailable: func() bool { return true },
		CreateSession: func(slug, featureDir string, workflows []string) error {
			return createErr
		},
		SendCommand: func(session, window, featureDir, runDir string, argv []string) error {
			t.Fatal("send should not run after create failure")
			return nil
		},
		RunForeground: func(opts LaunchOptions) error {
			foregroundRan = true
			return nil
		},
		AppendHistory: func(featureDir, stage, workerID, result string) error {
			history = []string{stage, workerID, result}
			return nil
		},
	}

	result, err := launcher.Launch(LaunchOptions{
		FeatureDir: "/feature",
		State:      s,
		Plan:       plan,
		OnFallback: func(message string) {
			fallbackMessages = append(fallbackMessages, message)
		},
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if result.Mode != LaunchModeForeground {
		t.Fatalf("Mode = %q, want %q", result.Mode, LaunchModeForeground)
	}
	if !foregroundRan {
		t.Fatal("foreground did not run")
	}
	if len(fallbackMessages) != 1 || fallbackMessages[0] != "tmux session create failed (no tmux)" {
		t.Fatalf("fallbackMessages = %#v", fallbackMessages)
	}
	if !reflect.DeepEqual(history, []string{"develop", "", "launched in foreground"}) {
		t.Errorf("history = %#v", history)
	}
}

func TestLauncherUsesExistingTmuxWindowOverride(t *testing.T) {
	s := &state.State{
		Slug: "TICKET-1",
		Stage: state.Stage{
			Name: "develop",
		},
		Runtime: state.Runtime{
			Tmux: &state.TmuxRuntime{Session: "existing"},
		},
	}
	plan := &runner.Plan{
		CWD:        "/feature",
		LaunchArgv: []string{"codex", "do it"},
		Worker:     &workers.Worker{Name: "Dev", Engine: "codex"},
	}

	var sent []string
	var history []string
	launcher := Launcher{
		TmuxAvailable: func() bool { return true },
		SessionExists: func(session string) bool { return session == "existing" },
		CreateSession: func(slug, featureDir string, workflows []string) error {
			t.Fatal("existing-session launch should not create a session")
			return nil
		},
		SendCommand: func(session, window, featureDir, runDir string, argv []string) error {
			sent = []string{session, window, runDir}
			return nil
		},
		AppendHistory: func(featureDir, stage, workerID, result string) error {
			history = []string{stage, workerID, result}
			return nil
		},
		RunForeground: func(opts LaunchOptions) error {
			t.Fatal("foreground should not run")
			return nil
		},
		AttachHint: func(session, window string) string { return session + ":" + window },
	}

	result, err := launcher.Launch(LaunchOptions{
		FeatureDir:          "/feature",
		State:               s,
		Plan:                plan,
		Window:              "jit",
		RequireExistingTmux: true,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if result.Mode != LaunchModeTmux {
		t.Fatalf("Mode = %q, want %q", result.Mode, LaunchModeTmux)
	}
	if !reflect.DeepEqual(sent, []string{"existing", "jit", "/feature"}) {
		t.Errorf("sent = %#v", sent)
	}
	if !reflect.DeepEqual(history, []string{"jit", "", "launched in tmux session existing:jit"}) {
		t.Errorf("history = %#v", history)
	}
}

func TestLauncherSkipsTmuxWhenDisabled(t *testing.T) {
	s := &state.State{Slug: "TICKET-1", Stage: state.Stage{Name: "develop"}}
	plan := &runner.Plan{
		CWD:        "/feature",
		LaunchArgv: []string{"codex", "do it"},
		Worker:     &workers.Worker{Name: "Dev", Engine: "codex"},
	}

	var foregroundRan bool
	var history []string
	launcher := Launcher{
		TmuxAvailable: func() bool {
			t.Fatal("tmux availability should not be checked")
			return true
		},
		RunForeground: func(opts LaunchOptions) error {
			foregroundRan = true
			return nil
		},
		AppendHistory: func(featureDir, stage, workerID, result string) error {
			history = []string{stage, workerID, result}
			return nil
		},
	}

	result, err := launcher.Launch(LaunchOptions{
		FeatureDir:  "/feature",
		State:       s,
		Plan:        plan,
		DisableTmux: true,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if result.Mode != LaunchModeForeground {
		t.Fatalf("Mode = %q, want %q", result.Mode, LaunchModeForeground)
	}
	if !foregroundRan {
		t.Fatal("foreground did not run")
	}
	if !reflect.DeepEqual(history, []string{"develop", "", "launched in foreground"}) {
		t.Errorf("history = %#v", history)
	}
}

func TestLauncherRecordsForegroundFailure(t *testing.T) {
	s := &state.State{Slug: "TICKET-1", Stage: state.Stage{Name: "develop"}}
	plan := &runner.Plan{
		CWD:        "/feature",
		LaunchArgv: []string{"codex", "do it"},
		Worker:     &workers.Worker{ID: "dev", Name: "Dev", Engine: "codex"},
	}

	runErr := errors.New("agent failed")
	var history []string
	launcher := Launcher{
		TmuxAvailable: func() bool { return false },
		RunForeground: func(opts LaunchOptions) error {
			return runErr
		},
		AppendHistory: func(featureDir, stage, workerID, result string) error {
			history = []string{stage, workerID, result}
			return nil
		},
	}

	result, err := launcher.Launch(LaunchOptions{
		FeatureDir: "/feature",
		State:      s,
		Plan:       plan,
	})
	if !errors.Is(err, runErr) {
		t.Fatalf("Launch error = %v, want %v", err, runErr)
	}
	if result.Mode != LaunchModeForeground {
		t.Fatalf("Mode = %q, want %q", result.Mode, LaunchModeForeground)
	}
	if !reflect.DeepEqual(history, []string{"develop", "dev", "launch failed in foreground: agent failed"}) {
		t.Errorf("history = %#v", history)
	}
}
