package v1alpha2

import (
	"strings"
)

// Trimmed from apis/core/v1alpha2/types.cue. Only the types needed by the
// experiment's #Capability, #Module, #Platform, and #ContextBuilder shapes.

#ApiVersion: "opmodel.dev/v1alpha2"

#LabelsAnnotationsType: [string]: string | int | bool | [string | int | bool]

#NameType: string & =~"^[a-z0-9]([a-z0-9-]*[a-z0-9])?$" & strings.MinRunes(1) & strings.MaxRunes(63)

#ModulePathType: string & =~"^[a-z0-9.-]+(/[a-z0-9.-]+)*$" & strings.MinRunes(1) & strings.MaxRunes(254)

#MajorVersionType: string & =~"^v[0-9]+$"

#FQNType: string & =~"^[a-z0-9.-]+(/[a-z0-9.-]+)*/[a-z0-9]([a-z0-9-]*[a-z0-9])?@v[0-9]+$"

#UUIDType: string & =~"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"

#KebabToPascal: {
	X="in": string
	let _parts = strings.Split(X, "-")
	out: strings.Join([for p in _parts {
		let _runes = strings.Runes(p)
		strings.ToUpper(strings.SliceRunes(p, 0, 1)) + strings.SliceRunes(p, 1, len(_runes))
	}], "")
}
