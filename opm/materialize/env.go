package materialize

import (
	"os"
	"strings"
)

// resolverEnv returns a copy of the process environment with CUE_REGISTRY
// overridden by registry when non-empty. The process environment is never
// mutated (no os.Setenv) — the mapping is plumbed into the modconfig resolver
// and load.Config.Env for the operation only, mirroring opm/schema's loader
// (Principle I). An empty registry inherits the process CUE_REGISTRY.
func resolverEnv(registry string) []string {
	base := os.Environ()
	if registry == "" {
		return base
	}
	env := make([]string, 0, len(base)+1)
	seen := false
	for _, e := range base {
		if strings.HasPrefix(e, "CUE_REGISTRY=") {
			env = append(env, "CUE_REGISTRY="+registry)
			seen = true
			continue
		}
		env = append(env, e)
	}
	if !seen {
		env = append(env, "CUE_REGISTRY="+registry)
	}
	return env
}
