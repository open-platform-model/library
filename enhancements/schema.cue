// Schema for enhancements/NNN-<slug>/config.yaml.
//
// Repo-internal — NOT part of apis/core. Lives here so the contract sits
// next to the data it validates. Use
//   cue vet -d '#EnhancementConfig' enhancements/schema.cue <config.yaml>
// to validate a single file, or `task enhancements:vet` for the full sweep.
package enhancements

import "strings"

#DateStr: =~"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"
#IDStr:   =~"^[0-9]{3}$"
#SlugStr: =~"^[a-z0-9]([a-z0-9-]*[a-z0-9])?$"

// Title bounded for one-line display in `task enhancements:list`.
// Slug captures the long form; README/01/02/… docs carry full prose.
#TitleStr: string & strings.MinRunes(1) & strings.MaxRunes(50)

#Status:       "draft" | "accepted" | "implemented" | "superseded"
#ImplStatus:   "not-started" | "in-progress" | "partial" | "complete"
#SemverImpact: "major" | "minor" | "none"

// OpenSpec change slug — dated kebab prefix, e.g. "2026-05-08-add-kernel-struct".
// `task enhancements:vet` separately checks each listed slice exists under
// openspec/changes/archive/.
#SliceStr: =~"^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9]([a-z0-9-]*[a-z0-9])?$"

#EnhancementConfig: {
	id!:      #IDStr
	slug!:    #SlugStr
	title!:   #TitleStr
	status!:  #Status
	created!: #DateStr
	// ISO 8601 strings sort lexicographically — `>=created` enforces monotonic time.
	updated!:        #DateStr & >=created
	authors!:        [_, ...string]
	implementation!: #ImplementationStatus
	related!:        [...#IDStr]
	supersedes!:     [...#IDStr]
	superseded_by!:  null | #IDStr

	// Optional metadata. Status-conditional constraints below tighten them.
	semver?:        #SemverImpact
	slices?:        [...#SliceStr]
	competes_with?: [...#IDStr]

	// Cross-field rules. status (design lifecycle) and implementation.status
	// (code lifecycle) are independent axes; these constraints couple them
	// only where the combination would be incoherent.

	// semver becomes required once design impact is known (anything past draft).
	if status != "draft" {
		semver!: #SemverImpact
	}

	// accepted = design frozen, code in flight. Cannot be `complete` (that's
	// what `implemented` is for).
	if status == "accepted" {
		implementation: status: "not-started" | "in-progress" | "partial"
	}

	// implemented = all design intent shipped. `partial`/`in-progress` here
	// would mean we lied about the status; carve remaining work into a new
	// enhancement instead.
	if status == "implemented" {
		implementation: status: "complete"
	}

	// Tighten null|#IDStr to non-null when the entry is actually superseded.
	if status == "superseded" {
		superseded_by: #IDStr
	}
}

#ImplementationStatus: {
	status!: #ImplStatus
	notes?:  string

	// `date` is the canonical completion date. It is only meaningful — and
	// only allowed — when status reaches `complete`. Snapshot dates on
	// `partial`/`in-progress`/`not-started` go stale immediately and just
	// add noise; keep them out of structured metadata. Even at `complete`,
	// `date` is optional: some enhancements (especially umbrellas) reach
	// completion through a sequence of slices and the meaningful date lives
	// in the impl-status quote block in README.md.
	if status == "complete" {
		date!: #DateStr
	}
	if status != "complete" {
		// Forbid `date` by constraining the optional field to bottom — if
		// the field is present, validation fails.
		date?: _|_
	}
}
