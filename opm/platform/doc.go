// Package platform defines the Platform and PlatformMetadata types,
// mirroring the #Platform definition introduced in catalog enhancement
// 014-platform-construct. A Platform represents a deployment target's
// identity, type, and registered Module extensions in the unified
// (APIVersion, Metadata, Package) artifact shape used elsewhere in the
// kernel.
//
// Catalog enhancement 014 introduces #Platform as the replacement for
// #Provider. #Platform carries a #registry of #ModuleRegistration entries
// and exposes computed CUE views — #knownResources, #knownTraits,
// #composedTransformers, #matchers — that the kernel matcher consumes
// when walking a consumer Module's FQN demand. The matcher rewrite
// retires the previous Provider package; Platform is now the kernel's
// sole input for matching and execution.
//
// All Platform views remain accessible only via Package.LookupPath using
// the version-binding paths (api.Binding.Paths().Registry,
// .ComposedTransformers, .Matchers, .KnownResources, .KnownTraits). They
// are intentionally not eagerly decoded into Go: the matcher iterates them
// lazily and the views are CUE-computed.
//
// See:
//   - catalog/enhancements/014-platform-construct/ — the source enhancement
//   - library/enhancements/001-kernel-redesign-around-platform/ — umbrella
//   - openspec/changes/add-platform-construct/ — this slice's proposal
package platform
