package renderstage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/open-platform-model/library/opm/internal/sourcetree"
	"github.com/open-platform-model/library/opm/module"
)

// Staged is one render module written to a directory: what the kernel needs
// to build it and to report on the version skew it was staged under.
type Staged struct {
	// Dir is the render module's root: cue.mod/module.cue,
	// cue.mod/local-module.cue and render.cue live here, as does a
	// materialized overlay-mode input tree.
	Dir string

	// Skew holds the per-path resolved-versions rows (D18), instance list
	// against platform list.
	Skew []VersionRow
}

// Stage writes the render module for instance and platform into dir (which
// must exist and be empty): materializes an overlay-mode input under it,
// promotes the two module files, writes the cue.mod pair, verifies OPM-path
// coverage, compares skew, and writes the glue. It performs no build.
func Stage(dir string, instance, platform *module.Source, runtimeName string) (*Staged, error) {
	if instance == nil {
		return nil, errors.New("instance carries no source")
	}
	if platform == nil {
		return nil, errors.New("platform carries no source")
	}
	if runtimeName == "" {
		return nil, errors.New("runtime name must be non-empty")
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving staging directory: %w", err)
	}

	instDir, err := serveDir(absDir, "instance", instance)
	if err != nil {
		return nil, fmt.Errorf("staging instance tree: %w", err)
	}
	platDir, err := serveDir(absDir, "platform", platform)
	if err != nil {
		return nil, fmt.Errorf("staging platform tree: %w", err)
	}

	instMF, err := ReadModFile(instance)
	if err != nil {
		return nil, fmt.Errorf("instance module file: %w", err)
	}
	platMF, err := ReadModFile(platform)
	if err != nil {
		return nil, fmt.Errorf("platform module file: %w", err)
	}

	promotion, err := Promote(platMF, instMF, platDir, instDir)
	if err != nil {
		return nil, fmt.Errorf("promoting dependency lists: %w", err)
	}
	modDir := filepath.Join(absDir, "cue.mod")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating %s: %w", modDir, err)
	}
	moduleBytes, err := promotion.ModuleFile()
	if err != nil {
		return nil, err
	}
	modulePath := filepath.Join(modDir, "module.cue")
	if err := os.WriteFile(modulePath, moduleBytes, 0o644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", modulePath, err)
	}
	localBytes, err := promotion.LocalModuleFile()
	if err != nil {
		return nil, err
	}
	localPath := filepath.Join(modDir, "local-module.cue")
	if err := os.WriteFile(localPath, localBytes, 0o644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", localPath, err)
	}

	// The D13 tripwire: re-read what was written, never the in-memory list.
	written, err := os.ReadFile(modulePath)
	if err != nil {
		return nil, fmt.Errorf("re-reading %s: %w", modulePath, err)
	}
	if err := VerifyCoverage(written, modulePath, map[string]*ModFile{"instance": instMF, "platform": platMF}); err != nil {
		return nil, err
	}

	skew, err := CompareSkew(platMF, instMF)
	if err != nil {
		return nil, fmt.Errorf("comparing version skew: %w", err)
	}

	instPkg, err := sourcetree.PackageName(instance)
	if err != nil {
		return nil, fmt.Errorf("instance package: %w", err)
	}
	platPkg, err := sourcetree.PackageName(platform)
	if err != nil {
		return nil, fmt.Errorf("platform package: %w", err)
	}
	instImport, err := ImportPath(instMF.Module, instance.Pkg, instPkg)
	if err != nil {
		return nil, err
	}
	platImport, err := ImportPath(platMF.Module, platform.Pkg, platPkg)
	if err != nil {
		return nil, err
	}
	glue, err := RenderGlue(GlueInputs{InstancePath: instImport, PlatformPath: platImport, RuntimeName: runtimeName})
	if err != nil {
		return nil, err
	}
	gluePath := filepath.Join(absDir, RenderFileName)
	if err := os.WriteFile(gluePath, glue, 0o644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", gluePath, err)
	}

	return &Staged{Dir: absDir, Skew: skew}, nil
}

// Build evaluates the staged render module exactly once in cueCtx and returns
// the built value. env is the environment slice cue/load consults (nil for
// the process environment). A load failure (an import that does not resolve,
// a malformed module file) is returned as an error; an evaluation error on
// the built value is NOT, because the fail-closed gate is one such error and
// the kernel reads `diagnostics` beside it.
func Build(cueCtx *cue.Context, staged *Staged, env []string) (cue.Value, error) {
	if cueCtx == nil || staged == nil {
		return cue.Value{}, errors.New("build needs a context and a staged module")
	}
	cfg := &load.Config{
		Dir:        staged.Dir,
		ModuleRoot: staged.Dir,
		Env:        env,
	}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) != 1 {
		return cue.Value{}, fmt.Errorf("expected exactly one CUE package in the render module, found %d", len(instances))
	}
	if instances[0].Err != nil {
		return cue.Value{}, fmt.Errorf("loading the render module: %w", instances[0].Err)
	}
	return cueCtx.BuildInstance(instances[0]), nil
}

// serveDir returns the absolute directory cue/load serves src from: its own
// Root in on-disk mode, or a fresh subdirectory of dir into which the overlay
// is materialized.
func serveDir(dir, name string, src *module.Source) (string, error) {
	if src.Root == "" {
		return "", errors.New("source carries no module root")
	}
	if src.Overlay == nil {
		root, err := filepath.Abs(src.Root)
		if err != nil {
			return "", fmt.Errorf("resolving %s: %w", src.Root, err)
		}
		return root, nil
	}
	target := filepath.Join(dir, name)
	if _, err := src.WriteTo(target); err != nil {
		return "", err
	}
	return target, nil
}
