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

#Status:     "draft" | "accepted" | "implemented" | "superseded"
#ImplStatus: "not-started" | "in-progress" | "partial" | "complete"

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
}

#ImplementationStatus: {
	status!: #ImplStatus
	date?:   #DateStr
	notes?:  string
}
