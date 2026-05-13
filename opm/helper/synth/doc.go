// Package synth builds OPM artifact CUE values from in-memory typed inputs by
// unifying caller-supplied identity (name, namespace, module reference,
// values, labels, annotations) against the embedded schema definition for the
// target API version.
//
// Synth is a peer of opm/helper/loader, not a subpackage of it. The loader
// tree (opm/helper/loader/file, opm/helper/loader/bytes) reads existing
// artifact bytes from a source — filesystem or byte buffer. Synthesis is
// creation from typed inputs: there is no file to parse, no bytes to decode.
// Co-locating synth under loader would conflate verbs and force every reader
// to ignore the package doc when interpreting the package path. Keeping them
// as peers makes each verb legible at a glance.
//
// Recommended entry point: (*kernel.Kernel).SynthesizeRelease. It chains
// synth.Release into Kernel.ProcessModuleRelease so a caller building a
// release from typed inputs gets a fully validated, concrete *module.Release
// in one call. Call synth.Release directly only when there is no *Kernel on
// hand (rare; mostly unit-testing transformer behaviour in tight loops).
//
// Schema source of truth: synth.Release never reimplements derivations the
// CUE schema already owns (UUID stamping, components fan-out from #components,
// auto-secrets injection, standard label stamping). Every derived field flows
// through unification with the schema's #ModuleRelease definition, loaded
// from the version binding's embedded filesystem via
// api.Binding.SchemaValue. No CUE_REGISTRY round-trip, no filesystem read.
//
// Boundary scope: synth helpers live under opm/helper/ because they are
// opinionated frontend conveniences. Frontends MAY bypass them and unify
// against the schema themselves; nothing in synth is part of the kernel
// contract.
package synth
