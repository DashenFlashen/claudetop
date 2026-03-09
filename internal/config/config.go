package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	General GeneralConfig `toml:"general"`
}

type GeneralConfig struct {
	RootDir string `toml:"root_dir"`
}

// Dir returns the ~/.claudetop directory path.
func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claudetop")
}

// Path returns the config file path.
func Path() string {
	return filepath.Join(Dir(), "config.toml")
}

// Load reads the config file, returning defaults if missing.
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}

	cfg := &Config{
		General: GeneralConfig{
			RootDir: home,
		},
	}

	path := Path()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	return cfg, nil
}

// EnsureDir creates ~/.claudetop if it doesn't exist.
func EnsureDir() error {
	return os.MkdirAll(Dir(), 0755)
}
