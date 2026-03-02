package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Load reads and parses a .beads-plan.toml config file.
// Returns nil config (not error) if path is empty (no config found).
func Load(path string) (*Config, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return &cfg, nil
}

// LoadDefault finds and loads the config file from standard locations.
func LoadDefault() (*Config, error) {
	path := FindConfigFile()
	return Load(path)
}

// ActiveProfile returns the profile name to use, given a CLI flag override.
// Priority: flag > config default > empty (tier-only mode).
func ActiveProfile(cfg *Config, flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if cfg != nil && cfg.DefaultProfile != "" {
		return cfg.DefaultProfile
	}
	return ""
}

// GetProfile returns the named profile from config.
// Returns nil if config is nil or profile doesn't exist.
func GetProfile(cfg *Config, name string) (*Profile, error) {
	if name == "" {
		return nil, nil
	}
	if cfg == nil || cfg.Profiles == nil {
		return nil, fmt.Errorf("profile %q requested but no config file found", name)
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		available := make([]string, 0, len(cfg.Profiles))
		for k := range cfg.Profiles {
			available = append(available, k)
		}
		return nil, fmt.Errorf("profile %q not found in config (available: %v)", name, available)
	}
	return &p, nil
}
