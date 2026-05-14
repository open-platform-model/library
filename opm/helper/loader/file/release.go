package file

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/open-platform-model/library/opm/apiversion"
)

// LoadOptions configures package-loader behavior shared by LoadModulePackage,
// LoadReleasePackage, and LoadPlatformPackage.
type LoadOptions struct {
	// Registry overrides the CUE_REGISTRY value used while loading. Empty
	// means use the current process environment.
	//
	// The override is applied via load.Config.Env, NOT os.Setenv, so the
	// loader is safe to call concurrently from a long-running service
	// (Crossplane function, REST API, embedded SDK).
	Registry string
}

// LoadReleasePackage loads a #ModuleRelease CUE package from a directory and
// returns the raw cue.Value plus the detected apiVersion. Mirrors
// LoadModulePackage: every .cue file in dirPath that shares the package is
// unified into a single instance, so authors can split a release across
// release.cue, values.cue, and overlay files within one CUE package.
//
// LoadOptions.Registry, when non-empty, is applied via load.Config.Env so
// the release's module imports resolve from the override registry without
// mutating process state.
//
// An unrecognised or missing apiVersion produces an error wrapping
// apiversion.ErrUnknownAPIVersion.
//
// The recommended entry point is Kernel.LoadReleasePackage, which owns its
// [*cue.Context] and threads cross-cutting dependencies through every
// operation. Call this function directly only if you are not using a
// Kernel.
func LoadReleasePackage(ctx *cue.Context, dirPath string, opts LoadOptions) (cue.Value, apiversion.Version, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("resolving release directory: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("accessing release directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return cue.Value{}, "", fmt.Errorf("release path %q is not a directory", absDir)
	}

	cfg := &load.Config{
		Dir: absDir,
		Env: registryEnv(opts.Registry),
	}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) != 1 {
		return cue.Value{}, "", fmt.Errorf("expected exactly one CUE package in %s, found %d: %w", absDir, len(instances), ErrInvalidPackage)
	}
	if instances[0].Err != nil {
		return cue.Value{}, "", fmt.Errorf("loading release package from %s: %w", absDir, instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, "", fmt.Errorf("building release package from %s: %w", absDir, err)
	}

	ver, err := apiversion.Detect(val)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("detecting apiVersion in %s: %w", absDir, err)
	}

	if err := shapeGate(val, releaseSpec); err != nil {
		return cue.Value{}, "", fmt.Errorf("validating release package in %s: %w", absDir, err)
	}

	return val, ver, nil
}

// registryEnv returns a copy of os.Environ() with CUE_REGISTRY overridden if
// registry is non-empty. Returns nil when no override is requested so that
// load.Config falls back to the process environment unchanged.
//
// Building the env slice (rather than calling os.Setenv) keeps the loader
// safe under concurrency: load.Config consumes the slice locally without
// mutating process state.
func registryEnv(registry string) []string {
	if registry == "" {
		return nil
	}
	env := os.Environ()
	override := "CUE_REGISTRY=" + registry
	for i, kv := range env {
		if len(kv) >= len("CUE_REGISTRY=") && kv[:len("CUE_REGISTRY=")] == "CUE_REGISTRY=" {
			env[i] = override
			return env
		}
	}
	return append(env, override)
}
