package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/open-platform-model/library/pkg/apiversion"
)

// LoadModulePackage loads a module CUE package from a directory and returns
// the raw cue.Value plus the detected apiVersion. Used by callers (e.g. opm
// module vet) to load a module for validation or rendering.
//
// An unrecognised or missing apiVersion produces an error wrapping
// apiversion.ErrUnknownAPIVersion.
//
// Deprecated: use Kernel.LoadModulePackage. The Kernel owns its [*cue.Context]
// and threads cross-cutting dependencies through every operation.
func LoadModulePackage(ctx *cue.Context, dirPath string) (cue.Value, apiversion.Version, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("resolving module directory: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("accessing module directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return cue.Value{}, "", fmt.Errorf("module path %q is not a directory", absDir)
	}

	cfg := &load.Config{
		Dir: absDir,
	}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, "", fmt.Errorf("no CUE instances found in %s", absDir)
	}
	if instances[0].Err != nil {
		return cue.Value{}, "", fmt.Errorf("loading module package from %s: %w", absDir, instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, "", fmt.Errorf("building module package from %s: %w", absDir, err)
	}

	ver, err := apiversion.Detect(val)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("detecting apiVersion in %s: %w", absDir, err)
	}

	return val, ver, nil
}
