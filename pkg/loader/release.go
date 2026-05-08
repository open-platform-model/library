package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/open-platform-model/library/pkg/apiversion"
)

// LoadOptions configures release-file loading behavior.
type LoadOptions struct {
	// Registry overrides the CUE_REGISTRY value used while loading a release
	// file. Empty means use the current process environment.
	//
	// The override is applied via load.Config.Env, NOT os.Setenv, so the
	// loader is safe to call concurrently from a long-running service
	// (Crossplane function, REST API, embedded SDK).
	Registry string
}

// LoadReleaseFile loads a #ModuleRelease from a standalone .cue file.
// CUE imports (including registry module references) are resolved via
// load.Instances() using the file's parent directory for cue.mod resolution.
//
// The returned cue.Value may have #module unfilled if the release file does not
// import a module. The release file must import a module to fill #module.
//
// Returns the evaluated CUE value, the directory used for CUE resolution, and
// the detected apiVersion. An unrecognised or missing apiVersion produces an
// error wrapping apiversion.ErrUnknownAPIVersion; callers are expected to
// reject the artifact rather than rendering with an empty Version.
//
// Deprecated: use Kernel.LoadReleaseFile. The Kernel owns its [*cue.Context]
// and threads cross-cutting dependencies through every operation.
func LoadReleaseFile(ctx *cue.Context, filePath string, opts LoadOptions) (cue.Value, string, apiversion.Version, error) {
	var err error
	filePath, err = resolveReleaseFile(filePath)
	if err != nil {
		return cue.Value{}, "", "", err
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return cue.Value{}, "", "", fmt.Errorf("resolving release file path: %w", err)
	}

	parentDir := filepath.Dir(absPath)

	cfg := &load.Config{
		Dir: parentDir,
		Env: registryEnv(opts.Registry),
	}
	instances := load.Instances([]string{filepath.Base(absPath)}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, "", "", fmt.Errorf("no CUE instances found for %s", absPath)
	}
	if instances[0].Err != nil {
		return cue.Value{}, "", "", fmt.Errorf("loading release file: %w", instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, "", "", fmt.Errorf("building release file: %w", err)
	}

	ver, err := apiversion.Detect(val)
	if err != nil {
		return cue.Value{}, "", "", fmt.Errorf("detecting apiVersion in %s: %w", absPath, err)
	}

	return val, parentDir, ver, nil
}

// LoadValuesFile loads a standalone CUE values file and returns the concrete
// values as a cue.Value. The function first tries to extract a "values" field
// from the loaded file (the standard OPM values file shape); if no such field
// exists the whole evaluated file value is returned instead.
//
// Used by module-only vet validation when -f is provided but there is no
// release.cue in the module directory.
//
// Deprecated: use Kernel.LoadValuesFile. The Kernel owns its [*cue.Context]
// and threads cross-cutting dependencies through every operation.
func LoadValuesFile(ctx *cue.Context, path string) (cue.Value, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return cue.Value{}, fmt.Errorf("resolving values file path: %w", err)
	}
	if _, statErr := os.Stat(absPath); statErr != nil {
		if os.IsNotExist(statErr) {
			return cue.Value{}, fmt.Errorf("values file %q not found", path)
		}
		return cue.Value{}, fmt.Errorf("accessing values file %q: %w", path, statErr)
	}

	parentDir := filepath.Dir(absPath)
	cfg := &load.Config{
		Dir: parentDir,
	}
	instances := load.Instances([]string{filepath.Base(absPath)}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, fmt.Errorf("no CUE instances found for %s", path)
	}
	if instances[0].Err != nil {
		return cue.Value{}, fmt.Errorf("loading values file: %w", instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building values file: %w", err)
	}

	// Standard OPM values files wrap values in a "values" field.
	// Return that field when it exists so the caller gets the raw config value.
	if valuesField := val.LookupPath(cue.ParsePath("values")); valuesField.Exists() && valuesField.Err() == nil {
		return valuesField, nil
	}

	return val, nil
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

// resolveReleaseFile resolves either a release directory or direct file path.
func resolveReleaseFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("release path must not be empty")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("release path %q not found", path)
		}
		return "", fmt.Errorf("stat release path: %w", err)
	}
	if info.IsDir() {
		releasePath := filepath.Join(path, "release.cue")
		if _, err := os.Stat(releasePath); err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("release path %q does not contain release.cue", path)
			}
			return "", fmt.Errorf("stat release file: %w", err)
		}
		return releasePath, nil
	}
	return path, nil
}
