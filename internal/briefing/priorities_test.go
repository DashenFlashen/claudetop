package briefing_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"claudetop/internal/briefing"
)

func TestWritePriorities(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path, err := briefing.WritePriorities("Resolve annotation timeout, Move CSV forward, Process inbox")
	if err != nil {
		t.Fatalf("WritePriorities: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	today := time.Now().Format("2006-01-02")
	if !strings.Contains(content, today) {
		t.Errorf("expected today's date %q in output, got:\n%s", today, content)
	}
	if !strings.Contains(content, "1. Resolve annotation timeout") {
		t.Errorf("expected numbered item 1, got:\n%s", content)
	}
	if !strings.Contains(content, "2. Move CSV forward") {
		t.Errorf("expected numbered item 2, got:\n%s", content)
	}
	if !strings.Contains(content, "3. Process inbox") {
		t.Errorf("expected numbered item 3, got:\n%s", content)
	}
}

func TestWritePrioritiesEmptyLinesSkipped(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path, err := briefing.WritePriorities("item one, , item two")
	if err != nil {
		t.Fatalf("WritePriorities: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if strings.Contains(content, "2. ") && !strings.Contains(content, "2. item two") {
		t.Errorf("empty item should be skipped: %s", content)
	}
}
