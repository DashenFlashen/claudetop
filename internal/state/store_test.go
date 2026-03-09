package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"claudetop/internal/session"
)

func TestLoadEmptyWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	s, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(s.Sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(s.Sessions))
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	os.MkdirAll(filepath.Join(tmp, ".claudetop"), 0755)

	sess := &session.Session{
		ID:        "test-session",
		Name:      "test",
		CreatedAt: time.Now().Truncate(time.Second),
	}
	s := &State{Sessions: []*session.Session{sess}}

	if err := Save(s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(loaded.Sessions))
	}
	if loaded.Sessions[0].ID != "test-session" {
		t.Errorf("expected ID=test-session, got %q", loaded.Sessions[0].ID)
	}
}
