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
	"os"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/module"

	"github.com/open-platform-model/library/opm/helper/loader/internal/shape"
	"github.com/open-platform-model/library/opm/helper/loader/internal/stage"
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
	res, err := LoadModulePackageWithSource(ctx, cueCtx, modPath, version, opts)
	if err != nil {
		return cue.Value{}, err
	}
	return res.Value, nil
}

// StagedSource bundles a registry-loaded module's built value with the staged
// source tree the build used: the deterministic synthetic root every overlay
// key sits under, plus the load.Config.Overlay carrying the module's files
// (including its own cue.mod/module.cue). A consumer can reuse Root + Overlay to
// build a follow-on package INSIDE the module's own main module — letting the
// module's already-tidied cue.mod/module.cue drive transitive resolution —
// without a second registry fetch (Principle V, CUE-native resolution).
type StagedSource struct {
	Value   cue.Value
	Root    string
	Overlay map[string]load.Source
}

// LoadModulePackageWithSource loads a published #Module exactly as
// LoadModulePackage does (same fetch, main-module staging, shape gate) and
// additionally returns the staged source (Root + Overlay) the build used, so
// callers can reuse it. The returned Overlay is the build's own map; callers
// that mutate it (e.g. to overlay additional files) MUST clone it first.
func LoadModulePackageWithSource(ctx context.Context, cueCtx *cue.Context, modPath, version string, opts LoadOptions) (StagedSource, error) {
	mv, err := module.NewVersion(modPath, version)
	if err != nil {
		return StagedSource{}, fmt.Errorf("parsing module version %s@%s: %w", modPath, version, err)
	}

	env := registryEnv(opts.Registry)

	reg, err := modconfig.NewRegistry(&modconfig.Config{Env: env})
	if err != nil {
		return StagedSource{}, fmt.Errorf("building module registry resolver: %w", err)
	}

	// Fetch downloads if necessary and returns the extracted module's source
	// location (the modcache returns {FS: OSDirFS(extractDir), Dir: "."}).
	loc, err := reg.Fetch(ctx, mv)
	if err != nil {
		return StagedSource{}, fmt.Errorf("fetching module %s: %w", mv, err)
	}

	synthRoot, overlay, err := stage.OverlayFromSource(loc, modPath, version)
	if err != nil {
		return StagedSource{}, fmt.Errorf("staging module %s in overlay: %w", mv, err)
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
		return StagedSource{}, fmt.Errorf("expected exactly one CUE package in module %s, found %d: %w", mv, len(instances), shape.ErrInvalidPackage)
	}
	if instances[0].Err != nil {
		return StagedSource{}, fmt.Errorf("loading module package %s: %w", mv, instances[0].Err)
	}

	val := cueCtx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return StagedSource{}, fmt.Errorf("building module package %s: %w", mv, err)
	}

	if err := shape.Gate(val, shape.ModuleSpec); err != nil {
		return StagedSource{}, fmt.Errorf("validating module package %s: %w", mv, err)
	}

	return StagedSource{Value: val, Root: synthRoot, Overlay: overlay}, nil
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
