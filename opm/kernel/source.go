package kernel

import (
	"cuelang.org/go/cue"
)

// Source is one values input for [Kernel.ValidateConfigDetailed] and for
// [WithValues] on [Kernel.AcquireInstanceFromDir].
//
// A Source pairs a values payload with its stable origin so that
// per-position diagnostics flowing out of CUE error trees carry the
// originating source's filename. The library does not invent a Go-typed
// wrapper around CUE's error attribution — instead, it relies on
// [token.Pos.Filename], populated from [cue.Filename] at compile time. It
// carries no display label: presentation is outside the kernel's contract,
// and Origin is what CUE positions report.
type Source struct {
	// Value is the raw values payload for this source.
	//
	// Value MUST have been compiled with [cue.Filename](Origin) for
	// per-source attribution to flow into errors. Use
	// [Kernel.LoadSourceFromFile] or [Kernel.LoadSourceFromBytes] to
	// construct a Source whose Value satisfies this contract automatically.
	// Hand-built Sources MUST set the filename themselves when compiling.
	Value cue.Value

	// Origin is the stable identifier for machine-readable correlation
	// (file path, K8s object reference, composition input key). It MUST
	// match the [cue.Filename] used when Value was compiled, so error
	// positions report Origin via [token.Pos.Filename].
	Origin string
}
