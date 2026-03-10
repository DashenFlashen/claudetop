package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"claudetop/internal/config"
	"claudetop/internal/session"
)

// State is the persisted application state.
type State struct {
	Sessions []*session.Session `json:"sessions"`
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
	if os.IsNotExist(err) {
		return &State{Sessions: []*session.Session{}}, nil
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
