package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"claudetop/internal/config"
	"claudetop/internal/session"
	"claudetop/internal/state"
	"claudetop/internal/tmux"
	"claudetop/internal/ui"
)

func main() {
	// Verify tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Fprintln(os.Stderr, "claudetop: tmux is required but not found in PATH")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudetop: config error: %v\n", err)
		os.Exit(1)
	}

	if err := config.EnsureDir(); err != nil {
		fmt.Fprintf(os.Stderr, "claudetop: cannot create config dir: %v\n", err)
		os.Exit(1)
	}

	st, err := state.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudetop: state error: %v\n", err)
		os.Exit(1)
	}

	reconnect(st)

	m := ui.New(cfg, st)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

// reconnect checks each persisted session against live tmux sessions.
// Sessions without a live tmux counterpart are marked as dead.
func reconnect(st *state.State) {
	live, err := tmux.LiveSessions()
	if err != nil {
		// tmux may not be running yet; all sessions will appear dead
		for _, s := range st.Sessions {
			s.Dead = true
		}
		return
	}

	liveSet := map[string]bool{}
	for _, id := range live {
		liveSet[id] = true
	}

	for _, s := range st.Sessions {
		if liveSet[s.ID] {
			s.Status = session.StatusDone
			s.Dead = false
		} else {
			s.Dead = true
		}
	}
}
