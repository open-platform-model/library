package module

import "cuelang.org/go/cue/load"

// Source is the staged source tree an artifact was loaded or synthesized
// from: the module root the tree is keyed under, the package directory inside
// it, and (for in-memory trees) the load.Config.Overlay carrying the files.
//
// A Source is in one of two modes:
//
//   - Overlay mode (Overlay non-empty): the tree lives in memory, keyed under
//     the deterministic synthetic Root. This is how a module fetched from a
//     registry is staged (Kernel.AcquireModuleFromRegistry) and how a
//     synthesized instance is staged inside its module's tree
//     (opm/helper/synth, surfaced by Kernel.SynthesizeInstance).
//   - On-disk mode (Overlay nil): the tree lives at Root on the real
//     filesystem. This is how an artifact acquired from a directory is
//     described (Kernel.AcquirePlatformFromDir, Kernel.AcquireInstanceFromDir).
//
// It exists so an artifact can be RE-USED as the input of a follow-on build:
// a module acquired from the registry becomes the main module of the synth
// build (so its already-tidied cue.mod/module.cue drives transitive dependency
// resolution), and an instance or platform carrying its tree can be imported
// as a package by a later render build. Carrying the staged source on the
// artifact avoids a second fetch or a second directory load.
//
// Source is carried by Module (registry path only), Instance (synthesis and
// directory acquire) and Platform (directory acquire; platform.Source is an
// alias of this type). It is nil for artifacts constructed from a bare value
// (e.g. a unit-test CompileString).
type Source struct {
	// Root is the absolute module root of the tree: the load.Config.ModuleRoot
	// a consumer builds against. In overlay mode it is the synthetic root every
	// Overlay key sits under; in on-disk mode it is a real directory.
	Root string

	// Pkg is the package directory relative to Root that holds the artifact's
	// CUE package. Empty means the root package (".").
	Pkg string

	// Overlay maps absolute paths under Root to their file contents, exactly as
	// load.Config.Overlay expects, including cue.mod/module.cue. Nil selects
	// on-disk mode: the tree is read from Root on the filesystem.
	Overlay map[string]load.Source
}

// HasSource reports whether the module carries a staged registry source tree
// (non-nil Source with a populated overlay). Consumers that must build inside
// the module's own root — e.g. synth.Instance — gate on this and return a
// deterministic error when it is false, rather than silently fetching.
func (m *Module) HasSource() bool {
	return m != nil && m.Source != nil && m.Source.Root != "" && len(m.Source.Overlay) > 0
}
