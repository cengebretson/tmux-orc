package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Available returns true if tmux is installed and in PATH.
func Available() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// SessionExists returns true if a tmux session with the given name exists.
func SessionExists(name string) bool {
	return exec.Command("tmux", "has-session", "-t", name).Run() == nil
}

// WindowExists returns true if the named window exists in the session.
func WindowExists(session, window string) bool {
	target := session + ":" + window
	return exec.Command("tmux", "select-window", "-t", target).Run() == nil
}

// CreateSession creates a detached tmux session with windows named after each workflow.
func CreateSession(slug, featureDir string, workflows []string) error {
	if len(workflows) == 0 {
		workflows = []string{"shell"}
	}

	// Create detached session with first window
	if err := exec.Command("tmux", "new-session", "-d", "-s", slug, "-n", workflows[0], "-c", featureDir).Run(); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	// Add remaining workflow windows
	for _, w := range workflows[1:] {
		if err := exec.Command("tmux", "new-window", "-t", slug, "-n", w, "-c", featureDir).Run(); err != nil {
			return fmt.Errorf("add window %s: %w", w, err)
		}
	}

	// Select the first window — ignore error if focus fails (non-fatal)
	_ = exec.Command("tmux", "select-window", "-t", slug+":"+workflows[0]).Run()
	return nil
}

// SendCommand sends a shell command to a window in the session, creating the window if needed.
// runDir is the working directory the command should execute from.
func SendCommand(session, window, featureDir, runDir string, argv []string) error {
	target := session + ":" + window

	// Create window if it doesn't exist
	if !WindowExists(session, window) {
		if err := exec.Command("tmux", "new-window", "-t", session, "-n", window, "-c", featureDir).Run(); err != nil {
			return fmt.Errorf("create window %s: %w", window, err)
		}
	}

	// Write command to a temp script to avoid quoting issues with send-keys.
	// The script cds to runDir first so the agent process starts in the right directory,
	// then self-deletes after the command exits.
	script, err := writeScript(runDir, argv)
	if err != nil {
		return err
	}

	return exec.Command("tmux", "send-keys", "-t", target, "bash "+script, "Enter").Run()
}

// Attach attaches to a session (or switches if already inside tmux).
// target is "session" or "session:window".
func Attach(target string) error {
	if os.Getenv("TMUX") != "" {
		return exec.Command("tmux", "switch-client", "-t", target).Run()
	}
	cmd := exec.Command("tmux", "attach-session", "-t", target)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// KillSession kills a tmux session by name.
func KillSession(name string) error {
	if err := exec.Command("tmux", "kill-session", "-t", name).Run(); err != nil {
		return fmt.Errorf("kill session %s: %w", name, err)
	}
	return nil
}

// ListSessions returns all tmux session names, or nil if none exist.
func ListSessions() []string {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		return nil
	}
	return strings.Fields(string(out))
}

// AttachHint returns the command string a user should run to attach.
func AttachHint(session, window string) string {
	return fmt.Sprintf("tmux attach -t %s:%s", session, window)
}

func writeScript(runDir string, argv []string) (string, error) {
	f, err := os.CreateTemp("", "orc-launch-*.sh")
	if err != nil {
		return "", fmt.Errorf("temp script: %w", err)
	}
	defer f.Close() //nolint:errcheck

	var parts []string
	for _, arg := range argv {
		parts = append(parts, shellQuote(arg))
	}
	// cd to the right directory, run the command, then self-delete the script.
	if _, err := fmt.Fprintf(f, "#!/usr/bin/env bash\ncd %s\n%s\nrm -f %s\n",
		shellQuote(runDir),
		strings.Join(parts, " "),
		shellQuote(f.Name()),
	); err != nil {
		return "", fmt.Errorf("write script: %w", err)
	}
	return f.Name(), nil
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\"'\\$`!;|&<>(){}") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
