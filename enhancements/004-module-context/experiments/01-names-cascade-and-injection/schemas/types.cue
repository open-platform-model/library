package v1alpha2

import (
	"strings"
)

// Trimmed from apis/core/v1alpha2/types.cue. Only the types touched by
// #ModuleContext / #ContextBuilder and the trimmed #Module / #Component stubs.

#ApiVersion: "opmodel.dev/v1alpha2"

#LabelsAnnotationsType: [string]: string | int | bool | [string | int | bool]

#NameType: string & =~"^[a-z0-9]([a-z0-9-]*[a-z0-9])?$" & strings.MinRunes(1) & strings.MaxRunes(63)

#ModuleFQNType: string & =~"^[a-z0-9.-]+(/[a-z0-9.-]+)*/[a-z0-9]([a-z0-9-]*[a-z0-9])?:\\d+\\.\\d+\\.\\d+.*$"

#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"

#UUIDType: string & =~"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
