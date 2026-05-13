package file

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
)

// LoadPlatformFile loads a #Platform from a standalone .cue file (or from a
// directory containing platform.cue). CUE imports are resolved via
// load.Instances() using the file's parent directory for cue.mod resolution.
//
// Returns the evaluated CUE value and the directory used for CUE resolution.
// apiVersion validation and binding resolution happen in
// platform.NewPlatformFromValue; this function is the I/O substrate only.
//
// The recommended entry point is Kernel.LoadPlatformFile, which owns its
// [*cue.Context] and threads cross-cutting dependencies through every
// operation. Call this function directly only if you are not using a
// Kernel.
func LoadPlatformFile(ctx *cue.Context, path string, opts LoadOptions) (cue.Value, string, error) {
	resolved, err := resolvePlatformFile(path)
	if err != nil {
		return cue.Value{}, "", err
	}

	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return cue.Value{}, "", fmt.Errorf("resolving platform file path: %w", err)
	}

	parentDir := filepath.Dir(absPath)

	cfg := &load.Config{
		Dir: parentDir,
		Env: registryEnv(opts.Registry),
	}
	instances := load.Instances([]string{filepath.Base(absPath)}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, "", fmt.Errorf("no CUE instances found for %s", absPath)
	}
	if instances[0].Err != nil {
		return cue.Value{}, "", fmt.Errorf("loading platform file: %w", instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, "", fmt.Errorf("building platform file: %w", err)
	}

	return val, parentDir, nil
}

// resolvePlatformFile resolves either a platform directory (containing
// platform.cue) or a direct file path. Mirrors resolveReleaseFile.
func resolvePlatformFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("platform path must not be empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("platform path %q not found", path)
		}
		return "", fmt.Errorf("stat platform path: %w", err)
	}
	if info.IsDir() {
		platformPath := filepath.Join(path, "platform.cue")
		if _, err := os.Stat(platformPath); err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("platform path %q does not contain platform.cue", path)
			}
			return "", fmt.Errorf("stat platform file: %w", err)
		}
		return platformPath, nil
	}
	return path, nil
}
