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
// An input's own cue.mod/local-module.cue (a developer's redirection of a
// dependency to a directory or another module) is read in either mode
// ([ReadLocalModFile]) and, when the caller enables local replacements,
// promoted into the render module's main-module view under the precedence
// dependencies get: the platform's replacements whole, the instance's only
// for paths the platform's list does not name. A replaced path the promoted
// list lacks is listed with a placeholder version of its major, so the
// coverage invariant holds; the honoured set is reported as
// [ReplacementRow] values on [Staged]. With local replacements off, an input
// whose file carries a replacement is refused before anything is written.
//
// It is internal: the kernel's Render entry point owns the public types and
// the decode of the built value. Nothing here performs registry I/O of its own
// beyond the one cue/load build; the dependency list is string-level modfile
// mechanics over the two files the inputs already carry.
package renderstage
