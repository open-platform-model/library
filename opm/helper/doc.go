// Package helper is the opt-in convenience boundary of the OPM library.
//
// Anything under opm/helper/ is opinionated frontend convenience: it makes
// embedding the kernel easier, but a frontend MAY skip it and call the
// kernel directly. Anything outside opm/helper/ is part of the kernel
// contract that every frontend (CLI, controller, Crossplane fn, future
// runtimes) MUST honour.
//
// Helper subpackages are added by their owning slices of the
// kernel-redesign-around-platform enhancement. Drift requires a deliberate
// enhancement; one-off additions are not allowed.
//
// Current subpackages:
//
//   - loader/file  — filesystem-coupled loading of modules, releases, and
//     providers from a CUE module directory or .cue file. Use when the
//     frontend has access to a real filesystem.
//   - loader/bytes — in-memory loading skeleton; full implementation
//     deferred until a consumer (Crossplane composition fn, fuzzing
//     harness, in-memory tests) demands it.
//   - platform     — Platform composition (Compose(shell, modules) →
//     *Platform with #registry filled). Implemented by slice 10
//     (add-platform-composition-helper).
//   - synth        — artifact synthesis from in-memory typed inputs.
//     synth.Release composes a #ModuleRelease CUE value by unifying
//     (Module, name, namespace, values, labels, annotations) against the
//     embedded #ModuleRelease schema. Peer of loader/ (loading parses
//     bytes; synth creates from typed inputs). Recommended entry point is
//     (*Kernel).SynthesizeRelease, which chains synth.Release into
//     ProcessModuleRelease for a fully validated *module.Release.
//
// Layered values validation now lives on the kernel itself: see
// Kernel.ValidateConfigDetailed and the Source / ValidateOption types in
// opm/kernel. The earlier opm/helper/values subpackage was removed as
// part of redesign-config-validation.
//
// Planned subpackages (added by their respective slices):
//
//   - embed    — one-call embedding wrappers for the most common patterns.
//     Deferred until a consumer asks for it (YAGNI).
//
// In scope: opinionated convenience that wraps kernel primitives for a
// specific embedding pattern.
//
// Out of scope: anything the kernel must own (artifact types, render
// pipeline, validation rules, version dispatch). Those live outside
// opm/helper/.
//
// Slice 07 (reorganize-helpers-under-helper) established this boundary by
// moving opm/loader to opm/helper/loader/file. See the umbrella enhancement
// at enhancements/001-kernel-redesign-around-platform/ for the full
// design.
package helper
