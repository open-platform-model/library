package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

// LoadDir is the kernel's one evaluate-and-shape-gate step. It builds exactly
// one CUE package — pkg, relative to the module root at root — in ctx and runs
// the artifact shape gate described by spec over the result. The two source
// modes are selected by overlay:
//
//   - overlay == nil  → on-disk package: load.Config.Dir is root and the files
//     are read from the filesystem. root must exist and be a directory.
//   - overlay != nil  → in-memory package: the overlay supplies the .cue files
//     under root (its cue.mod/module.cue included; the set cue/load reads) and
//     root doubles as the module root, so the staged cue.mod/module.cue drives
//     transitive dependency resolution. This is how a registry-fetched module,
//     a values-layered instance package and a synthesized instance package are
//     all built.
//
// pkg is a package path relative to root ("." or "" for the root package,
// "./sub" for a subdirectory). env, when non-nil, is the environment slice
// load.Config consults — the CUE_REGISTRY override the kernel plumbs through
// [cueenv.Override], never os.Setenv, so LoadDir is safe under concurrency.
//
// Keeping this routine single-sourced guarantees an overlay-built artifact and
// an on-disk artifact are evaluated, shape-gated and error-wrapped identically:
// the only difference between the acquire verbs is where the package files come
// from.
func LoadDir(ctx *cue.Context, root, pkg string, overlay map[string][]byte, env []string, spec ArtifactSpec) (cue.Value, error) {
	if pkg == "" {
		pkg = "."
	}

	if overlay == nil {
		// The on-disk mode is the caller-facing one: report a bad path as a
		// path problem rather than as "matched no packages".
		dir := root
		if rel := strings.TrimPrefix(pkg, "./"); rel != "." {
			dir = filepath.Join(root, filepath.FromSlash(rel))
		}
		info, err := os.Stat(dir)
		if err != nil {
			return cue.Value{}, fmt.Errorf("accessing %s directory %q: %w", spec.Label, dir, err)
		}
		if !info.IsDir() {
			return cue.Value{}, fmt.Errorf("%s path %q is not a directory", spec.Label, dir)
		}
	}

	cfg := &load.Config{
		Dir: root,
		Env: env,
	}
	if overlay != nil {
		// The one place the library hands cue/load an overlay: the staged tree
		// travels as bytes on module.Source and is wrapped here, so no caller
		// deals in load.Source.
		cfg.Overlay = make(map[string]load.Source, len(overlay))
		for path, data := range overlay {
			cfg.Overlay[path] = load.FromBytes(data)
		}
		cfg.ModuleRoot = root
	}

	instances := load.Instances([]string{pkg}, cfg)
	if len(instances) != 1 {
		return cue.Value{}, fmt.Errorf("expected exactly one CUE package in %s (%s), found %d: %w", root, pkg, len(instances), oerrors.ErrInvalidPackage)
	}
	if instances[0].Err != nil {
		return cue.Value{}, fmt.Errorf("loading %s package from %s (%s): %w", spec.Label, root, pkg, instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building %s package from %s: %w", spec.Label, root, err)
	}

	if err := Gate(val, spec); err != nil {
		return cue.Value{}, fmt.Errorf("validating %s package in %s: %w", spec.Label, root, err)
	}

	return val, nil
}
