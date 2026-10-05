package loader

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	oerrors "github.com/open-platform-model/library/opm/errors"
	opmmodule "github.com/open-platform-model/library/opm/module"
)

// Options carries the load settings every build shares, so a new setting is
// added here once rather than at every [LoadDir] call site.
type Options struct {
	// Env is the environment slice load.Config consults: the CUE_REGISTRY
	// override the kernel builds with [cueenv.Override], which owns the
	// concurrency rule. Nil reads the process environment unchanged.
	Env []string
}

// CheckDir reports whether dir exists and is a directory, in the words every
// directory acquire verb uses for a bad path: `accessing <label> directory
// "<dir>": <stat error>` (wrapping the stat error, so errors.Is matches
// fs.ErrNotExist) or `<label> path "<dir>" is not a directory`, the label
// taken from spec. It reads nothing beyond the stat, so a caller runs it
// before reading the tree and a bad path is reported as a path problem
// rather than as an unreadable tree or "matched no packages".
func CheckDir(dir string, spec ArtifactSpec) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("accessing %s directory %q: %w", spec.Label, dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s path %q is not a directory", spec.Label, dir)
	}
	return nil
}

// LoadDir is the kernel's one evaluate-and-shape-gate step. It builds exactly
// one CUE package — the package at src.Pkg under the module root src.Root —
// in cueCtx and runs the artifact shape gate described by spec over the
// result. The two source modes are selected by src.Overlay:
//
//   - Overlay == nil  → on-disk package: load.Config.Dir is the package
//     directory (Root joined with Pkg), which must exist and be a directory
//     ([CheckDir]), and the files are read from the filesystem.
//   - Overlay != nil  → in-memory package: the overlay supplies the .cue files
//     under Root (its cue.mod/module.cue included) and Root doubles as the
//     module root, so the staged cue.mod/module.cue drives transitive
//     dependency resolution; the package is built as "./<Pkg>". A file the
//     overlay does not carry is read from the host filesystem beneath it. This
//     is how a registry-fetched artifact ([FetchArtifact] stages the fetch
//     under a synthetic root and builds it here, for every kind), a module or
//     catalog acquired from a directory, a values-layered instance package and
//     a synthesized instance package are all built.
//
// opts carries the load settings ([Options]). A nil src, or one with no Root,
// is a caller bug and is refused with a plain error.
//
// Keeping this routine single-sourced guarantees an overlay-built artifact and
// an on-disk artifact are evaluated, shape-gated and error-wrapped identically:
// the only difference between the acquire verbs is where the package files come
// from.
func LoadDir(cueCtx *cue.Context, src *opmmodule.Source, opts Options, spec ArtifactSpec) (cue.Value, error) {
	if src == nil || src.Root == "" {
		return cue.Value{}, errors.New("source carries no module root")
	}
	rel := strings.TrimPrefix(src.Pkg, "./")

	cfg := &load.Config{Env: opts.Env}
	// pkgDir is the package directory in both modes, so a build or gate error
	// names the package that failed. root is the directory the load is
	// reported against: the package directory on disk, the module root in
	// overlay mode, where pkg names the package beneath it.
	pkgDir := filepath.Join(src.Root, filepath.FromSlash(rel))
	root, pkg := src.Root, "."
	if src.Overlay == nil {
		root = pkgDir
		if err := CheckDir(root, spec); err != nil {
			return cue.Value{}, err
		}
		cfg.Dir = root
	} else {
		if rel != "" && rel != "." {
			pkg = "./" + rel
		}
		// The staged tree travels as bytes on module.Source and is wrapped
		// here for this build, so no caller of LoadDir deals in load.Source.
		cfg.Dir = src.Root
		cfg.ModuleRoot = src.Root
		cfg.Overlay = make(map[string]load.Source, len(src.Overlay))
		for path, data := range src.Overlay {
			cfg.Overlay[path] = load.FromBytes(data)
		}
	}

	instances := load.Instances([]string{pkg}, cfg)
	if len(instances) != 1 {
		return cue.Value{}, fmt.Errorf("expected exactly one CUE package in %s (%s), found %d: %w", root, pkg, len(instances), oerrors.ErrInvalidPackage)
	}
	if instances[0].Err != nil {
		return cue.Value{}, fmt.Errorf("loading %s package from %s (%s): %w", spec.Label, root, pkg, instances[0].Err)
	}

	val := cueCtx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building %s package from %s: %w", spec.Label, pkgDir, err)
	}

	if err := gate(val, spec); err != nil {
		return cue.Value{}, fmt.Errorf("validating %s package in %s: %w", spec.Label, pkgDir, err)
	}

	return val, nil
}
