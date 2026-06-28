// Package stage builds an in-memory load.Config.Overlay from a fetched CUE
// module's source location, keyed under a deterministic synthetic root. It is
// single-sourced here because two callers need the identical staging behavior:
//
//   - opm/helper/loader/registry loads a published #Module AS the main module,
//     so the module's own cue.mod/module.cue drives transitive resolution.
//   - opm/helper/synth synthesizes a #ModuleInstance INSIDE the module's staged
//     source tree (reusing the same module.cue), then loads an instance package
//     overlaid under a subdirectory of the same synthetic root.
//
// Keeping the staging logic in one place guarantees both callers root the module
// identically and resolve its dependencies through the same mechanism.
package stage

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue/load"
	"cuelang.org/go/mod/module"

	"github.com/open-platform-model/library/opm/helper/loader/internal/shape"
)

// OverlayFromSource reads every file in the fetched module's source location
// into a load.Config.Overlay keyed under a deterministic synthetic absolute
// root, returning that root and the overlay. The root is derived from
// path@version and need not exist on the real filesystem; the loader treats the
// overlaid files as if present there (and all parent dirs as existing).
//
// The returned overlay includes the module's own cue.mod/module.cue, so a build
// rooted at the returned root uses the module's published (already-tidied)
// dependency closure to resolve transitive imports.
func OverlayFromSource(loc module.SourceLoc, modPath, version string) (string, map[string]load.Source, error) {
	synthRoot := SyntheticRoot(modPath, version)
	overlay := map[string]load.Source{}

	root := loc.Dir
	if root == "" {
		root = "."
	}

	err := fs.WalkDir(loc.FS, root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(loc.FS, p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		// loc.FS uses io/fs slash paths; compute the path relative to the
		// module root and rebase it under the synthetic OS root.
		rel := p
		if root != "." {
			rel = strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
		}
		key := filepath.Join(synthRoot, filepath.FromSlash(rel))
		overlay[key] = load.FromBytes(data)
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	if len(overlay) == 0 {
		return "", nil, fmt.Errorf("fetched module source is empty: %w", shape.ErrInvalidPackage)
	}
	return synthRoot, overlay, nil
}

// SyntheticRoot returns a deterministic absolute path used as the in-memory
// module root for the overlay. It is derived purely from path@version (no
// randomness, no clock) so the load is reproducible, and is sanitized into a
// single path segment so it never collides with real source on disk.
func SyntheticRoot(modPath, version string) string {
	repl := strings.NewReplacer("/", "_", ":", "_", "@", "_", "+", "_")
	safe := repl.Replace(modPath + "@" + version)
	return string(filepath.Separator) + filepath.Join("opm-registry-module", safe)
}
