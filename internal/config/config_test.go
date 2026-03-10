package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.General.RootDir != tmp {
		t.Errorf("expected RootDir=%q, got %q", tmp, cfg.General.RootDir)
	}
	if !cfg.General.AutoNameSessions {
		t.Error("expected AutoNameSessions=true by default")
	}
}

func TestLoadFromFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	dir := filepath.Join(tmp, ".claudetop")
	os.MkdirAll(dir, 0755)

	content := "[general]\nroot_dir = \"/tmp/repos\"\nauto_name_sessions = false\n"
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.General.RootDir != "/tmp/repos" {
		t.Errorf("expected /tmp/repos, got %q", cfg.General.RootDir)
	}
	if cfg.General.AutoNameSessions {
		t.Error("expected AutoNameSessions=false")
	}
}
