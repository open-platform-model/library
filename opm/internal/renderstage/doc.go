// Package renderstage assembles the single-build render module (enhancement
// 0019 D9): it reads the two committed cue.mod/module.cue resolutions the
// render inputs carry, promotes them into the render module's dependency list
// (D13), checks that list for OPM-namespace coverage (the D13 refusal
// invariant), compares the two committed lists for catalog version skew
// (D7/D18), stages the generated render module into a directory, and builds it
// once in a caller-supplied cue.Context (D8).
//
// The directory holds only the generated module: its cue.mod pair and the
// glue. An on-disk input is referenced in place through its local-module.cue
// replacement; an overlay-mode input is re-keyed under the directory that
// replacement names and served to the build through load.Config.Overlay, so
// no file of it is written.
//
// It is internal: the kernel's Render entry point owns the public types and
// the decode of the built value. Nothing here performs registry I/O of its own
// beyond the one cue/load build; the dependency list is string-level modfile
// mechanics over the two files the inputs already carry.
package renderstage
