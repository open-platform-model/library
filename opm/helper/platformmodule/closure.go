package platformmodule

import (
	"context"
	"errors"
	"fmt"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/module"
)

// ModFileSource yields a published module's cue.mod/module.cue. It is the
// one method of [modconfig.Registry] the closure needs, kept as a narrow
// interface so tests can supply a fixture graph without a registry.
type ModFileSource interface {
	ModFile(ctx context.Context, mv module.Version) (*modfile.File, error)
}

// RegistryConfig is the caller-supplied configuration a registry-backed
// [ModFileSource] resolves through. Nothing here is read from the process by
// the helper itself (kernel neutrality): the frontend states every value.
type RegistryConfig struct {
	// Registry is the CUE registry mapping (CUE_REGISTRY syntax) module files
	// resolve through. Empty falls back to CUE_REGISTRY in Env.
	Registry string

	// ClientType is reported to registries in the User-Agent header. Empty
	// falls back to modconfig's own default ("cuelang.org/go").
	ClientType string

	// Env is the environment the CUE module cache location (CUE_CACHE_DIR)
	// and, when Registry is empty, CUE_REGISTRY are read from. Nil selects
	// modconfig's default, the current process environment; passing nil is
	// the caller's explicit choice, never a hidden lookup by the helper.
	Env []string
}

// NewRegistry returns a module-file source resolving through cfg. Module
// files it fetches are the same artifacts a build fetches, so a closure
// derivation never adds an artifact class to the frontend's registry path.
func NewRegistry(cfg RegistryConfig) (ModFileSource, error) {
	reg, err := modconfig.NewRegistry(&modconfig.Config{
		CUERegistry: cfg.Registry,
		ClientType:  cfg.ClientType,
		Env:         cfg.Env,
	})
	if err != nil {
		return nil, fmt.Errorf("configuring module registry: %w", err)
	}
	return reg, nil
}

// Closure derives the generated module's full dependency list from roots: a
// breadth-first walk over each reachable module version's published module
// file, selecting the maximum version per major-qualified path, the roots
// participating in the maximum. This is minimum version selection computed
// the way `cue mod tidy` computes it (0019 D13: tidying happens once, at
// platform-module generation), minus the prune of modules no import reaches,
// which pins a path nothing evaluates and is harmless. Derived entries carry
// no default-major marker; `cue mod tidy` writes none for a platform either,
// because the platform imports nothing unqualified. Local replacements
// (cue.mod/local-module.cue, the "local" path) are skipped.
//
// A root or transitive requirement naming an unpublished build fails with
// an error naming the module path and version, the same wording the CUE
// resolver uses for a missing pin. The walk honours ctx cancellation.
func Closure(ctx context.Context, src ModFileSource, roots []Dep) ([]Dep, error) {
	if src == nil {
		return nil, errors.New("closure needs a module-file source")
	}
	selected := make(map[string]module.Version, len(roots))
	visited := make(map[string]bool, len(roots))
	var queue []module.Version

	push := func(mv module.Version) {
		if cur, ok := selected[mv.Path()]; !ok || mv.Compare(cur) > 0 {
			selected[mv.Path()] = mv
		}
		if !visited[mv.String()] {
			visited[mv.String()] = true
			queue = append(queue, mv)
		}
	}

	for _, r := range roots {
		mv, err := module.NewVersion(r.Path, r.Version)
		if err != nil {
			return nil, fmt.Errorf("dependency root %s@%s: %w", r.Path, r.Version, err)
		}
		push(mv)
	}

	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		mv := queue[0]
		queue = queue[1:]
		mf, err := src.ModFile(ctx, mv)
		if err != nil {
			return nil, fmt.Errorf("resolving dependency %s: %w", mv, err)
		}
		for _, dep := range mf.DepVersions() {
			if dep.IsLocal() {
				continue
			}
			push(dep)
		}
	}

	out := make([]Dep, 0, len(selected))
	for path, mv := range selected {
		out = append(out, Dep{Path: path, Version: mv.Version()})
	}
	sortDeps(out)
	return out, nil
}
