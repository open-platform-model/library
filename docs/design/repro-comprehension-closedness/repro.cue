// Self-contained minimal reproduction of the alpha.2+ closedness regression.
// No registry, no cross-package imports.
//
//   cue vet -c=false ./...
//     v0.17.0-alpha.1 -> clean (exit 0)
//     v0.17.0-alpha.2 / alpha.3 / master(2026-06-16) -> "field not allowed"
//
// First bad commit (git bisect): 339485ddf008a5b536714a5ed0fc625769a0f1a1
//   "internal/core/adt: dependency-tracking comprehension pushdown".
//
// The three essential ingredients (each verified necessary by ablation):
//   1. `spec` is populated by EMBEDDING a referenced struct (`spec: {_allFields}`),
//      not by writing the fields directly. Direct fields do NOT trigger it.
//   2. an embedded field is typed by a closed definition that itself carries a
//      nested optional (`statelessWorkload: #Schema`, `#Schema.scaling?: #Inner`).
//   3. a self-referential CONDITIONAL comprehension at the same struct level
//      reads a field of the struct under construction
//      (`if spec.statelessWorkload.scaling != _|_ { ... }`). An *unconditional*
//      copy does NOT trigger it; the `if` guard is required.
// `close()` is NOT required (recursive closedness of the enclosing #definition
// suffices); the `spec.`-prefixed self-path is NOT required (a local reference
// also triggers); the comprehension-over-a-map is NOT required (inline fields
// trigger it too).
package repro

#Inner: {count: int}
#Schema: {
	a:        string
	scaling?: #Inner
}
#Component: {
	_allFields: {
		scaling:           #Inner
		statelessWorkload: #Schema
	}
	spec: {_allFields}
}
#SW: #Component & {
	spec: {
		statelessWorkload: #Schema
		if spec.statelessWorkload.scaling != _|_ {
			scaling: spec.statelessWorkload.scaling
		}
	}
}
web: #SW & {
	spec: statelessWorkload: {a: "x", scaling: count: 3}
}
