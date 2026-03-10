package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"claudetop/internal/config"
	"claudetop/internal/session"
)

// InboxItem is a captured note in the inbox.
type InboxItem struct {
	ID      string    `json:"id"`
	Content string    `json:"content"`
	Source  string    `json:"source"`
	AddedAt time.Time `json:"added_at"`
	Parked  bool      `json:"parked,omitempty"`
}

// NewInboxItem creates a new inbox item with a unique ID.
func NewInboxItem(content, source string) *InboxItem {
	return &InboxItem{
		ID:      fmt.Sprintf("%x", time.Now().UnixNano()),
		Content: content,
		Source:  source,
		AddedAt: time.Now(),
	}
}

// State is the persisted application state.
type State struct {
	Sessions   []*session.Session `json:"sessions"`
	InboxItems []*InboxItem       `json:"inbox_items,omitempty"`
}

func path() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

// Load reads state from disk. Returns empty state if file is missing.
func Load() (*State, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return &State{Sessions: []*session.Session{}, InboxItems: []*InboxItem{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	if s.Sessions == nil {
		s.Sessions = []*session.Session{}
	}
	if s.InboxItems == nil {
		s.InboxItems = []*InboxItem{}
	}
	return &s, nil
}

// Save writes state to disk atomically via a temp file rename.
func Save(s *State) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return os.Rename(tmp, p)
}
