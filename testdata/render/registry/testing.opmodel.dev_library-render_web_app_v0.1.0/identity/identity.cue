// The module's identity package: the single source of its path and version,
// imported UNQUALIFIED by the module's root package, the ordinary authoring
// shape (opm module init writes it). An overlay-mode instance synthesized
// from this module enters the render build through a directory replacement,
// where this self-import resolves only because the render module marks the
// module's major default (renderstage.Promote).
package identity

ModulePath: "testing.opmodel.dev/library-render/web_app@v0"
Version:    "0.1.0"
