package materialize

import (
	"context"
	"fmt"

	"cuelang.org/go/mod/modconfig"
	"cuelang.org/go/mod/modregistry"
)

// enumerateVersions lists the published versions of a catalog module path
// against the configured registry. It returns the registry's `v`-prefixed,
// SemVer-sorted version forms (e.g. ["v0.1.0", "v0.2.0"]).
//
// DIAGNOSTIC-ONLY (0010 D14): selection never reads this list — a
// subscription's authored `version!` names the one build Materialize pulls.
// Enumeration survives solely to enrich the error when the pull of a named
// build fails ([pullFailureDiagnostic]), reporting what IS published.
//
// path is the subscription key (#ModulePathType). A major-free key (core v1,
// e.g. "opmodel.dev/catalogs/opm") enumerates every published version
// regardless of major. A major-suffixed key ("…/opm@v2", the core-v2 form)
// scopes the list to that major: ModuleVersions splits the suffix and keeps
// only tags within it, so the published list shown to the user never mixes in
// another line's versions. env carries the CUE_REGISTRY mapping via
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
