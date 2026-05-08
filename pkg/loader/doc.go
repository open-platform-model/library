// Package loader is a deprecation shim that re-exports the
// filesystem-coupled loader from its new home at
// github.com/open-platform-model/library/pkg/helper/loader/file.
//
// Slice 07 (reorganize-helpers-under-helper) of the kernel-redesign
// enhancement moved the loader under the opt-in helper boundary at
// pkg/helper/. Existing import paths break at the package level; this
// shim keeps a single SemVer cycle's worth of source compatibility for
// downstream consumers.
//
// Migrate by replacing
//
//	"github.com/open-platform-model/library/pkg/loader"
//
// with
//
//	loader "github.com/open-platform-model/library/pkg/helper/loader/file"
//
// (or import without the alias and adjust call sites). Symbols, return
// types, and error semantics are identical.
//
// Deprecated: import github.com/open-platform-model/library/pkg/helper/loader/file
// directly. This shim is scheduled for removal in the next MAJOR release
// of the library; see CHANGELOG for the cycle.
package loader
