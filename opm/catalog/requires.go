package catalog

import (
	"fmt"
	"path/filepath"

	"cuelang.org/go/mod/modfile"

	"github.com/open-platform-model/library/opm/internal/sourcetree"
)

// modFileName is the module file's path relative to a module root.
const modFileName = "cue.mod/module.cue"

// Requires returns the dependency requirements the catalog COMMITTED in its
// own cue.mod/module.cue, keyed by major-qualified module path
// ("opmodel.dev/core@v2") and valued with the canonical version the file
// records ("v2.0.0-alpha.9"). Those are the two strings and nothing else: a
// consumer comparing a catalog's committed resolution against a platform's
// never sees CUE's module types, and version arithmetic within a major is
// string work.
//
// It reads the committed file, not the build: the point of the answer is what
// the catalog's author pinned when the artifact was published, which is the
// side of the comparison a consumer cannot reconstruct from its own
// resolution. The file is read from the staged Source — from the overlay in
// overlay mode, from disk in on-disk mode — so a catalog acquired from a
// registry answers without a second fetch.
//
// Reports, never refusals. Whether a requirement is acceptable, newer, older
// or incompatible is the caller's judgement; this returns what is written.
//
// Errors: a catalog carrying no Source (one built straight from a value by
// [NewCatalogFromValue] rather than acquired) has no committed file to read
// and says so; a module file that is absent or unparseable is reported with
// its path. A dependency a local replacement serves may carry no version in
// the file; its entry maps to the empty string rather than being dropped, so
// the path is still visible to a caller reconciling the two sides.
func (c *Catalog) Requires() (map[string]string, error) {
	if c == nil {
		return nil, fmt.Errorf("catalog is nil")
	}
	if c.Source == nil || c.Source.Root == "" {
		return nil, fmt.Errorf("catalog carries no source tree, so it has no committed %s to read", modFileName)
	}

	path := filepath.Join(c.Source.Root, filepath.FromSlash(modFileName))
	data, err := sourcetree.ReadFile(c.Source, path)
	if err != nil {
		return nil, fmt.Errorf("reading catalog %s: %w", modFileName, err)
	}
	f, err := modfile.Parse(data, path)
	if err != nil {
		return nil, fmt.Errorf("parsing catalog %s: %w", modFileName, err)
	}

	reqs := make(map[string]string, len(f.Deps))
	for depPath, dep := range f.Deps {
		if dep == nil {
			continue
		}
		reqs[depPath] = dep.Version
	}
	return reqs, nil
}
