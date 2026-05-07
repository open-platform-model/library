package types

import (
	"strings"
)

#LabelsAnnotationsType: [string]: string | int | bool | [string | int | bool]

// NameType: RFC 1123 DNS label — lowercase alphanumeric with hyphens, max 63 chars
#NameType: string & =~"^[a-z0-9]([a-z0-9-]*[a-z0-9])?$" & strings.MinRunes(1) & strings.MaxRunes(63)

// ModulePathType: plain registry path without embedded version
// Example: "opmodel.dev/opm/modules", "opmodel.dev/opm/traits/workload"
#ModulePathType: string & =~"^[a-z0-9.-]+(/[a-z0-9.-]+)*$" & strings.MinRunes(1) & strings.MaxRunes(254)

// MajorVersionType: major version prefix used in primitive FQNs
// Example: "v1", "v0"
#MajorVersionType: string & =~"^v[0-9]+$"

// ModuleFQNType: container-style FQN for #Module — path/name:semver
// Example: "opmodel.dev/opm/modules/my-app:1.2.3"
#ModuleFQNType: string & =~"^[a-z0-9.-]+(/[a-z0-9.-]+)*/[a-z0-9]([a-z0-9-]*[a-z0-9])?:\\d+\\.\\d+\\.\\d+.*$"

// BundleFQNType: FQN for #Bundle — path/name:vN (major version)
// Example: "opmodel.dev/opm/bundles/game-stack:v1"
#BundleFQNType: string & =~"^[a-z0-9.-]+(/[a-z0-9.-]+)*/[a-z0-9]([a-z0-9-]*[a-z0-9])?:v[0-9]+$"

// Semver 2.0
#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"

// FQNType: primitive definition FQN — path/name@version
// Example: "opmodel.dev/opm/traits/workload/scaling@v1"
// Example: "opmodel.dev/opm/resources/workload/container@v1"
// Example: "github.com/myorg/traits/network/expose@v2"
#FQNType: string & =~"^[a-z0-9.-]+(/[a-z0-9.-]+)*/[a-z0-9]([a-z0-9-]*[a-z0-9])?@v[0-9]+$"

// UUIDType: RFC 4122 UUID in standard format (lowercase hex)
#UUIDType: string & =~"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"

// OPM namespace UUID for uuid computations via uuid.SHA1 (UUID v5).
// This UUID MUST remain immutable across all versions — it is the root namespace
// for all OPM uuid generation. The CLI uses the same constant.
OPMNamespace: "11bc6112-a6e8-4021-bec9-b3ad246f9466"

// KebabToPascal converts a kebab-case string to PascalCase.
// Usage: (#KebabToPascal & {"in": "stateless-workload"}).out => "StatelessWorkload"
#KebabToPascal: {
	X="in": string
	let _parts = strings.Split(X, "-")
	out: strings.Join([for p in _parts {
		let _runes = strings.Runes(p)
		strings.ToUpper(strings.SliceRunes(p, 0, 1)) + strings.SliceRunes(p, 1, len(_runes))
	}], "")
}

// KebabToCamel converts a kebab-case string to camelCase.
// Usage: (#KebabToCamel & {"in": "k8up-backup"}).out => "k8upBackup"
// The first segment stays lowercase; every subsequent segment is capitalized.
#KebabToCamel: {
	X="in": string
	let _parts = strings.Split(X, "-")
	out: strings.Join([for i, p in _parts {
		if i == 0 {p}
		if i > 0 {
			let _runes = strings.Runes(p)
			strings.ToUpper(strings.SliceRunes(p, 0, 1)) + strings.SliceRunes(p, 1, len(_runes))
		}
	}], "")
}
