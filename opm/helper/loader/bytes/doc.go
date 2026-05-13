// Package bytes provides in-memory loaders for OPM artifacts: module
// CUE packages, releases, and providers built from byte buffers rather
// than a filesystem.
//
// It is the sibling of opm/helper/loader/file. Use bytes when the
// embedding caller does not have a real filesystem and instead receives
// raw artifact bytes — for example a Crossplane composition function
// that gets module/release CUE inline on its gRPC request, or a
// fuzzing harness that synthesises artifacts in memory.
//
// Status: skeleton only. The package intentionally exposes no functions
// yet. The full implementation lands in a follow-up slice once a
// concrete consumer (Crossplane fn, in-memory tests, fuzzing target)
// pulls on the design — adding API speculatively would violate the
// library's YAGNI rule. See the umbrella enhancement at
// enhancements/001-kernel-redesign-around-platform/02-design.md for the
// long-term plan.
package bytes
