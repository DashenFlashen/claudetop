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
}

func TestLoadFromFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	dir := filepath.Join(tmp, ".claudetop")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	content := "[general]\nroot_dir = \"/tmp/repos\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.General.RootDir != "/tmp/repos" {
		t.Errorf("expected /tmp/repos, got %q", cfg.General.RootDir)
	}
}

func TestLoadSkills(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	dir := filepath.Join(tmp, ".claudetop")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	content := `[general]
root_dir = "/tmp/repos"

[[skills]]
key = "o"
name = "Overview"
description = "Daily overview"
command = "claude -p /overview"
mode = "output"

[[skills]]
key = "g"
name = "Grafana"
description = "Interactive investigation"
command = "claude"
mode = "interactive"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(cfg.Skills))
	}
	if cfg.Skills[0].Key != "o" || cfg.Skills[0].Mode != "output" {
		t.Errorf("unexpected first skill: %+v", cfg.Skills[0])
	}
	if cfg.Skills[1].Key != "g" || cfg.Skills[1].Mode != "interactive" {
		t.Errorf("unexpected second skill: %+v", cfg.Skills[1])
	}
}
