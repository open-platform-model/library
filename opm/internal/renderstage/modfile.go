package renderstage

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"cuelang.org/go/mod/modfile"

	"github.com/open-platform-model/library/opm/internal/sourcetree"
	"github.com/open-platform-model/library/opm/module"
)

// ModFileName is the module file path relative to a module root.
const ModFileName = "cue.mod/module.cue"

// LocalModFileName is the path, relative to a module root, of the optional
// main-module dependency view cue/load reads in place of module.cue's deps:
// the file a developer redirects dependencies with (CUE v0.17.0).
const LocalModFileName = "cue.mod/local-module.cue"

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
	// their entries. A dependency may carry no version: cue accepts the
	// shape for a path a local replacement serves, and promotion refuses it
	// when no promoted replacement covers the path.
	Deps map[string]Dep

	// file is the parsed module file, kept so the module's local view can be
	// parsed against it without a second parse.
	file *modfile.File
}

// LocalModFile is the parsed view of one input's cue.mod/local-module.cue:
// the dependencies the developer redirected and the entries the file lists.
// cue/load reads the file only from the main module, so an input's view
// reaches the render build only by promotion into the render module's own.
type LocalModFile struct {
	// Replacements maps each replaced major-qualified path to its target: an
	// absolute directory (a relative one is resolved against the input's
	// module root, since cue/load resolves it against the main module's root
	// and the render module's root is elsewhere), or a module path verbatim.
	Replacements map[string]string

	// Deps is every entry the local file lists, versions and default markers
	// as cue/load reads them against the module file (a version omitted in
	// the local file is the module file's). Replacement targets are on
	// Replacements, never here.
	Deps map[string]Dep
}

// ParseModFile parses a module.cue in its standard (strict) format: every
// dependency carries its major in the path and, unless a local replacement
// serves it, a canonical version. filename is used for error messages only.
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
		file:   f,
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

// ReadLocalModFile reads and parses the cue.mod/local-module.cue of a staged
// source tree against its parsed module file (base, from [ReadModFile]), in
// either mode. An absent file is the normal case and yields a nil view. A
// relative directory target is resolved against src.Root.
func ReadLocalModFile(src *module.Source, base *ModFile) (*LocalModFile, error) {
	if src == nil || src.Root == "" {
		return nil, fmt.Errorf("source carries no module root")
	}
	if base == nil || base.file == nil {
		return nil, fmt.Errorf("local view of %s needs its parsed module file", src.Root)
	}
	path := filepath.Join(src.Root, filepath.FromSlash(LocalModFileName))
	data, err := sourcetree.ReadFile(src, path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	eff, err := modfile.ParseLocal(data, path, base.file)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	local := &LocalModFile{
		Replacements: map[string]string{},
		Deps:         make(map[string]Dep, len(eff.Deps)),
	}
	for depPath, dep := range eff.Deps {
		if dep == nil {
			continue
		}
		local.Deps[depPath] = Dep{Version: dep.Version, Default: dep.Default}
		if dep.ReplaceWith == "" {
			continue
		}
		target := dep.ReplaceWith
		if isReplaceDir(target) && !filepath.IsAbs(target) {
			target = filepath.Join(src.Root, filepath.FromSlash(target))
		}
		local.Replacements[depPath] = target
	}
	return local, nil
}

// isReplaceDir reports whether a replaceWith value names a directory rather
// than a module path, by cue/load's own rule: a directory starts with "."
// or "/" (or is an absolute path on the host).
func isReplaceDir(target string) bool {
	return strings.HasPrefix(target, ".") || strings.HasPrefix(target, "/") || filepath.IsAbs(target)
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
