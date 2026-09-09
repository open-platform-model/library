package kernel

// Source is one values input for every values-taking kernel entry:
// [Kernel.ValidateConfigDetailed], the trailing values sources of
// [Kernel.AcquireInstanceFromDir] and [InstanceInput.Values] on
// [Kernel.SynthesizeInstance].
//
// A Source pairs a values payload, as CUE source bytes, with its stable
// origin. It carries no [cue.Value]: a value is bound to the context that
// built it, and every kernel operation builds in a context of its own, so the
// kernel compiles Data with [cue.Filename](Origin) in the context of the
// operation that uses it, where the source meets the schema it is checked
// against. Per-position diagnostics then carry Origin through
// [token.Pos.Filename] without a Go-typed wrapper around CUE's error
// attribution. It carries no display label either: presentation is outside
// the kernel's contract, and Origin is what CUE positions report.
//
// [Kernel.LoadSourceFromFile] and [Kernel.LoadSourceFromBytes] construct a
// Source after checking that the payload parses; a hand-built Source is
// compiled the same way and reports a syntax error in the operation that
// uses it.
type Source struct {
	// Origin is the stable identifier for machine-readable correlation (file
	// path, K8s object reference, composition input key). It is the filename
	// the kernel compiles Data under, so error positions report Origin via
	// [token.Pos.Filename]. An absolute path naming an existing file marks a
	// file-backed source: the kernel loads it through cue/load at that file's
	// directory, so its imports resolve as they do for any CUE file.
	Origin string

	// Data is the values payload as CUE source. It is compiled where it is
	// used, never ahead of use; an empty payload is "no values supplied".
	Data []byte
}
