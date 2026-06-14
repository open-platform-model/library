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
// the platform value, so the filled #composedTransformers / #matchers share
// one context with the platform (cross-context values cannot be filled
// together — see design.md D2 spike outcome).
type CueContextOwner interface {
	CueContext() *cue.Context
}

// MaterializedPlatform is the sealed, post-realization view of a #Platform.
// It is produced by [Materialize] and consumed (in the follow-up match
// rewrite) by the matcher. Once returned it is treated as immutable.
//
// Package answers the same LookupPath calls the matcher already makes against
// a platform value — schema.ComposedTransformers, schema.MatchersResources,
// schema.MatchersTraits — with the kernel-filled index present. That keeps
// the match-signature swap a minimal diff: the matcher reads the same paths,
// now populated.
type MaterializedPlatform struct {
	// Source is the *platform.Platform this view was realized from. Held for
	// diagnostics (a MaterializeError can name the originating platform) and
	// for the later MissingFQN.alternatives lookups.
	Source *platform.Platform

	// Package is Source.Package with #composedTransformers and #matchers
	// filled by Materialize. It is built with the owner's *cue.Context.
	//
	// WARNING: do NOT read a transformer's #transform out of Package to render
	// its output. Package is a *closed* c.#Platform value, and FillPath-ing the
	// composed map into a closed, independently-built value corrupts the lazy
	// in-expression resolution of output-local hidden fields in the transformers
	// (a CUE Go-API closedness bug — see
	// docs/design/transformer-output-hidden-field-scope-bug.md §12). The matcher
	// reads only FQNs/labels off Package, which are unaffected; the executor MUST
	// read transforms from Composed instead.
	Package cue.Value

	// Composed is the open #composedTransformers map (FQN → #ComponentTransformer)
	// as produced by indexCatalogs, BEFORE it is filled onto the closed Package.
	// Reading a transform from this open value renders output-local hidden fields
	// correctly; reading it from Package does not (see Package's warning). The
	// executor sources every #transform from here.
	Composed cue.Value

	// Resolved maps each enabled subscription path to the bare SemVer that
	// Materialize recorded for it. When a filter selects several versions
	// (all of which are pulled and indexed), this is the highest survivor.
	// Diagnostic-only: callers SHOULD log it but MUST NOT branch behavior on it.
	Resolved map[string]string
}
