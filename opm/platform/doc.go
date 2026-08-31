// Package platform defines the Platform and PlatformMetadata types,
// mirroring the #Platform definition introduced in catalog enhancement
// 014-platform-construct. A Platform represents a deployment target's
// identity, type, and registered Module extensions in the unified
// (APIVersion, Metadata, Package) artifact shape used elsewhere in the
// kernel.
//
// Catalog enhancement 014 introduces #Platform as the replacement for
// #Provider. #Platform carries a #registry of #ModuleRegistration entries
// and declares the computed CUE views (#composedTransformers, #matchers)
// over which the kernel matcher and executor operate. The matcher rewrite
// retires the previous Provider package; Platform is now the kernel's sole
// input for matching and execution.
//
// The #composedTransformers / #matchers data is NOT materialized onto the
// closed platform value: as of federate-materialize-transformers (ADR-003),
// Materialize builds those natively in the owner *cue.Context and exposes
// them as materialize.MaterializedPlatform.Transformers / .Matchers, and
// never fills them onto Package (doing so corrupts output-local hidden
// fields in #transforms). The schema still declares the fields, but on a
// materialized platform they are unfilled here: read the composed map /
// reverse index off MaterializedPlatform, not off platform.Package.
//
// See:
//   - catalog/enhancements/014-platform-construct/ — the source enhancement
//   - library/enhancements/001-kernel-redesign-around-platform/ — umbrella
//   - openspec/changes/add-platform-construct/ — this slice's proposal
package platform
