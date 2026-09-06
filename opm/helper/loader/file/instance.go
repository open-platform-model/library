package file

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/internal/cueenv"
)

// LoadOptions configures package-loader behavior shared by LoadModulePackage,
// LoadInstancePackage, and LoadPlatformPackage.
type LoadOptions struct {
	// Registry overrides the CUE_REGISTRY value used while loading. Empty
	// means use the current process environment.
	//
	// The override is applied via load.Config.Env, NOT os.Setenv, so the
	// loader is safe to call concurrently from a long-running service
	// (Crossplane function, REST API, embedded SDK).
	Registry string
}

// LoadInstancePackage loads a #ModuleInstance CUE package from a directory and
// returns the raw cue.Value. Mirrors LoadModulePackage: every .cue file in
// dirPath that shares the package is unified into a single instance, so
// authors can split an instance across instance.cue, values.cue, and overlay
// files within one CUE package.
//
// LoadOptions.Registry, when non-empty, is applied via load.Config.Env so
// the instance's module imports resolve from the override registry without
// mutating process state.
//
// The recommended entry point is Kernel.LoadInstancePackage, which owns its
// [*cue.Context] and threads cross-cutting dependencies through every
// operation. Call this function directly only if you are not using a
// Kernel.
//
// Was: LoadReleasePackage
func LoadInstancePackage(ctx *cue.Context, dirPath string, opts LoadOptions) (cue.Value, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return cue.Value{}, fmt.Errorf("resolving instance directory: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return cue.Value{}, fmt.Errorf("accessing instance directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return cue.Value{}, fmt.Errorf("instance path %q is not a directory", absDir)
	}

	// Filesystem source: overlay nil selects on-disk loading in the shared
	// build-and-shape-gate step (the same step synth.Instance drives with an
	// in-memory overlay).
	return buildAndShapeGate(ctx, absDir, ".", nil, cueenv.Override(opts.Registry, ""), instanceSpec)
}
