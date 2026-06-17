package file

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
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
// returns the raw cue.Value. Mirrors LoadModulePackage: every .cue file in
// dirPath that shares the package is unified into a single instance, so
// authors can split a release across release.cue, values.cue, and overlay
// files within one CUE package.
//
// LoadOptions.Registry, when non-empty, is applied via load.Config.Env so
// the release's module imports resolve from the override registry without
// mutating process state.
//
// The recommended entry point is Kernel.LoadReleasePackage, which owns its
// [*cue.Context] and threads cross-cutting dependencies through every
// operation. Call this function directly only if you are not using a
// Kernel.
func LoadReleasePackage(ctx *cue.Context, dirPath string, opts LoadOptions) (cue.Value, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return cue.Value{}, fmt.Errorf("resolving release directory: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return cue.Value{}, fmt.Errorf("accessing release directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return cue.Value{}, fmt.Errorf("release path %q is not a directory", absDir)
	}

	// Filesystem source: overlay nil selects on-disk loading in the shared
	// build-and-shape-gate step (the same step synth.Release drives with an
	// in-memory overlay).
	return buildAndShapeGate(ctx, absDir, nil, registryEnv(opts.Registry), releaseSpec)
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
