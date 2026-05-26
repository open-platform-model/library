// Package core ships the OPM core CUE schema as an embedded filesystem so
// the kernel can validate artifacts deterministically without a network
// registry round-trip.
//
// This Go package coexists with the CUE module rooted at this directory
// (apis/core/cue.mod/module.cue declares opmodel.dev/core@v0). CUE's own
// loader does not look at .go files; Go's `embed` directive does not look at
// .cue files unless explicitly listed. The two stay independent.
//
// The schema is single-version (flat layout): every *.cue file at this
// directory is part of the `core` package. A breaking schema revision
// rotates the CUE module major (v0 → v1) — not a sibling subdirectory.
package core

import "embed"

// Schema holds every CUE source file that defines the core schema, plus
// the cue.mod/module.cue manifest. The embed pattern is intentionally narrow
// so INDEX.md and other non-schema content stay out of the binary.
//
//go:embed cue.mod/module.cue *.cue
var Schema embed.FS
