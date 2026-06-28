package module

import "cuelang.org/go/cue/load"

// Source is the staged in-memory source tree of a module fetched from a
// registry: the deterministic synthetic root the module's files were keyed
// under, plus the load.Config.Overlay carrying those files (including the
// module's own cue.mod/module.cue).
//
// It exists so a module acquired from the registry can be RE-USED as the main
// module of a follow-on build — notably by opm/helper/synth, which stages a
// #ModuleInstance package inside this tree so the module's own (already-tidied)
// cue.mod/module.cue drives transitive dependency resolution. Carrying the
// staged source on the acquired *Module avoids a second registry fetch.
//
// Source is populated only when the module was acquired through the
// source-carrying registry path (Kernel.AcquireModuleFromRegistry). It is nil
// for modules constructed from a bare value (e.g. a unit-test CompileString or
// the value-returning LoadModuleFromRegistry path).
type Source struct {
	// Root is the synthetic absolute module root every Overlay key sits under.
	// It is the load.Config.ModuleRoot a consumer builds against.
	Root string

	// Overlay maps absolute paths under Root to their file contents, exactly as
	// load.Config.Overlay expects. Every file of the fetched module is present,
	// including cue.mod/module.cue.
	Overlay map[string]load.Source
}

// HasSource reports whether the module carries a staged registry source tree
// (non-nil Source with a populated overlay). Consumers that must build inside
// the module's own root — e.g. synth.Instance — gate on this and return a
// deterministic error when it is false, rather than silently fetching.
func (m *Module) HasSource() bool {
	return m != nil && m.Source != nil && m.Source.Root != "" && len(m.Source.Overlay) > 0
}
