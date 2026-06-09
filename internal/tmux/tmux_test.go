package tmux

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestWriteScriptCleansUpAfterCommandFailure(t *testing.T) {
	script, err := writeScript(t.TempDir(), []string{"false"})
	if err != nil {
		t.Fatalf("writeScript: %v", err)
	}

	if err := exec.Command("bash", script).Run(); err == nil {
		t.Fatal("expected script command to fail")
	}
	if _, err := os.Stat(script); !os.IsNotExist(err) {
		t.Fatalf("script still exists after failed command: %v", err)
	}
}

func TestWriteScriptQuotesArguments(t *testing.T) {
	runDir := t.TempDir()
	script, err := writeScript(runDir, []string{"printf", "%s", "value with 'quotes'"})
	if err != nil {
		t.Fatalf("writeScript: %v", err)
	}
	defer os.Remove(script) //nolint:errcheck

	content, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "trap 'rm -f ") {
		t.Fatalf("script missing cleanup trap:\n%s", text)
	}
	if !strings.Contains(text, "cd "+shellQuote(runDir)) {
		t.Fatalf("script missing quoted run dir:\n%s", text)
	}
	if !strings.Contains(text, shellQuote("value with 'quotes'")) {
		t.Fatalf("script missing quoted argument:\n%s", text)
	}
}
