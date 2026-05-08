// Package v1alpha2 ships the OPM v1alpha2 CUE schema as an embedded
// filesystem so the kernel can validate artifacts deterministically without a
// network registry round-trip.
//
// This Go package coexists with the CUE package of the same name in this
// directory. CUE's own loader does not look at .go files; Go's `embed`
// directive does not look at .cue files unless explicitly listed. The two
// stay independent.
package v1alpha2

import "embed"

// Schema holds every CUE source file that defines the v1alpha2 schema, plus
// the cue.mod/module.cue manifest. The embed pattern is intentionally narrow
// so docs/, INDEX.md, and other non-schema content stay out of the binary.
//
//go:embed *.cue cue.mod/module.cue
var Schema embed.FS
