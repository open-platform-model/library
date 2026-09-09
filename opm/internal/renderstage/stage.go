package renderstage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/open-platform-model/library/opm/internal/sourcetree"
	"github.com/open-platform-model/library/opm/module"
)

// Staged is one render module written to a directory: what the kernel needs
// to build it and to report on the version skew it was staged under.
type Staged struct {
	// Dir is the render module's root: cue.mod/module.cue,
	// cue.mod/local-module.cue and render.cue live here, and nothing else.
	Dir string

	// Overlay holds every file of an overlay-mode input, keyed under Dir at
	// the directory its local-module.cue replacement names (<Dir>/instance,
	// <Dir>/platform). Build serves them to cue/load through
	// load.Config.Overlay; none of them is written to Dir. Empty when both
	// inputs are on disk.
	Overlay map[string][]byte

	// Skew holds the per-path resolved-versions rows (D18), instance list
	// against platform list.
	Skew []VersionRow

	// Replacements holds one row per local replacement the render module
	// honours (an input's own cue.mod/local-module.cue, promoted), in path
	// order. Nil unless the caller enabled local replacements and an input
	// carried one that promotion kept.
	Replacements []ReplacementRow
}

// Stage writes the render module for instance and platform into dir (which
// must exist and be empty): promotes the two module files, writes the
// cue.mod pair, verifies OPM-path coverage, compares skew, and writes the
// glue. An overlay-mode input is not written: its entries are re-keyed under
// dir onto Staged.Overlay for Build to serve from memory, and an on-disk
// input is referenced in place. It performs no build.
//
// localReplacements decides what an input's own cue.mod/local-module.cue
// means: true promotes its replacements into the render module's
// main-module view (platform's whole, instance's on paths the platform does
// not name) and reports them on Staged.Replacements; false refuses, before
// anything is written, an input whose file carries a replacement, since
// silently dropping the file is what made a developer's redirection invisible
// at render time. An input without the file stages identically either way.
func Stage(dir string, instance, platform *module.Source, runtimeName string, localReplacements bool) (*Staged, error) {
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

	overlay := map[string][]byte{}
	instDir, err := serveDir(absDir, "instance", instance, overlay)
	if err != nil {
		return nil, fmt.Errorf("staging instance tree: %w", err)
	}
	platDir, err := serveDir(absDir, "platform", platform, overlay)
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
	instLocal, err := ReadLocalModFile(instance, instMF)
	if err != nil {
		return nil, fmt.Errorf("instance local module file: %w", err)
	}
	platLocal, err := ReadLocalModFile(platform, platMF)
	if err != nil {
		return nil, fmt.Errorf("platform local module file: %w", err)
	}
	if !localReplacements {
		for _, in := range []struct {
			by    string
			mod   string
			local *LocalModFile
		}{{byPlatform, platMF.Module, platLocal}, {byInstance, instMF.Module, instLocal}} {
			if in.local != nil && len(in.local.Replacements) > 0 {
				return nil, fmt.Errorf("%s %q carries %s with replacements; the caller did not enable local replacements", in.by, in.mod, LocalModFileName)
			}
		}
		instLocal, platLocal = nil, nil
	}

	promotion, err := Promote(platMF, instMF, platLocal, instLocal, platDir, instDir)
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

	return &Staged{Dir: absDir, Overlay: overlay, Skew: skew, Replacements: promotion.Rows}, nil
}

// Build evaluates the staged render module exactly once in cueCtx and returns
// the built value. env is the environment slice cue/load consults (nil for
// the process environment). The overlay-mode inputs Stage collected are
// handed to cue/load as load.Config.Overlay, so their replacement directories
// are served from memory. A load failure (an import that does not resolve,
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
	if len(staged.Overlay) > 0 {
		cfg.Overlay = make(map[string]load.Source, len(staged.Overlay))
		for path, data := range staged.Overlay {
			cfg.Overlay[path] = load.FromBytes(data)
		}
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
// Root in on-disk mode, or <dir>/<name> in overlay mode. Nothing is written
// for an overlay-mode source: every entry is re-keyed from src.Root to that
// directory and added to overlay, so the replacement exists only in the
// build's overlay filesystem. An entry outside src.Root is refused, the same
// check the library's overlay writer applies.
func serveDir(dir, name string, src *module.Source, overlay map[string][]byte) (string, error) {
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
	for key, data := range src.Overlay {
		rel, err := filepath.Rel(root, key)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("overlay entry %s is outside the source root %s", key, root)
		}
		overlay[filepath.Join(target, rel)] = data
	}
	return target, nil
}
