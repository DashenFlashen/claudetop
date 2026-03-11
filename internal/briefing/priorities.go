package briefing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"claudetop/internal/config"
)

// WritePriorities saves today's priorities to ~/.claudetop/today.md.
// text is split on commas; each non-empty item becomes a numbered list entry.
// Returns the path written.
func WritePriorities(text string) (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	path := filepath.Join(dir, "today.md")

	var items []string
	for _, part := range strings.Split(text, ",") {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Priorities — %s\n\n", time.Now().Format("2006-01-02")))
	if len(items) == 0 {
		sb.WriteString(strings.TrimSpace(text) + "\n")
	} else {
		for i, item := range items {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
		}
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("write today.md: %w", err)
	}
	return path, nil
}
