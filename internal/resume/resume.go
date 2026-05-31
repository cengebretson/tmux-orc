package resume

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cengebretson/orc/internal/state"
)

type Context struct {
	Ticket     string
	Slug       string
	Stage      string
	Status     string
	Workflow   string
	History    []state.HistoryEntry
	StageFiles []string // files found in the stage output folder
	HasDecisions bool
	Prompt     string // the assembled recovery prompt
}

// Build reads the feature folder and assembles a recovery context for a stuck ticket.
func Build(root, featureDir string) (*Context, error) {
	s, err := state.Load(featureDir)
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	ctx := &Context{
		Ticket:   s.Ticket,
		Slug:     s.Slug,
		Stage:    s.Stage.Name,
		Status:   s.Status,
		Workflow: s.Workflow,
		History:  s.History,
	}

	// Check for DECISIONS.md.
	if _, err := os.Stat(filepath.Join(featureDir, "DECISIONS.md")); err == nil {
		ctx.HasDecisions = true
	}

	// List files written to the current stage output folder.
	stageDir := filepath.Join(featureDir, s.Stage.Name)
	if entries, err := os.ReadDir(stageDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				ctx.StageFiles = append(ctx.StageFiles, e.Name())
			}
		}
	}

	ctx.Prompt = buildPrompt(root, featureDir, s, ctx)
	return ctx, nil
}

func buildPrompt(root, featureDir string, s *state.State, ctx *Context) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Before starting: read AGENTS.md and ORC.md. Run `orc start %s` to mark in_progress.\n\n", s.Ticket))
	b.WriteString(fmt.Sprintf("## Resuming %s — stage: %s\n\n", s.Ticket, s.Stage.Name))
	b.WriteString(fmt.Sprintf("This session was interrupted. Status was `%s` when it stopped.\n\n", s.Status))

	// Last few history entries for context.
	b.WriteString("### Recent history\n\n")
	entries := s.History
	if len(entries) > 5 {
		entries = entries[len(entries)-5:]
	}
	for _, h := range entries {
		b.WriteString(fmt.Sprintf("- **%s** (%s, %s): %s\n", h.Stage, h.Owner, h.At, h.Result))
	}
	b.WriteString("\n")

	// What the stage has produced so far.
	if len(ctx.StageFiles) > 0 {
		b.WriteString(fmt.Sprintf("### Partial outputs in %s/\n\n", s.Stage.Name))
		for _, f := range ctx.StageFiles {
			b.WriteString(fmt.Sprintf("- `%s/%s`\n", s.Stage.Name, f))
		}
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Read these files to understand what was completed before the interruption.\n\n"))
	} else {
		b.WriteString(fmt.Sprintf("No output files found in `%s/` — the stage may not have started yet.\n\n", s.Stage.Name))
	}

	// Key context files.
	b.WriteString("### Context to read\n\n")
	b.WriteString(fmt.Sprintf("- `features/%s/STATE.yaml` — current state and history\n", s.Slug))
	b.WriteString(fmt.Sprintf("- `features/%s/TICKET.md` — original ticket\n", s.Slug))
	b.WriteString(fmt.Sprintf("- `features/%s/SPEC.md` — scope and requirements\n", s.Slug))
	if ctx.HasDecisions {
		b.WriteString(fmt.Sprintf("- `features/%s/DECISIONS.md` — decisions made so far\n", s.Slug))
	}
	b.WriteString(fmt.Sprintf("- `stages/%s.md` — stage instructions\n", s.Stage.Name))
	b.WriteString("\n")

	// End-of-session instruction.
	b.WriteString("### When done\n\n")
	b.WriteString(fmt.Sprintf("Run one of:\n"))
	b.WriteString(fmt.Sprintf("  orc advance %s --owner <worker-id> --result \"<summary>\"\n", s.Ticket))
	b.WriteString(fmt.Sprintf("  orc wait %s \"<what you need from the human>\"\n", s.Ticket))
	b.WriteString(fmt.Sprintf("  orc block %s \"<what is preventing progress>\"\n", s.Ticket))

	return b.String()
}
