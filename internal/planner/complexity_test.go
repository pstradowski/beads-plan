package planner

import (
	"testing"

	"github.com/pstradowski/beads-plan/internal/config"
)

func TestAssessComplexityLow(t *testing.T) {
	tests := []struct {
		title string
	}{
		{"Add .gitignore for build artifacts"},
		{"Update configuration file"},
		{"Create project scaffold and boilerplate"},
		{"Rename variable in helper"},
	}
	for _, tt := range tests {
		c := AssessComplexity(tt.title, "", "")
		if c != ComplexityLow {
			t.Errorf("AssessComplexity(%q) = %s, want low", tt.title, c)
		}
	}
}

func TestAssessComplexityMedium(t *testing.T) {
	tests := []struct {
		title string
	}{
		{"Implement HTTP handler for user creation"},
		{"Add database query for fetching orders"},
		{"Write unit tests for parser"},
	}
	for _, tt := range tests {
		c := AssessComplexity(tt.title, "", "")
		if c != ComplexityMedium {
			t.Errorf("AssessComplexity(%q) = %s, want medium", tt.title, c)
		}
	}
}

func TestAssessComplexityHigh(t *testing.T) {
	tests := []struct {
		title  string
		spec   string
		design string
	}{
		{"Refactor authentication and authorization system", "", ""},
		{"Design distributed orchestration layer", "", ""},
		{"Implement cross-cutting security middleware", "integration with API and database services", ""},
	}
	for _, tt := range tests {
		c := AssessComplexity(tt.title, tt.spec, tt.design)
		if c != ComplexityHigh {
			t.Errorf("AssessComplexity(%q, %q, %q) = %s, want high", tt.title, tt.spec, tt.design, c)
		}
	}
}

func TestComplexityToTier(t *testing.T) {
	tests := []struct {
		c    Complexity
		want Tier
	}{
		{ComplexityLow, TierFast},
		{ComplexityMedium, TierStandard},
		{ComplexityHigh, TierAdvanced},
	}
	for _, tt := range tests {
		got := ComplexityToTier(tt.c)
		if got != tt.want {
			t.Errorf("ComplexityToTier(%s) = %s, want %s", tt.c, got, tt.want)
		}
	}
}

func TestAssignTierWithProfile(t *testing.T) {
	profile := &config.Profile{
		Fast:     "gpt-4o-mini",
		Standard: "gpt-4o",
		Advanced: "o3",
	}

	result := AssignTier("Update config file", "", "", profile)
	if result.Tier != TierFast {
		t.Errorf("expected tier=fast, got %s", result.Tier)
	}
	if result.Model != "gpt-4o-mini" {
		t.Errorf("expected model=gpt-4o-mini, got %s", result.Model)
	}
}

func TestAssignTierWithoutProfile(t *testing.T) {
	result := AssignTier("Implement HTTP handler for user service", "", "", nil)
	if result.Tier != TierStandard {
		t.Errorf("expected tier=standard, got %s", result.Tier)
	}
	if result.Model != "" {
		t.Errorf("expected empty model without profile, got %s", result.Model)
	}
}

func TestAssignTierAdvanced(t *testing.T) {
	profile := &config.Profile{
		Fast:     "small",
		Standard: "medium",
		Advanced: "large",
	}

	result := AssignTier("Refactor distributed authentication", "integration with API and database", "", profile)
	if result.Tier != TierAdvanced {
		t.Errorf("expected tier=advanced, got %s", result.Tier)
	}
	if result.Model != "large" {
		t.Errorf("expected model=large, got %s", result.Model)
	}
}

func TestAssessComplexityUsesContext(t *testing.T) {
	// Title alone looks low, but spec context pushes it higher
	title := "Update config"
	c := AssessComplexity(title, "", "")
	if c != ComplexityLow {
		t.Errorf("without context: expected low, got %s", c)
	}

	// Adding integration-heavy spec context should raise it
	c = AssessComplexity(title, "requires API endpoint integration with database service and middleware handler", "")
	if c == ComplexityLow {
		t.Errorf("with integration context: should not be low, got %s", c)
	}
}
