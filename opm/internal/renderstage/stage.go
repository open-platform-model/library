package renderstage

import (
	"errors"
	"fmt"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/loader"
	"github.com/open-platform-model/library/opm/internal/sourcetree"
	"github.com/open-platform-model/library/opm/module"
)

// RenderRoot is the synthetic root every render module is staged under: an
// absolute path directly beneath the file-system root (/opm-render on Unix;
// on Windows, on the current volume). Nothing is written there: the
// generated module and every overlay-mode input are served to the build from
// [Staged.Overlay], keyed under it. cue/load merges a real directory's
// entries into the overlay, so [Stage] refuses when anything exists at the
// path. It is the same for every render, which is safe because an overlay
// belongs to one load.Instances call and nothing in cue/load keys
// process-wide state on the main module's root (the module cache keys by
// module@version), and it makes the positions in a render build error the
// same from one render to the next. It is a variable only so a test can
// point it at an existing directory; nothing else assigns it.
var RenderRoot = sourcetree.VolumeRoot("opm-render")

// Staged is one render module staged in memory: what the kernel needs to
// build it and to report on the version skew it was staged under.
type Staged struct {
	// Dir is the render module's root, [RenderRoot]. Nothing exists there on
	// disk; cue.mod/module.cue, cue.mod/local-module.cue and render.cue are
	// entries of Overlay under it.
	Dir string

	// Overlay always carries the generated module: cue.mod/module.cue,
	// cue.mod/local-module.cue and render.cue under Dir. It also holds every
	// file of an overlay-mode input, keyed under Dir at the directory its
	// local-module.cue replacement names (<Dir>/instance, <Dir>/platform).
	// Build serves all of it to cue/load through load.Config.Overlay; an
	// on-disk input is not in it and is read from its own directory.
	Overlay map[string][]byte

	// Skew holds the per-path resolved-versions rows (0019:D18), instance list
	// against platform list.
	Skew []VersionRow

	// Replacements holds one row per local replacement the render module
	// honours (an input's own cue.mod/local-module.cue, promoted), in path
	// order. Nil unless the caller enabled local replacements and an input
	// carried one that promotion kept.
	Replacements []ReplacementRow
}

// StageOptions are the caller's per-render switches Stage honours.
type StageOptions struct {
	// LocalReplacements decides what an input's own cue.mod/local-module.cue
	// means: true promotes its replacements into the render module's
	// main-module view (platform's whole, instance's on paths the platform
	// does not name) and reports them on Staged.Replacements; false refuses,
	// before anything is staged, an input whose file carries a replacement,
	// since silently dropping the file is what made a developer's redirection
	// invisible at render time. An input without the file stages identically
	// either way.
	LocalReplacements bool

	// SkipUnprovided is written into the glue as a literal: true moves every
	// unprovided provider-fulfilled demand out of the refusal and onto the
	// skipped verdict, inside the build.
	SkipUnprovided bool
}

// Stage stages the render module for instance and platform in memory under
// [RenderRoot]: it promotes the two module files, places the cue.mod pair on
// Staged.Overlay, verifies OPM-path coverage against the module.cue bytes the
// build will be served, compares skew, and places the glue. An overlay-mode
// input's entries are re-keyed under RenderRoot onto Staged.Overlay, and an
// on-disk input is referenced in place. It writes nothing to the filesystem
// and performs no build.
//
// opts carries the caller's per-render switches; see [StageOptions].
func Stage(instance, platform *module.Source, runtimeName string, opts StageOptions) (*Staged, error) {
	if instance == nil {
		return nil, errors.New("instance carries no source")
	}
	if platform == nil {
		return nil, errors.New("platform carries no source")
	}
	if runtimeName == "" {
		return nil, errors.New("runtime name must be non-empty")
	}
	root := RenderRoot
	if err := sourcetree.CheckRootAbsent("render root", "render module", root); err != nil {
		return nil, err
	}

	overlay := map[string][]byte{}
	instDir, err := serveDir(root, "instance", instance, overlay)
	if err != nil {
		return nil, fmt.Errorf("staging instance tree: %w", err)
	}
	platDir, err := serveDir(root, "platform", platform, overlay)
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
	if !opts.LocalReplacements {
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
	modDir := filepath.Join(root, "cue.mod")
	moduleBytes, err := promotion.ModuleFile()
	if err != nil {
		return nil, err
	}
	modulePath := filepath.Join(modDir, "module.cue")
	overlay[modulePath] = moduleBytes
	localBytes, err := promotion.LocalModuleFile()
	if err != nil {
		return nil, err
	}
	overlay[filepath.Join(root, filepath.FromSlash(LocalModFileName))] = localBytes

	// The 0019:D13 tripwire: re-parse the bytes the build is served, never
	// the in-memory list.
	if err := VerifyCoverage(overlay[modulePath], modulePath, map[string]*ModFile{"instance": instMF, "platform": platMF}); err != nil {
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
	glue, err := RenderGlue(GlueInputs{InstancePath: instImport, PlatformPath: platImport, RuntimeName: runtimeName, SkipUnprovided: opts.SkipUnprovided})
	if err != nil {
		return nil, err
	}
	overlay[filepath.Join(root, RenderFileName)] = glue

	return &Staged{Dir: root, Overlay: overlay, Skew: skew, Replacements: promotion.Rows}, nil
}

// Build evaluates the staged render module exactly once in cueCtx and returns
// the built value. opts carries the load settings: Env, the environment
// slice cue/load consults (nil for the process environment), and Registry,
// the registry the build resolves its dependencies through (nil lets
// cue/load build one from Env). Staged.Overlay (the generated module and the
// overlay-mode inputs) is handed to cue/load as load.Config.Overlay, so the
// main module and those replacement directories are served from memory and
// nothing under Staged.Dir needs to exist on disk. A load failure (an import
// that does not resolve, a malformed module file) is returned as an error;
// an evaluation error on the built value is NOT, because the fail-closed
// gate is one such error and the kernel reads `diagnostics` beside it.
func Build(cueCtx *cue.Context, staged *Staged, opts loader.Options) (cue.Value, error) {
	if cueCtx == nil || staged == nil {
		return cue.Value{}, errors.New("build needs a context and a staged module")
	}
	cfg := &load.Config{
		Dir:        staged.Dir,
		ModuleRoot: staged.Dir,
		Env:        opts.Env,
	}
	if opts.Registry != nil {
		cfg.Registry = opts.Registry
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
		return cue.Value{}, fmt.Errorf("loading the render module: %w", oerrors.Classify(instances[0].Err))
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
		if err != nil || !filepath.IsLocal(rel) {
			return "", fmt.Errorf("overlay entry %s is outside the source root %s", key, root)
		}
		overlay[filepath.Join(target, rel)] = data
	}
	return target, nil
}
