package config

import (
	"os"
	"path/filepath"
)

// Profile maps abstract tiers to concrete model identifiers.
type Profile struct {
	Fast     string `toml:"fast"`
	Standard string `toml:"standard"`
	Advanced string `toml:"advanced"`
}

// Config holds the parsed .beads-plan.toml configuration.
type Config struct {
	DefaultProfile string             `toml:"default_profile"`
	Profiles       map[string]Profile `toml:"profile"`
}

// ResolveModel returns the concrete model for a tier using the given profile.
// Returns empty string if the profile or tier mapping doesn't exist.
func (c *Config) ResolveModel(profileName, tier string) string {
	if c == nil || c.Profiles == nil {
		return ""
	}
	p, ok := c.Profiles[profileName]
	if !ok {
		return ""
	}
	switch tier {
	case "fast":
		return p.Fast
	case "standard":
		return p.Standard
	case "advanced":
		return p.Advanced
	default:
		return ""
	}
}

// FindConfigFile searches for .beads-plan.toml in standard locations.
// Search order: current directory, parent directories up to git root,
// then ~/.config/beads-plan/config.toml.
// Returns the path to the first found config, or empty string if none exists.
func FindConfigFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return findUserConfig()
	}

	for {
		candidate := filepath.Join(dir, ".beads-plan.toml")
		if fileExists(candidate) {
			return candidate
		}
		// Stop at git root
		if fileExists(filepath.Join(dir, ".git")) {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}

	return findUserConfig()
}

func findUserConfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(home, ".config", "beads-plan", "config.toml")
	if fileExists(candidate) {
		return candidate
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
