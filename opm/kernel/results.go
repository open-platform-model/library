package kernel

import (
	"github.com/open-platform-model/library/opm/compile"
)

// MatchPlan is the result of [Kernel.Match].
type MatchPlan = compile.MatchPlan

// CompileResult is the result of [Kernel.Compile].
type CompileResult = compile.CompileResult

// PlanResult is the result of [Kernel.Plan].
type PlanResult struct {
	// MatchPlan is the raw match outcome, exposed so callers can inspect
	// per-(component, transformer) details without re-running Match.
	MatchPlan *MatchPlan

	// Components is the per-component summary, sorted by name.
	Components []compile.ComponentSummary

	// Unmatched is the list of component FQNs with no matching transformer.
	Unmatched []string

	// Warnings is a list of human-readable advisory messages (e.g.
	// unhandled traits). A non-empty Warnings slice does NOT indicate
	// failure.
	Warnings []string
}
