// Package sourcetree walks, reads, names and writes the source tree a
// [module.Source] describes, in both of its modes: on-disk (Overlay nil, the
// tree is read from Root on the filesystem) and overlay (every file is an
// entry of Overlay, its bytes keyed by its absolute path under Root).
//
// It is under opm/internal/ so the kernel, the render stage and the loader
// share one implementation: the kernel builds an overlay from an on-disk
// instance module to layer values onto it, the registry loader builds one
// from a fetched module's filesystem under a synthetic root, and the render
// stage reads a source's module file and package clause before handing the
// tree to module.Source.WriteTo. Writing an overlay out is that method's job,
// not this package's.
//
// Both walkers apply one filter: a regular file is included iff its name ends
// in ".cue", the module's own cue.mod/module.cue included. That is the set
// cue/load reads (nothing in the workspace uses @embed), and it keeps an
// on-disk walk from swallowing .git or vendored trees. Directory names are
// not filtered, so a vendored cue.mod/pkg/**/*.cue is included as cue/load
// would read it.
package sourcetree

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue/parser"

	"github.com/open-platform-model/library/opm/module"
)

// cueSuffix is the file-name suffix the walkers include.
const cueSuffix = ".cue"

// PackageName returns the package clause shared by the .cue files directly
// inside src's package directory (Root joined with Pkg), read from the
// overlay in overlay mode and from disk otherwise. Files without a package
// clause are not part of the package and are skipped; a directory declaring
// no clause or more than one is an error naming the directory.
func PackageName(src *module.Source) (string, error) {
	if src == nil || src.Root == "" {
		return "", errors.New("source carries no module root")
	}
	pkgDir := filepath.Join(src.Root, filepath.FromSlash(src.Pkg))
	files, err := packageFiles(src, pkgDir)
	if err != nil {
		return "", err
	}
	names := map[string]bool{}
	for _, f := range files {
		data, err := ReadFile(src, f)
		if err != nil {
			return "", err
		}
		parsed, err := parser.ParseFile(f, data, parser.PackageClauseOnly)
		if err != nil {
			return "", fmt.Errorf("parsing %s: %w", f, err)
		}
		if n := parsed.PackageName(); n != "" {
			names[n] = true
		}
	}
	switch len(names) {
	case 0:
		return "", fmt.Errorf("no package clause in %s", pkgDir)
	case 1:
		for n := range names {
			return n, nil
		}
	}
	list := make([]string, 0, len(names))
	for n := range names {
		list = append(list, n)
	}
	sort.Strings(list)
	return "", fmt.Errorf("%s declares more than one package: %v", pkgDir, list)
}

// packageFiles lists the .cue files directly inside pkgDir, sorted.
func packageFiles(src *module.Source, pkgDir string) ([]string, error) {
	var files []string
	if src.Overlay != nil {
		for key := range src.Overlay {
			if filepath.Dir(key) == pkgDir && strings.HasSuffix(key, cueSuffix) {
				files = append(files, key)
			}
		}
	} else {
		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", pkgDir, err)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), cueSuffix) {
				files = append(files, filepath.Join(pkgDir, e.Name()))
			}
		}
	}
	sort.Strings(files)
	return files, nil
}

// OverlayFromDir reads every .cue file under root (the module's own
// cue.mod/module.cue included) into an overlay map keyed by the
// file's path under root; pass an absolute root so the keys are the absolute
// paths cue/load expects. The overlay mirrors the on-disk tree rather than
// re-deciding what cue/load includes, so a build from it evaluates exactly as
// the on-disk build would.
func OverlayFromDir(root string) (map[string][]byte, error) {
	overlay := map[string][]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), cueSuffix) {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		overlay[p] = data
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading module tree %s: %w", root, err)
	}
	return overlay, nil
}

// OverlayFromFS reads every .cue file under dir of fsys (an io/fs tree with
// slash paths; dir "" or "." is the tree's root) into an overlay map
// keyed under keyRoot, an absolute path that need not exist on disk: each
// file's path relative to dir is rebased under it. A fetched module's
// [module.SourceLoc] is the expected input, with [SyntheticRoot] as keyRoot.
// An empty result is not an error here; the caller decides what an empty
// module means.
func OverlayFromFS(fsys fs.FS, dir, keyRoot string) (map[string][]byte, error) {
	if dir == "" {
		dir = "."
	}
	overlay := map[string][]byte{}
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), cueSuffix) {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		rel := p
		if dir != "." {
			rel = strings.TrimPrefix(strings.TrimPrefix(p, dir), "/")
		}
		overlay[filepath.Join(keyRoot, filepath.FromSlash(rel))] = data
		return nil
	})
	if err != nil {
		return nil, err
	}
	return overlay, nil
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

// ReadFile returns the contents of path inside src: the overlay entry in
// overlay mode, the file on disk otherwise. A file that does not exist in
// either mode is reported with an error wrapping [fs.ErrNotExist], so a
// caller reading an optional file checks absence the same way for both.
func ReadFile(src *module.Source, path string) ([]byte, error) {
	if src == nil || src.Root == "" {
		return nil, errors.New("source carries no module root")
	}
	if src.Overlay != nil {
		entry, ok := src.Overlay[path]
		if !ok {
			return nil, fmt.Errorf("%s: not present in the staged overlay under %s: %w", path, src.Root, fs.ErrNotExist)
		}
		return entry, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return data, nil
}
