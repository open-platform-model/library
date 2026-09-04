// Package platformmodule generates a platform CUE module from catalog
// coordinates (enhancement 0019 D5/D13). A platform is a CUE module that
// imports its catalogs, and the kernel's only platform input is such a module
// on disk ([kernel.Kernel.AcquirePlatformFromDir]); a frontend that starts
// from typed coordinates (a Platform CR, a seeded local default) turns them
// into that module here instead of writing its own generator.
//
// Three seams, each independently testable:
//
//   - [Generate] is pure: typed input plus the resolved dependency closure in,
//     deterministic file bytes out. The same input always yields byte-identical
//     files (cue.mod/module.cue in modfile canonical format, platform.cue
//     embedding core.#Platform and importing every catalog under a positional
//     alias).
//   - [Closure] derives the module's full dependency list from the pinned
//     modules' published module files: the roots ([Roots]: core and every
//     subscribed catalog) plus everything they transitively require, at the
//     maximum version any requirement names. It is the tidied list a
//     `cue mod tidy` would write, computed once at generation (0019 D13).
//     Module files are read through a caller-configured [ModFileSource]
//     ([NewRegistry]); tests supply a fixture graph.
//   - [Files.WriteTo] places the generated files under a caller-owned
//     directory. Directory lifecycle (generations, staging swaps, retention)
//     stays with the frontend.
//
// The core pin defaults to the release the kernel was verified against
// ([schema.DefaultSchemaVersion]); [WithCoreVersion] overrides it. The
// generated module's own path is caller input ([Input.ModulePath]) and lives
// under the reserved, never-published platforms namespace (0019 D6).
//
// This package is opt-in helper convenience (see package opm/helper): a
// frontend MAY write its platform module by hand instead.
package platformmodule
