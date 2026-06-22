package materialize

import (
	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/platform"
)

// CueContextOwner is the minimal context-owner interface [Materialize]
// accepts. *kernel.Kernel satisfies it; tests may pass any value exposing a
// *cue.Context. Keeping the interface here frees the materialize package from
// importing opm/kernel.
//
// The owner's *cue.Context is used to build every pulled catalog AND to read
// the platform value, so the native Transformers / Matchers surfaces share one
// context with the platform (cross-context values cannot be unified or filled
// together).
type CueContextOwner interface {
	CueContext() *cue.Context
}

// MaterializedPlatform is the sealed, post-realization view of a #Platform.
// It is produced by [Materialize] and consumed by the matcher and executor.
// Once returned it is treated as immutable and is safe for concurrent
// read-only consumption.
//
// The composed transformer map and the matcher reverse index are exposed as
// first-class fields built natively in the owner *cue.Context. They are NOT
// filled onto the closed c.#Platform value: doing so corrupts the lazy
// in-expression resolution of output-local hidden fields in transformer
// #transforms (a CUE Go-API closedness bug — see ADR-003 and
// docs/design/transformer-output-hidden-field-scope-bug.md §12). Federating
// the native surfaces instead of collapsing them into the closed platform
// removes that footgun by construction: there is no closed twin to misread.
type MaterializedPlatform struct {
	// Source is the *platform.Platform this view was realized from. Its
	// Source.Package is the closed c.#Platform spec, reachable for #registry,
	// metadata, and diagnostics (a MaterializeError can name the originating
	// platform; MissingFQN.alternatives may inspect it). It is NOT filled with
	// the composed map or matcher index.
	Source *platform.Platform

	// Transformers is the open #composedTransformers map (FQN →
	// #ComponentTransformer) produced by indexCatalogs in the owner context.
	// It is the canonical surface for reading a transformer's #transform:
	// because it is built natively (never filled into a closed value), reading
	// a #transform off it — including output-local hidden fields — renders
	// concrete. Multi-version composition is preserved: each selected version's
	// transformers are indexed under their distinct version-bearing FQNs.
	Transformers cue.Value

	// Matchers is the open #matchers reverse index ({resources, traits}:
	// primitive FQN → [transformers]) produced by indexCatalogs in the owner
	// context. The matcher looks up demanded FQNs in Matchers.resources /
	// Matchers.traits to find candidate transformers.
	Matchers cue.Value

	// Resolved maps each enabled subscription path to the bare SemVer that
	// Materialize recorded for it. When a filter selects several versions
	// (all of which are pulled and indexed), this is the highest survivor.
	// Diagnostic-only: callers SHOULD log it but MUST NOT branch behavior on it.
	Resolved map[string]string
}
