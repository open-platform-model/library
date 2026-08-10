package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
)

// DefaultSchemaModule is the module identifier used by [OCILoader.Load] when
// [OCILoader.Module] is empty. The default tracks the v0 major; CUE resolves
// the floating major to the latest v0.x.y available at first-load time.
const DefaultSchemaModule = "opmodel.dev/core@v1"

// PublicRegistry is the documented CUE_REGISTRY mapping for resolving the
// OPM core schema from its canonical GHCR location with a fallback to
// registry.cue.works. The library does NOT auto-apply this value as a
// default; callers opt in by setting CUE_REGISTRY=schema.PublicRegistry
// (or by passing it via [OCILoader.Registry]).
//
// Operators in restricted environments may set CUE_REGISTRY to a mirror or
// to an inline configuration without touching this constant.
const PublicRegistry = "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"

// Loader resolves the OPM core CUE schema and returns it as a built
// [cue.Value]. Implementations MUST return a value whose definitions
// (#Module, #ModuleInstance, #Platform, #Resource, #Trait,
// #ComponentTransformer, …) are reachable via LookupPath.
//
// The library exposes exactly one Loader implementation: [OCILoader]. Any
// other Loader satisfying the interface is internal-only and MUST NOT
// appear in the public API surface.
type Loader interface {
	Load(ctx *cue.Context) (cue.Value, error)
}

// OCILoader resolves the OPM core schema through CUE's module system. It
// is the canonical and only public [Loader] implementation.
//
// The zero value is a valid Loader: empty fields resolve via process
// environment (CUE_REGISTRY, CUE_CACHE_DIR) and the [DefaultSchemaModule]
// identifier. Explicit field values override environment values.
//
// OCILoader.Load does not mutate process state (no os.Setenv); env
// overrides are plumbed into [load.Config.Env] for the single load call.
type OCILoader struct {
	// Module is the schema module identifier. Empty means
	// [DefaultSchemaModule] ("opmodel.dev/core@v1").
	//
	// A bare major form ("…@v0") is automatically expanded to "…@v0.latest"
	// before calling [load.Instances]; CUE's standalone-package loader
	// requires either a fully qualified version, "@latest", or
	// "<major>.latest" outside a module context.
	Module string

	// Registry overrides CUE_REGISTRY for this load. Empty inherits from
	// the process environment.
	Registry string

	// CacheDir overrides CUE_CACHE_DIR for this load. Empty inherits from
	// the process environment (or CUE's default ~/.cache/cuelang/).
	CacheDir string
}

// Load implements [Loader].
func (l OCILoader) Load(ctx *cue.Context) (cue.Value, error) {
	val, _, err := l.loadVersioned(ctx)
	return val, err
}

// loadVersioned is the package-internal entry point used by [Cache.Get]
// to capture the resolved schema version alongside the value. External
// callers see only [OCILoader.Load].
func (l OCILoader) loadVersioned(ctx *cue.Context) (cue.Value, string, error) {
	if ctx == nil {
		return cue.Value{}, "", fmt.Errorf("schema OCILoader: nil *cue.Context")
	}

	moduleID := l.Module
	if moduleID == "" {
		moduleID = DefaultSchemaModule
	}

	loadID := moduleID
	if isBareMajorVersion(loadID) {
		// load.Instances rejects bare major versions outside a module
		// context; expand "…@vN" → "…@vN.latest" so the standalone-package
		// loader resolves the latest patch within the major.
		loadID = loadID + ".latest"
	}

	cfg := &load.Config{Env: mergeEnv(os.Environ(), l.Registry, l.CacheDir)}
	instances := load.Instances([]string{loadID}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, "", fmt.Errorf("schema OCILoader: load.Instances returned no instances for %q", moduleID)
	}
	if instances[0].Err != nil {
		return cue.Value{}, "", fmt.Errorf("schema OCILoader: loading %q: %w", moduleID, instances[0].Err)
	}

	val := ctx.BuildInstance(instances[0])
	version := resolvedVersionFromInstanceDir(instances[0].Dir)
	return val, version, nil
}

// bareMajorRE matches a module identifier whose version suffix is just a
// bare major like "@v0" or "@v12" (no minor / patch / .latest).
var bareMajorRE = regexp.MustCompile(`@v\d+$`)

func isBareMajorVersion(moduleID string) bool {
	return bareMajorRE.MatchString(moduleID)
}

// versionPathRE matches a trailing "@vMAJOR.MINOR.PATCH" segment on a
// module cache extract path. Used to recover the resolved version from
// [build.Instance.Dir] (which CUE materializes as
// <cacheDir>/extract/<base-path>@<version>) for diagnostics.
var versionPathRE = regexp.MustCompile(`@v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

func resolvedVersionFromInstanceDir(dir string) string {
	if dir == "" {
		return ""
	}
	base := filepath.Base(dir)
	match := versionPathRE.FindString(base)
	if match == "" {
		return ""
	}
	return strings.TrimPrefix(match, "@")
}

// mergeEnv returns a copy of base with CUE_REGISTRY / CUE_CACHE_DIR
// overridden when the corresponding override is non-empty. The base slice
// is not mutated. Empty overrides inherit the underlying environment.
func mergeEnv(base []string, registry, cacheDir string) []string {
	env := make([]string, 0, len(base)+2)
	seenRegistry := false
	seenCacheDir := false
	for _, e := range base {
		switch {
		case registry != "" && strings.HasPrefix(e, "CUE_REGISTRY="):
			env = append(env, "CUE_REGISTRY="+registry)
			seenRegistry = true
		case cacheDir != "" && strings.HasPrefix(e, "CUE_CACHE_DIR="):
			env = append(env, "CUE_CACHE_DIR="+cacheDir)
			seenCacheDir = true
		default:
			env = append(env, e)
		}
	}
	if registry != "" && !seenRegistry {
		env = append(env, "CUE_REGISTRY="+registry)
	}
	if cacheDir != "" && !seenCacheDir {
		env = append(env, "CUE_CACHE_DIR="+cacheDir)
	}
	return env
}
