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

func TestInboxItemSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	os.MkdirAll(filepath.Join(tmp, ".claudetop"), 0755)

	item := NewInboxItem("fix the thing", "manual")
	s := &State{
		Sessions:   []*session.Session{},
		InboxItems: []*InboxItem{item},
	}

	if err := Save(s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(loaded.InboxItems) != 1 {
		t.Fatalf("expected 1 inbox item, got %d", len(loaded.InboxItems))
	}
	if loaded.InboxItems[0].Content != "fix the thing" {
		t.Errorf("expected Content=%q, got %q", "fix the thing", loaded.InboxItems[0].Content)
	}
	if loaded.InboxItems[0].Source != "manual" {
		t.Errorf("expected Source=%q, got %q", "manual", loaded.InboxItems[0].Source)
	}
}

func TestLoadEmptyInboxWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	s, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if s.InboxItems == nil {
		t.Error("expected InboxItems to be non-nil slice")
	}
	if len(s.InboxItems) != 0 {
		t.Errorf("expected 0 inbox items, got %d", len(s.InboxItems))
	}
}

func TestLastBriefingDatePersists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	s := &State{
		Sessions:         []*session.Session{},
		InboxItems:       []*InboxItem{},
		LastBriefingDate: "2026-03-11",
	}
	if err := Save(s); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.LastBriefingDate != "2026-03-11" {
		t.Errorf("got %q, want %q", loaded.LastBriefingDate, "2026-03-11")
	}
}

func TestSaveAndLoadParkedSession(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	os.MkdirAll(filepath.Join(tmp, ".claudetop"), 0755)

	sess := &session.Session{
		ID:        "parked-session",
		Name:      "milvus",
		CreatedAt: time.Now().Truncate(time.Second),
		Parked:    true,
		ParkNote:  "waiting for Björn",
	}
	s := &State{Sessions: []*session.Session{sess}}

	if err := Save(s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !loaded.Sessions[0].Parked {
		t.Error("expected Parked=true")
	}
	if loaded.Sessions[0].ParkNote != "waiting for Björn" {
		t.Errorf("expected ParkNote=%q, got %q", "waiting for Björn", loaded.Sessions[0].ParkNote)
	}
}
