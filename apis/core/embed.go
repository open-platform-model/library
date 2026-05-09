// Package core ships the OPM core CUE schema as an embedded filesystem so
// the kernel can validate artifacts deterministically without a network
// registry round-trip.
//
// This Go package coexists with the CUE module rooted at this directory
// (apis/core/cue.mod/module.cue declares opmodel.dev/core@v1). CUE's own
// loader does not look at .go files; Go's `embed` directive does not look at
// .cue files unless explicitly listed. The two stay independent.
//
// The embed pattern covers cue.mod/module.cue (the manifest) plus every
// versioned schema package below this directory. Add a new line to the
// directive when introducing additional version subpackages.
package core

import "embed"

// Schema holds every CUE source file that defines the core schema, plus
// the cue.mod/module.cue manifest. The embed pattern is intentionally narrow
// so docs/, INDEX.md, and other non-schema content stay out of the binary.
//
//go:embed cue.mod/module.cue v1alpha2/*.cue
var Schema embed.FS
