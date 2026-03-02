package planner

import (
	"github.com/pstradowski/beads-plan/internal/config"
)

// Tier represents the capability tier for task execution.
type Tier string

const (
	TierFast     Tier = "fast"
	TierStandard Tier = "standard"
	TierAdvanced Tier = "advanced"
)

// TierAssignment holds the result of complexity assessment and tier mapping.
type TierAssignment struct {
	Complexity Complexity
	Tier       Tier
	Model      string // resolved model string, empty if no profile
}

// ComplexityToTier maps complexity level to capability tier.
func ComplexityToTier(c Complexity) Tier {
	switch c {
	case ComplexityLow:
		return TierFast
	case ComplexityHigh:
		return TierAdvanced
	default:
		return TierStandard
	}
}

// AssignTier assesses complexity and maps to tier, optionally resolving
// the concrete model via a provider profile.
func AssignTier(title string, specContext string, designContext string, profile *config.Profile) TierAssignment {
	complexity := AssessComplexity(title, specContext, designContext)
	tier := ComplexityToTier(complexity)

	result := TierAssignment{
		Complexity: complexity,
		Tier:       tier,
	}

	if profile != nil {
		switch tier {
		case TierFast:
			result.Model = profile.Fast
		case TierStandard:
			result.Model = profile.Standard
		case TierAdvanced:
			result.Model = profile.Advanced
		}
	}

	return result
}
