package materialize

import (
	"context"
	"fmt"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modregistry"
)

// enumerateVersions lists the published versions of a catalog module path
// against the configured registry. It returns the registry's `v`-prefixed,
// SemVer-sorted version forms (e.g. ["v0.1.0", "v0.2.0"]); the filter
// (D4) normalizes the `v`-prefix against the bare-SemVer catalog FQN form.
//
// path is the bare subscription path (#ModulePathType, e.g.
// "opmodel.dev/catalogs/opm"); a bare path enumerates every published version
// regardless of major. env carries the CUE_REGISTRY mapping via
// [resolverEnv]; no process environment is mutated.
func enumerateVersions(ctx context.Context, env []string, path string) ([]string, error) {
	resolver, err := modconfig.NewResolver(&modconfig.Config{Env: env})
	if err != nil {
		return nil, fmt.Errorf("building module resolver: %w", err)
	}
	client := modregistry.NewClientWithResolver(resolver)
	versions, err := client.ModuleVersions(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("listing versions for %q: %w", path, err)
	}
	return versions, nil
}
