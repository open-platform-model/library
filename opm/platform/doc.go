// Package platform defines the Platform and PlatformMetadata types,
// mirroring the #Platform definition of the OPM core schema. A Platform
// represents a deployment target's identity, type, and the catalogs it
// carries, in the unified (Metadata, Package, Source) artifact shape used
// elsewhere in the kernel.
//
// A platform is a CUE module on disk that imports its catalogs: every
// #registry entry embeds a catalog by import and core derives the entry's
// version and the platform's #composedTransformers from it (0019:D5/D17).
// The kernel acquires it with Kernel.AcquirePlatformFromDir, which stamps
// Source, and renders against it with Kernel.Render, which
// imports the platform package into the render build. The composed
// transformers are read by the render glue, in CUE, inside the build; the
// one derived view Go reads by path is the contract inventory
// (#Platform.#contracts). The platform's constructor decodes it, and the
// core floor (whether #contracts carries providedBy), once, and records
// them on the Platform; Platform.Contracts and Platform.CoreFloor return
// the recorded result.
package platform

// Design records behind the package doc above, for maintainers:
//   - enhancements/0019 (workspace root): single-build render
//   - enhancements/0015 (workspace root): the contract inventory
//   - adr/006-single-build-artifact-construction.md (one CUE build per artifact)
//   - adr/005-shares-nothing-renders.md (the build's lifetime and concurrency)
