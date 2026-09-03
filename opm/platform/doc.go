// Package platform defines the Platform and PlatformMetadata types,
// mirroring the #Platform definition of the OPM core schema. A Platform
// represents a deployment target's identity, type, and the catalogs it
// carries, in the unified (Metadata, Package, Source) artifact shape used
// elsewhere in the kernel.
//
// A platform is a CUE module on disk that imports its catalogs: every
// #registry entry embeds a catalog by import and core derives the entry's
// version and the platform's #composedTransformers from it (enhancement
// 0019 D5/D17). The kernel acquires it with Kernel.AcquirePlatformFromDir,
// which stamps Source, and renders against it with Kernel.Render, which
// imports the platform package into the render build. No Go code navigates
// the platform value by path: the composed transformers are read by the
// render glue, in CUE, inside the build.
//
// See:
//   - enhancements/0019 (workspace root) — single-build render
//   - adr/003-single-build-cue-evaluation-invariant.md
package platform
