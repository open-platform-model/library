// Package platform is the opt-in helper for composing a fully-realised
// *platform.Platform from a Platform "shell" and a list of registered
// Modules. It is the runtime counterpart to the static path described in
// catalog enhancement 014: where an admin-authored platform.cue lists each
// #ModuleRegistration directly, this helper performs the same registration
// programmatically — for opm-operator reconciling ModuleRelease CRs, for
// the Crossplane composition fn, or for any frontend that builds the
// active Module set in Go.
//
// Compose takes a Platform shell (a base value with metadata, type, ctx,
// and possibly a partial #registry) plus the Modules to register, and
// returns a fresh Platform whose #registry is populated and whose
// computed views (#composedTransformers, #matchers, #knownResources,
// #knownTraits) resolve through the schema's CUE comprehensions. The
// helper is purely additive — inputs are not mutated — and idempotent:
// composing the same (shell, modules) twice yields semantically identical
// Platforms.
//
// Transformer-FQN collisions across registered Modules unify naturally
// at #composedTransformers (the map is keyed by transformer FQN; identical
// bodies are no-ops, divergent bodies fail unification). Compose returns
// the underlying CUE diagnostic when this happens.
//
// ID scheme: per catalog 014 D16, registry keys are kebab-case
// (#NameType). Compose uses each Module's metadata.name as the registry
// key, mirroring the catalog convention exactly. A frontend that needs a
// different scheme (per-environment overrides, tenant-prefixed IDs)
// should compose a different shell or pre-rename the Module before
// passing it in; an IDFunc option will be added if a real consumer asks.
//
// This is slice 10 of the kernel-redesign-around-platform enhancement.
// See enhancements/001-kernel-redesign-around-platform/02-design.md and
// the slice 09 matcher contract in opm/compile.
package platform
