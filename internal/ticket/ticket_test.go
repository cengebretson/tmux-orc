package ticket_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cengebretson/orc/internal/ticket"
)

func TestLoadFindsActiveTicket(t *testing.T) {
	root := t.TempDir()
	writeState(t, filepath.Join(root, "features", "TICKET-1"))

	got, err := ticket.Load(root, "TICKET-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.State.Ticket != "TICKET-1" {
		t.Fatalf("Ticket = %q, want TICKET-1", got.State.Ticket)
	}
}

func TestLoadWithArchiveFindsArchivedTicket(t *testing.T) {
	root := t.TempDir()
	writeState(t, filepath.Join(root, "features", "_archive", "TICKET-1"))

	got, err := ticket.LoadWithArchive(root, "TICKET-1")
	if err != nil {
		t.Fatalf("LoadWithArchive: %v", err)
	}
	if got.State.Ticket != "TICKET-1" {
		t.Fatalf("Ticket = %q, want TICKET-1", got.State.Ticket)
	}
}

func TestLoadDoesNotFindArchivedTicket(t *testing.T) {
	root := t.TempDir()
	writeState(t, filepath.Join(root, "features", "_archive", "TICKET-1"))

	if _, err := ticket.Load(root, "TICKET-1"); err == nil {
		t.Fatal("Load found archived ticket, want error")
	}
}

func writeState(t *testing.T, featureDir string) {
	t.Helper()
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(featureDir, "STATE.yaml"), []byte(`
ticket: TICKET-1
slug: TICKET-1
status: active
stage:
  worker: bob-developer
  name: develop
`), 0644); err != nil {
		t.Fatal(err)
	}
}
