package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".beads-plan.toml")
	content := `
default_profile = "anthropic"

[profile.anthropic]
fast = "haiku"
standard = "sonnet"
advanced = "opus"

[profile.openai]
fast = "gpt-4o-mini"
standard = "gpt-4o"
advanced = "o3"
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultProfile != "anthropic" {
		t.Errorf("expected default_profile=anthropic, got %s", cfg.DefaultProfile)
	}
	if len(cfg.Profiles) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(cfg.Profiles))
	}
	if cfg.Profiles["anthropic"].Advanced != "opus" {
		t.Errorf("expected anthropic.advanced=opus, got %s", cfg.Profiles["anthropic"].Advanced)
	}
	if cfg.Profiles["openai"].Fast != "gpt-4o-mini" {
		t.Errorf("expected openai.fast=gpt-4o-mini, got %s", cfg.Profiles["openai"].Fast)
	}
}

func TestLoadEmptyPath(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config for empty path")
	}
}

func TestLoadMalformedTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".beads-plan.toml")
	os.WriteFile(path, []byte("this is not [valid toml"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for malformed TOML")
	}
}

func TestLoadPartialProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".beads-plan.toml")
	content := `
[profile.minimal]
fast = "small-model"
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Profiles["minimal"].Fast != "small-model" {
		t.Errorf("expected fast=small-model, got %s", cfg.Profiles["minimal"].Fast)
	}
	if cfg.Profiles["minimal"].Standard != "" {
		t.Errorf("expected empty standard, got %s", cfg.Profiles["minimal"].Standard)
	}
}

func TestResolveModel(t *testing.T) {
	cfg := &Config{
		Profiles: map[string]Profile{
			"test": {Fast: "f", Standard: "s", Advanced: "a"},
		},
	}

	tests := []struct {
		profile, tier, want string
	}{
		{"test", "fast", "f"},
		{"test", "standard", "s"},
		{"test", "advanced", "a"},
		{"test", "unknown", ""},
		{"missing", "fast", ""},
	}
	for _, tt := range tests {
		got := cfg.ResolveModel(tt.profile, tt.tier)
		if got != tt.want {
			t.Errorf("ResolveModel(%q, %q) = %q, want %q", tt.profile, tt.tier, got, tt.want)
		}
	}
}

func TestResolveModelNilConfig(t *testing.T) {
	var cfg *Config
	if got := cfg.ResolveModel("any", "fast"); got != "" {
		t.Errorf("expected empty for nil config, got %q", got)
	}
}

func TestActiveProfile(t *testing.T) {
	cfg := &Config{DefaultProfile: "default-prof"}

	// Flag takes precedence
	if got := ActiveProfile(cfg, "flag-prof"); got != "flag-prof" {
		t.Errorf("expected flag-prof, got %s", got)
	}
	// Falls back to config default
	if got := ActiveProfile(cfg, ""); got != "default-prof" {
		t.Errorf("expected default-prof, got %s", got)
	}
	// Nil config, no flag
	if got := ActiveProfile(nil, ""); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestGetProfile(t *testing.T) {
	cfg := &Config{
		Profiles: map[string]Profile{
			"exists": {Fast: "f"},
		},
	}

	// Existing profile
	p, err := GetProfile(cfg, "exists")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Fast != "f" {
		t.Errorf("expected fast=f, got %s", p.Fast)
	}

	// Missing profile
	_, err = GetProfile(cfg, "missing")
	if err == nil {
		t.Error("expected error for missing profile")
	}

	// Empty name (tier-only mode)
	p, err = GetProfile(cfg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Error("expected nil profile for empty name")
	}

	// Nil config with profile requested
	_, err = GetProfile(nil, "any")
	if err == nil {
		t.Error("expected error for nil config with profile name")
	}
}

func resolveSymlinks(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

func TestFindConfigFile(t *testing.T) {
	// Create a temp dir with .git and config
	dir := resolveSymlinks(t.TempDir())
	os.Mkdir(filepath.Join(dir, ".git"), 0755)
	configPath := filepath.Join(dir, ".beads-plan.toml")
	os.WriteFile(configPath, []byte("default_profile = \"test\""), 0644)

	// Change to the temp dir
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	found := FindConfigFile()
	if found != configPath {
		t.Errorf("expected %s, got %s", configPath, found)
	}
}

func TestFindConfigFileSubdir(t *testing.T) {
	// Config in parent, cwd in child
	dir := resolveSymlinks(t.TempDir())
	os.Mkdir(filepath.Join(dir, ".git"), 0755)
	configPath := filepath.Join(dir, ".beads-plan.toml")
	os.WriteFile(configPath, []byte("default_profile = \"test\""), 0644)

	subdir := filepath.Join(dir, "sub", "deep")
	os.MkdirAll(subdir, 0755)

	orig, _ := os.Getwd()
	os.Chdir(subdir)
	defer os.Chdir(orig)

	found := FindConfigFile()
	if found != configPath {
		t.Errorf("expected %s, got %s", configPath, found)
	}
}

func TestFindConfigFileNone(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".git"), 0755)
	// No config file

	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	found := FindConfigFile()
	// Should return empty or user config (which likely doesn't exist in test)
	if found != "" {
		// Only fail if it found something unexpected
		if filepath.Base(found) != "config.toml" {
			t.Errorf("unexpected config found: %s", found)
		}
	}
}
