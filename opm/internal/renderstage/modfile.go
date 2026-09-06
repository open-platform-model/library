package renderstage

import (
	"fmt"
	"path/filepath"
	"strings"

	"cuelang.org/go/mod/modfile"

	"github.com/open-platform-model/library/opm/internal/sourcetree"
	"github.com/open-platform-model/library/opm/module"
)

// ModFileName is the module file path relative to a module root.
const ModFileName = "cue.mod/module.cue"

// Dep is one dependency entry of a parsed module file, with the default-major
// marker intact: a catalog's `default: true` for a path is honoured by cue/load
// only while that path is a root dependency (0019 02-design.md, "The render
// build"), so promotion must carry the marker into the render module.
type Dep struct {
	// Version is the canonical dependency version ("v1.2.3", "v2.0.0-alpha.7").
	Version string

	// Default marks this major as the default for imports of the path that
	// omit a major qualifier.
	Default bool
}

// ModFile is the parsed view of one input's committed cue.mod/module.cue.
type ModFile struct {
	// Module is the qualified module path, major suffix included
	// ("testing.opmodel.dev/library-parity@v0").
	Module string

	// Language is the declared language.version ("v0.17.0").
	Language string

	// Deps maps major-qualified dependency paths ("opmodel.dev/core@v2") to
	// their entries.
	Deps map[string]Dep
}

// ParseModFile parses a module.cue in its standard (strict) format: every
// dependency carries its major in the path and a canonical version. filename
// is used for error messages only.
func ParseModFile(data []byte, filename string) (*ModFile, error) {
	f, err := modfile.Parse(data, filename)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filename, err)
	}
	if f.QualifiedModule() == "" {
		return nil, fmt.Errorf("parsing %s: module path is empty", filename)
	}
	mf := &ModFile{
		Module: f.QualifiedModule(),
		Deps:   make(map[string]Dep, len(f.Deps)),
	}
	if f.Language != nil {
		mf.Language = f.Language.Version
	}
	for path, dep := range f.Deps {
		if dep == nil {
			continue
		}
		if dep.ReplaceWith != "" {
			// cue/load itself refuses this in module.cue; mirror it so a
			// hand-edited input cannot smuggle a replacement into the render.
			return nil, fmt.Errorf("parsing %s: dependency %q carries replaceWith, which is not allowed in module.cue", filename, path)
		}
		mf.Deps[path] = Dep{Version: dep.Version, Default: dep.Default}
	}
	return mf, nil
}

// ReadModFile reads and parses the cue.mod/module.cue of a staged source tree:
// from the overlay in overlay mode, from disk in on-disk mode.
func ReadModFile(src *module.Source) (*ModFile, error) {
	if src == nil || src.Root == "" {
		return nil, fmt.Errorf("source carries no module root")
	}
	path := filepath.Join(src.Root, filepath.FromSlash(ModFileName))
	data, err := sourcetree.ReadFile(src, path)
	if err != nil {
		return nil, err
	}
	return ParseModFile(data, path)
}

// IsOPMPath reports whether a major-qualified module path lives in the OPM
// namespace: its host element is opmodel.dev or a subdomain of it. This is the
// path set the D13 refusal invariant and the D7 skew comparison cover;
// fixture domains (testing.opmodel.dev) are included deliberately so the
// invariant is exercised by fixture-backed tests.
func IsOPMPath(path string) bool {
	host, _, _ := strings.Cut(path, "/")
	host, _, _ = strings.Cut(host, "@")
	return host == "opmodel.dev" || strings.HasSuffix(host, ".opmodel.dev")
}
