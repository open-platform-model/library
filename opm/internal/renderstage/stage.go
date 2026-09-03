package renderstage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"

	"github.com/open-platform-model/library/opm/module"
)

// Staged is one render module written to a directory, with everything the
// kernel needs to build it and to report on how it was derived.
type Staged struct {
	// Dir is the render module's root: cue.mod/module.cue,
	// cue.mod/local-module.cue and render.cue live here, as does a
	// materialized overlay-mode input tree.
	Dir string

	// Instance and Platform are the two committed module files the render
	// module was promoted from.
	Instance, Platform *ModFile

	// Promotion is the derived dependency list (D13).
	Promotion *Promotion

	// Skew holds the per-path resolved-versions rows (D18), instance list
	// against platform list.
	Skew []VersionRow

	// InstanceImport and PlatformImport are the generated import paths.
	InstanceImport, PlatformImport string
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

	instPkg, err := packageName(instance)
	if err != nil {
		return nil, fmt.Errorf("instance package: %w", err)
	}
	platPkg, err := packageName(platform)
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

	return &Staged{
		Dir:            absDir,
		Instance:       instMF,
		Platform:       platMF,
		Promotion:      promotion,
		Skew:           skew,
		InstanceImport: instImport,
		PlatformImport: platImport,
	}, nil
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

// RegistryEnv returns a copy of the process environment with CUE_REGISTRY
// overridden when registry is non-empty, and nil otherwise (cue/load then
// reads the process environment unchanged). Never os.Setenv.
func RegistryEnv(registry string) []string {
	if registry == "" {
		return nil
	}
	env := os.Environ()
	override := "CUE_REGISTRY=" + registry
	for i, kv := range env {
		if strings.HasPrefix(kv, "CUE_REGISTRY=") {
			env[i] = override
			return env
		}
	}
	return append(env, override)
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
	root := filepath.Clean(src.Root)
	for key, entry := range src.Overlay {
		rel, err := filepath.Rel(root, key)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("overlay entry %s is outside the source root %s", key, root)
		}
		data, err := sourceBytes(entry)
		if err != nil {
			return "", fmt.Errorf("overlay entry %s: %w", key, err)
		}
		out := filepath.Join(target, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return "", fmt.Errorf("creating %s: %w", filepath.Dir(out), err)
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return "", fmt.Errorf("writing %s: %w", out, err)
		}
	}
	return target, nil
}

// packageName returns the package clause shared by the .cue files of the
// source's package directory. Files without a package clause are not part of
// the package and are skipped; two clauses disagreeing is an error.
func packageName(src *module.Source) (string, error) {
	pkgDir := filepath.Join(src.Root, filepath.FromSlash(src.Pkg))
	files, err := packageFiles(src, pkgDir)
	if err != nil {
		return "", err
	}
	names := map[string]bool{}
	for _, f := range files {
		data, err := readSourceFile(src, f)
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
			if filepath.Dir(key) == pkgDir && strings.HasSuffix(key, ".cue") {
				files = append(files, key)
			}
		}
	} else {
		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", pkgDir, err)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".cue") {
				files = append(files, filepath.Join(pkgDir, e.Name()))
			}
		}
	}
	sort.Strings(files)
	return files, nil
}
