package session

import (
	"fmt"
	"time"
)

// Status represents the detected state of a Claude Code session.
type Status int

const (
	StatusStarting   Status = iota // first 10 seconds
	StatusWorking                  // actively producing output
	StatusNeedsInput               // waiting for user to type
	StatusPermission               // waiting for command approval
	StatusDone                     // idle, task complete
	StatusStuck                    // working but silent > 2 minutes
	StatusError                    // error pattern detected
)

func (s Status) String() string {
	switch s {
	case StatusStarting:
		return "starting"
	case StatusWorking:
		return "working"
	case StatusNeedsInput:
		return "needs_input"
	case StatusPermission:
		return "permission"
	case StatusDone:
		return "done"
	case StatusStuck:
		return "stuck"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Session represents a single Claude Code session.
type Session struct {
	// Persisted fields
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Dead      bool      `json:"dead,omitempty"`

	// Runtime-only (not persisted)
	Status       Status    `json:"-"`
	PaneContent  string    `json:"-"`
	LastOutputAt time.Time `json:"-"`
}

// TmuxName returns the namespaced tmux session name.
func (s *Session) TmuxName() string {
	return "ct-" + s.ID
}

// DisplayName returns the name to show in the sidebar.
func (s *Session) DisplayName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.ID
}

// NewSession creates a session with a unique ID.
func NewSession(name string, index int) *Session {
	id := name
	if id == "" {
		id = fmt.Sprintf("session-%d", index)
	}
	return &Session{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now(),
		Status:    StatusStarting,
	}
}
