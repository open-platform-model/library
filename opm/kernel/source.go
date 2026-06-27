package kernel

import (
	"cuelang.org/go/cue"
)

// Source is one labeled values input for [Kernel.ValidateConfigDetailed].
//
// A Source pairs a values payload with caller-supplied identity so that
// per-position diagnostics flowing out of CUE error trees carry the
// originating source's filename. The library does not invent a Go-typed
// wrapper around CUE's error attribution — instead, it relies on
// [token.Pos.Filename], populated from [cue.Filename] at compile time.
type Source struct {
	// Value is the raw values payload for this source.
	//
	// Value MUST have been compiled with [cue.Filename](Origin) for
	// per-source attribution to flow into errors. Use one of
	// [Kernel.LoadSourceFromFile], [Kernel.LoadSourceFromBytes], or
	// [Kernel.LoadSourceFromString] to construct a Source whose Value
	// satisfies this contract automatically. Hand-built Sources MUST set
	// the filename themselves when compiling.
	Value cue.Value

	// Name is the human-friendly label shown in UI. Examples:
	// "user values", "ConfigMap/foo", "instance overlay".
	Name string

	// Origin is the stable identifier for machine-readable correlation
	// (file path, K8s object reference, composition input key). It MUST
	// match the [cue.Filename] used when Value was compiled, so error
	// positions report Origin via [token.Pos.Filename].
	Origin string
}

// validateConfig is the unexported configuration struct accumulated from
// [ValidateOption] values passed to [Kernel.ValidateConfigDetailed]. It is
// the canonical home for orthogonal knobs (concreteness, future strict
// modes); each public option mutates one field.
type validateConfig struct {
	// partial, when true, skips the [cue.Concrete] check on the merged
	// value. The closed-schema [walkDisallowed] pass still runs.
	partial bool
}

// ValidateOption configures [Kernel.ValidateConfigDetailed]. Options compose
// via the functional-options pattern; new options can be added in MINOR
// instances without breaking existing call sites.
//
// The type is named ValidateOption (rather than the more terse Option) to
// avoid collision with [Option], the kernel-construction option type.
type ValidateOption func(*validateConfig)

// Partial returns a [ValidateOption] that skips the concreteness check on
// the merged value passed to [Kernel.ValidateConfigDetailed].
//
// With Partial: type errors, constraint violations on fields that ARE set,
// and disallowed-field errors (under closed schemas) all still surface.
// Missing required fields do NOT surface — concrete validation is the only
// pass that flags them, and Partial disables it.
//
// Use Partial for callers that intentionally validate incomplete drafts:
// CLI lint subcommands, IDE/LSP live feedback, admission webhooks that see
// only one of several values sources.
func Partial() ValidateOption {
	return func(c *validateConfig) {
		c.partial = true
	}
}
