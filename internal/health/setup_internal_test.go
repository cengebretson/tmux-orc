package health

import "testing"

func TestSetupSectionComplete(t *testing.T) {
	tests := []struct {
		name    string
		content string
		key     string
		want    bool
	}{
		{
			name:    "one space",
			content: "## Status\n\nshared: complete\nclaude: pending\n",
			key:     "shared",
			want:    true,
		},
		{
			name:    "two spaces (aligned template)",
			content: "shared: pending\nclaude: pending\ncodex:  complete\n",
			key:     "codex",
			want:    true,
		},
		{
			name:    "still pending",
			content: "shared: pending\nclaude: pending\ncodex:  pending\n",
			key:     "shared",
			want:    false,
		},
		{
			// Regression guard: the instruction prose contains the literal
			// "claude: complete" inside backticks but never on its own line, so
			// it must not be read as a completed section.
			name:    "inline prose does not count",
			content: "2. mark `claude: complete` when done\nclaude: pending\n",
			key:     "claude",
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setupSectionComplete(tt.content, tt.key); got != tt.want {
				t.Errorf("setupSectionComplete(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}
