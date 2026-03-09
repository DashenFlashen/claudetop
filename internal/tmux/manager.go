package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

const prefix = "ct-"

// SessionName converts a session ID to a tmux session name.
func SessionName(id string) string {
	return prefix + id
}

// Create creates a new detached tmux session and starts claude in it.
func Create(id, rootDir string) error {
	name := SessionName(id)

	out, err := exec.Command("tmux", "new-session", "-d", "-s", name, "-c", rootDir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux new-session: %w\n%s", err, out)
	}

	out, err = exec.Command("tmux", "send-keys", "-t", name, "claude", "Enter").CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux send-keys claude: %w\n%s", err, out)
	}

	return nil
}

// Kill destroys a tmux session.
func Kill(id string) error {
	name := SessionName(id)
	out, err := exec.Command("tmux", "kill-session", "-t", name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux kill-session: %w\n%s", err, out)
	}
	return nil
}

// Exists reports whether the tmux session is alive.
func Exists(id string) bool {
	return exec.Command("tmux", "has-session", "-t", SessionName(id)).Run() == nil
}

// CapturePane reads the current visible content of a tmux pane.
func CapturePane(id string) (string, error) {
	name := SessionName(id)
	out, err := exec.Command("tmux", "capture-pane", "-t", name, "-p", "-e").Output()
	if err != nil {
		return "", fmt.Errorf("tmux capture-pane: %w", err)
	}
	return string(out), nil
}

// SendKeys forwards a named key to a tmux session (e.g. "Enter", "BSpace").
func SendKeys(id string, key string) error {
	name := SessionName(id)
	out, err := exec.Command("tmux", "send-keys", "-t", name, key, "").CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux send-keys: %w\n%s", err, out)
	}
	return nil
}

// SendLiteralKey sends a literal character string to a tmux session.
func SendLiteralKey(id string, key string) error {
	name := SessionName(id)
	out, err := exec.Command("tmux", "send-keys", "-t", name, "-l", key).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux send-keys -l: %w\n%s", err, out)
	}
	return nil
}

// LiveSessions returns IDs of all ct- prefixed tmux sessions.
func LiveSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// tmux returns non-zero when there are no sessions — this is not an error
		outStr := strings.TrimSpace(string(out))
		if strings.Contains(outStr, "no server running") ||
			strings.Contains(outStr, "no sessions") ||
			outStr == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("tmux list-sessions: %w\n%s", err, out)
	}

	var ids []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			ids = append(ids, strings.TrimPrefix(line, prefix))
		}
	}
	return ids, nil
}
