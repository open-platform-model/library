// Package helper is the opt-in convenience boundary of the OPM library.
//
// Anything under opm/helper/ is opinionated frontend convenience: it makes
// embedding the kernel easier, but a frontend MAY skip it and call the
// kernel directly. Anything outside opm/helper/ is part of the kernel
// contract that every frontend (CLI, controller, Crossplane fn, future
// runtimes) MUST honour.
//
// The boundary is real in the import graph, not just described: no package
// outside opm/helper/ — opm/kernel, opm/module, opm/platform, opm/schema,
// opm/errors, opm/core, opm/compat and every package under opm/internal/ —
// imports anything under it, no exported kernel signature names a type
// declared here, and no kernel operation returns an error whose sentinel is
// declared here. A depguard rule in .golangci.yml enforces it on every PR.
//
// Helper subpackages are added by their owning slices of the
// kernel-redesign-around-platform enhancement. Drift requires a deliberate
// enhancement; one-off additions are not allowed.
//
// Current subpackages:
//
//   - platformmodule — platform CUE module generation from catalog
//     coordinates (0019 D5/D13): Generate renders cue.mod/module.cue and
//     platform.cue deterministically from typed registry entries and a
//     dependency list, Closure derives that list (the once-at-generation
//     tidy) from published module files through a caller-configured
//     registry, Files.WriteTo places the files in a caller-owned directory.
//     The result is what (*Kernel).AcquirePlatformFromDir accepts. A
//     frontend MAY write its platform module by hand instead.
//
// It is the only one. Three subpackages were folded into the kernel because
// the kernel itself depended on them, which made the "opt-in" tier mandatory:
//
//   - loader/file and loader/registry became opm/internal/loader, reached
//     through the acquire verbs (Kernel.AcquireModuleFromDir,
//     AcquireModuleFromRegistry, AcquirePlatformFromDir,
//     AcquireInstanceFromDir). Their shape-gate sentinels are declared in
//     opm/errors (ErrInvalidPackage, ErrWrongKind, ErrMissingRequiredField).
//   - synth became opm/internal/synth, reached through
//     Kernel.SynthesizeInstance with kernel.InstanceInput. Its sentinels are
//     in opm/errors too (ErrMissingModule, ErrMissingName,
//     ErrMissingNamespace, ErrMissingSource, ErrSchemaUnavailable).
//   - values was removed earlier: layered values validation lives on the
//     kernel as Kernel.ValidateConfigDetailed with the Source type.
//
// The opm/helper/platform subpackage (the Compose helper) was removed as
// part of rewrite-match-materialized, and platform synthesis (synth.Platform)
// with library-render-cutover: a platform is a CUE module on disk importing
// its catalogs, generated from coordinates by platformmodule or written by
// hand, acquired with (*Kernel).AcquirePlatformFromDir and rendered against
// with (*Kernel).Render. See openspec/changes/archive/.
//
// Planned subpackages (added by their respective slices):
//
//   - embed    — one-call embedding wrappers for the most common patterns.
//     Deferred until a consumer asks for it (YAGNI).
//
// In scope: opinionated convenience that wraps kernel primitives for a
// specific embedding pattern.
//
// Out of scope: anything the kernel must own (artifact types, artifact
// loading, synthesis, render pipeline, validation rules, version dispatch).
// Those live outside opm/helper/.
//
// See the umbrella enhancement at
// enhancements/001-kernel-redesign-around-platform/ for the full design.
package helper
