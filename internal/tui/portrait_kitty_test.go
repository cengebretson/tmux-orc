package tui

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestKittyPlaceholderGridComposesAsText(t *testing.T) {
	const cols, rows = 24, 12
	grid := kittyPlaceholderGrid(7, cols, rows)

	lines := strings.Split(grid, "\n")
	if len(lines) != rows {
		t.Fatalf("got %d lines, want %d", len(lines), rows)
	}
	// lipgloss must measure each placeholder line as cols cells wide —
	// this is the property that raw Kitty escape output lacks.
	for i, line := range lines {
		if w := lipgloss.Width(line); w != cols {
			t.Errorf("line %d: lipgloss.Width = %d, want %d", i, w, cols)
		}
		if !strings.HasPrefix(line, "\x1b[38;5;7m") {
			t.Errorf("line %d: missing image-ID foreground SGR", i)
		}
		if !strings.HasSuffix(line, "\x1b[39m") {
			t.Errorf("line %d: missing foreground reset", i)
		}
	}

	// Each cell encodes its row and column via combining diacritics.
	wantCell := string(kittyPlaceholder) + string(kittyDiacritics[3]) + string(kittyDiacritics[5])
	if !strings.Contains(lines[3], wantCell) {
		t.Errorf("line 3 missing placeholder cell for row 3, col 5")
	}
}

func TestKittyTransmitSingleChunk(t *testing.T) {
	var buf bytes.Buffer
	if err := kittyTransmit(&buf, 9, []byte("png-bytes"), 10, 5, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "\x1b_Ga=T,U=1,q=2,f=100,i=9,c=10,r=5,m=0;") {
		t.Errorf("unexpected control data: %q", out)
	}
	if !strings.HasSuffix(out, "\x1b\\") {
		t.Errorf("missing escape terminator: %q", out)
	}
}

func TestKittyTransmitChunksAndTmuxWrap(t *testing.T) {
	var buf bytes.Buffer
	data := bytes.Repeat([]byte{0xAB}, 5000) // base64 length > 4096 → two chunks
	if err := kittyTransmit(&buf, 3, data, 8, 4, true); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if got := strings.Count(out, "\x1bPtmux;"); got != 2 {
		t.Fatalf("got %d tmux passthrough wrappers, want 2", got)
	}
	if !strings.Contains(out, "\x1b\x1b_Ga=T,U=1,q=2,f=100,i=3,c=8,r=4,m=1;") {
		t.Errorf("first chunk missing doubled-ESC control data: %q", out[:80])
	}
	if !strings.Contains(out, "\x1b\x1b_Gm=0;") {
		t.Errorf("final chunk missing m=0 continuation")
	}
}

func TestKittyPortraitSupported(t *testing.T) {
	for _, v := range []string{"ORC_PORTRAIT", "KITTY_WINDOW_ID", "GHOSTTY_RESOURCES_DIR", "GHOSTTY_BIN_DIR", "TERM"} {
		t.Setenv(v, "")
	}
	if kittyPortraitSupported() {
		t.Error("supported with no terminal hints")
	}
	t.Setenv("TERM", "xterm-ghostty")
	if !kittyPortraitSupported() {
		t.Error("not supported with TERM=xterm-ghostty")
	}
	t.Setenv("ORC_PORTRAIT", "symbols")
	if kittyPortraitSupported() {
		t.Error("ORC_PORTRAIT=symbols should force fallback")
	}
	t.Setenv("ORC_PORTRAIT", "kitty")
	if !kittyPortraitSupported() {
		t.Error("ORC_PORTRAIT=kitty should force pixel mode")
	}
}

func TestKittyImageIDStable(t *testing.T) {
	a := kittyImageID("WARRIOR-test")
	if a < 1 || a > 255 {
		t.Fatalf("id %d outside 1..255", a)
	}
	if b := kittyImageID("WARRIOR-test"); b != a {
		t.Errorf("id not stable: %d then %d", a, b)
	}
	if c := kittyImageID(fmt.Sprintf("OTHER-%d", a)); c == a {
		t.Errorf("distinct classes share id %d", a)
	}
}
