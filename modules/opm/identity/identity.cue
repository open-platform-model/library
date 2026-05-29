// Package identity is the single source of the catalog's module path and
// version. It sits at the bottom of the catalog's import graph (it imports
// nothing within the module) so transformer/resource/trait/blueprint
// subpackages can source `ModulePath`/`Version` without a circular import,
// and the root `catalog.cue` can stamp transformer metadata in lockstep
// (enhancement 0001 D19/D-A).
//
// Publish-time stamping writes a transient `version_override.cue` into this
// package pinning a concrete SemVer; the committed tree always resolves
// `Version` to the "0.0.0-dev" default.
package identity

// #VersionType mirrors core.#VersionType (SemVer 2.0). Duplicated here so the
// identity package stays import-free at the bottom of the graph, matching the
// mirror convention used by schemas/common.cue for #NameType.
#VersionType: string & =~"^\\d+\\.\\d+\\.\\d+(-[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?(\\+[0-9A-Za-z-]+(\\.[0-9A-Za-z-]+)*)?$"

// ModulePath is the catalog's CUE module path (no @vN qualifier, no version).
ModulePath: "opmodel.dev/catalogs/opm"

// Version is the catalog's bare SemVer. Defaults to the dev sentinel in the
// committed tree; a publish-time version_override.cue unifies it to a
// concrete release version.
Version: #VersionType | *"0.0.0-dev"
