package file

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/open-platform-model/library/opm/apiversion"
)

// LoadPlatformPackage loads a #Platform CUE package from a directory and
// returns the raw cue.Value plus the detected apiVersion. Mirrors
// LoadModulePackage and LoadReleasePackage: every .cue file in dirPath that
// shares the package is unified into a single instance, so the platform is
// identified by its CUE package clause rather than a platform.cue filename.
//
// LoadOptions.Registry, when non-empty, is applied via load.Config.Env so
// the platform's transitive imports resolve from the override registry
// without mutating process state.
//
// An unrecognised or missing apiVersion produces an error wrapping
// apiversion.ErrUnknownAPIVersion.
//
// The recommended entry point is Kernel.LoadPlatformPackage, which owns its
// [*cue.Context] and threads cross-cutting dependencies through every
// operation. Call this function directly only if you are not using a
// Kernel.
func LoadPlatformPackage(ctx *cue.Context, dirPath string, opts LoadOptions) (cue.Value, apiversion.Version, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("resolving platform directory: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("accessing platform directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return cue.Value{}, "", fmt.Errorf("platform path %q is not a directory", absDir)
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
		return cue.Value{}, "", fmt.Errorf("loading platform package from %s: %w", absDir, instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, "", fmt.Errorf("building platform package from %s: %w", absDir, err)
	}

	ver, err := apiversion.Detect(val)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("detecting apiVersion in %s: %w", absDir, err)
	}

	if err := shapeGate(val, platformSpec); err != nil {
		return cue.Value{}, "", fmt.Errorf("validating platform package in %s: %w", absDir, err)
	}

	return val, ver, nil
}
