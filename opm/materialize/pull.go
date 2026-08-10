package materialize

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
)

// pullCatalog loads and builds a single catalog module at an exact version
// against the configured registry, reusing the same cue/load mechanism the
// schema OCILoader uses (Config.Env, no os.Setenv). It returns the built
// catalog value (a `c.#Catalog & {…}` package value whose top-level
// #transformers / metadata fields are reachable via LookupPath).
//
// octx MUST be the owner's *cue.Context so the result can be filled onto the
// platform value built with the same context. version is the `v`-prefixed
// module version (e.g. "v0.1.0"). Build/load failures return a plain error;
// the caller wraps them as a MaterializeError with subscription context.
func pullCatalog(octx *cue.Context, env []string, path, version string) (cue.Value, error) {
	loadID := path + "@" + version
	cfg := &load.Config{Env: env}
	instances := load.Instances([]string{loadID}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, fmt.Errorf("load.Instances returned no instances for %q", loadID)
	}
	if instances[0].Err != nil {
		return cue.Value{}, fmt.Errorf("loading %q: %w", loadID, instances[0].Err)
	}
	val := octx.BuildInstance(instances[0])
	if err := val.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building %q: %w", loadID, err)
	}
	return val, nil
}
