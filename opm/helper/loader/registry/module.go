// Package registry loads a published #Module from an OCI registry by
// path@version, the registry-sourced sibling of opm/helper/loader/file.
//
// It is opt-in convenience under the opm/helper/ boundary: a frontend MAY skip
// it and resolve registry modules another way. The recommended entry point is
// Kernel.LoadModuleFromRegistry, which owns its *cue.Context and threads the
// kernel's configured registry through the call.
package registry

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/module"

	"github.com/open-platform-model/library/opm/helper/loader/internal/shape"
)

// LoadOptions configures the registry module loader. Mirrors
// opm/helper/loader/file.LoadOptions so the two loaders are interchangeable
// from a caller's perspective.
type LoadOptions struct {
	// Registry overrides the CUE_REGISTRY value used while fetching and
	// loading. Empty means use the current process environment.
	//
	// The override is applied via the modconfig resolver and load.Config.Env,
	// NOT os.Setenv, so the loader is safe to call concurrently from a
	// long-running service (Crossplane function, controller, embedded SDK).
	Registry string
}

// LoadModulePackage loads a #Module published in an OCI registry, identified by
// its major-qualified module path (e.g. "example.com/modules/hello@v0") and
// version (e.g. "v0.0.2"), and returns the raw cue.Value built in cueCtx.
//
// It fetches the module's source via CUE's native module machinery
// (mod/modconfig) and loads it IN MEMORY AS THE MAIN MODULE: the fetched files
// are injected through load.Config.Overlay under a deterministic synthetic
// root, so the module's own cue.mod/module.cue drives transitive dependency
// resolution and its kind/metadata are evaluated at the package root. No
// wrapper package is synthesized and no temporary directory is written.
//
// The built value is validated with the same module shape gate as
// opm/helper/loader/file (concrete kind == "Module"; concrete metadata.name,
// metadata.modulePath, metadata.version), wrapping the shared
// ErrInvalidPackage / ErrWrongKind / ErrMissingRequiredField sentinels. It does
// NOT perform full schema validation, which remains the Kernel/Binding layer's
// contract.
//
// Mirrors opm/helper/loader/file.LoadModulePackage's return shape
// (cue.Value, error); apiVersion detection no longer lives at the loader layer
// (the opm/apiversion package was removed). The process environment is never
// mutated. Parse failures on caller input are wrapped rather than panicked.
func LoadModulePackage(ctx context.Context, cueCtx *cue.Context, modPath, version string, opts LoadOptions) (cue.Value, error) {
	mv, err := module.NewVersion(modPath, version)
	if err != nil {
		return cue.Value{}, fmt.Errorf("parsing module version %s@%s: %w", modPath, version, err)
	}

	env := registryEnv(opts.Registry)

	reg, err := modconfig.NewRegistry(&modconfig.Config{Env: env})
	if err != nil {
		return cue.Value{}, fmt.Errorf("building module registry resolver: %w", err)
	}

	// Fetch downloads if necessary and returns the extracted module's source
	// location (the modcache returns {FS: OSDirFS(extractDir), Dir: "."}).
	loc, err := reg.Fetch(ctx, mv)
	if err != nil {
		return cue.Value{}, fmt.Errorf("fetching module %s: %w", mv, err)
	}

	synthRoot, overlay, err := overlayFromSource(loc, modPath, version)
	if err != nil {
		return cue.Value{}, fmt.Errorf("staging module %s in overlay: %w", mv, err)
	}

	// Overlay (with FS left nil), NOT load.Config.FS. The spike confirmed that
	// pinning load.Config.FS to the fetched module's SourceLoc.FS FAILS on
	// transitive deps: the loader then reads ALL source — including deps — only
	// through that one FS, and the module's catalog/core deps live in separate
	// cache directories ("cannot find package opmodel.dev/catalogs/opm/resources").
	// Overlay injects only the target module's files while leaving normal
	// registry/cache dependency resolution intact. Do not "simplify" this to
	// FS-pinning. See design.md § Research & Decisions (add-registry-module-loader).
	cfg := &load.Config{
		Dir:        synthRoot,
		ModuleRoot: synthRoot,
		Overlay:    overlay,
		Env:        env,
	}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) != 1 {
		return cue.Value{}, fmt.Errorf("expected exactly one CUE package in module %s, found %d: %w", mv, len(instances), shape.ErrInvalidPackage)
	}
	if instances[0].Err != nil {
		return cue.Value{}, fmt.Errorf("loading module package %s: %w", mv, instances[0].Err)
	}

	val := cueCtx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building module package %s: %w", mv, err)
	}

	if err := shape.Gate(val, shape.ModuleSpec); err != nil {
		return cue.Value{}, fmt.Errorf("validating module package %s: %w", mv, err)
	}

	return val, nil
}

// overlayFromSource reads every file in the fetched module's source location
// into a load.Config.Overlay keyed under a deterministic synthetic absolute
// root, returning that root and the overlay. The root is derived from
// path@version and need not exist on the real filesystem; the loader treats the
// overlaid files as if present there (and all parent dirs as existing).
func overlayFromSource(loc module.SourceLoc, modPath, version string) (string, map[string]load.Source, error) {
	synthRoot := syntheticRoot(modPath, version)
	overlay := map[string]load.Source{}

	root := loc.Dir
	if root == "" {
		root = "."
	}

	err := fs.WalkDir(loc.FS, root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(loc.FS, p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		// loc.FS uses io/fs slash paths; compute the path relative to the
		// module root and rebase it under the synthetic OS root.
		rel := p
		if root != "." {
			rel = strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
		}
		key := filepath.Join(synthRoot, filepath.FromSlash(rel))
		overlay[key] = load.FromBytes(data)
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	if len(overlay) == 0 {
		return "", nil, fmt.Errorf("fetched module source is empty: %w", shape.ErrInvalidPackage)
	}
	return synthRoot, overlay, nil
}

// syntheticRoot returns a deterministic absolute path used as the in-memory
// module root for the overlay. It is derived purely from path@version (no
// randomness, no clock) so the load is reproducible, and is sanitized into a
// single path segment so it never collides with real source on disk.
func syntheticRoot(modPath, version string) string {
	repl := strings.NewReplacer("/", "_", ":", "_", "@", "_", "+", "_")
	safe := repl.Replace(modPath + "@" + version)
	return string(filepath.Separator) + filepath.Join("opm-registry-module", safe)
}

// registryEnv returns a copy of os.Environ() with CUE_REGISTRY overridden if
// registry is non-empty. Returns the process environment unchanged when no
// override is requested. Building the env slice (rather than calling os.Setenv)
// keeps the loader safe under concurrency: modconfig and load.Config consume
// the slice locally without mutating process state. Mirrors
// opm/helper/loader/file.registryEnv and opm/materialize.resolverEnv.
func registryEnv(registry string) []string {
	base := os.Environ()
	if registry == "" {
		return base
	}
	env := make([]string, 0, len(base)+1)
	seen := false
	for _, e := range base {
		if strings.HasPrefix(e, "CUE_REGISTRY=") {
			env = append(env, "CUE_REGISTRY="+registry)
			seen = true
			continue
		}
		env = append(env, e)
	}
	if !seen {
		env = append(env, "CUE_REGISTRY="+registry)
	}
	return env
}
