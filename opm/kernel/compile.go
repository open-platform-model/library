package kernel

import (
	"context"
	"fmt"

	"github.com/open-platform-model/library/opm/compile"
	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/module"
)

// compileModuleInstance is the canonical compile pipeline implementation. It
// runs the full Match → Finalize → Execute sequence against the supplied
// instance and platform and returns the resulting [*compile.CompileResult].
//
// Callers go through [Kernel.Compile] (which adds Tier-2 validation in front
// of this helper). This method lives on the kernel so the kernel owns the
// canonical impl and so the pipeline builds every value in the Kernel's own
// [*cue.Context] (k.cueCtx) — consuming the materialized platform's Package as
// read-only input — rather than borrowing the platform's context.
func (k *Kernel) compileModuleInstance(
	ctx context.Context,
	inst *module.Instance,
	mp *materialize.MaterializedPlatform,
	runtimeName string,
) (*compile.CompileResult, error) {
	if runtimeName == "" {
		return nil, fmt.Errorf("runtimeName must be non-empty")
	}
	if inst == nil {
		return nil, fmt.Errorf("instance is required")
	}
	if mp == nil {
		return nil, fmt.Errorf("platform is required")
	}

	schemaComponents := inst.MatchComponents()
	if !schemaComponents.Exists() {
		return nil, fmt.Errorf("instance %q: no components field in instance spec", inst.Metadata.Name)
	}

	dataComponents, err := compile.FinalizeValue(k.cueCtx, schemaComponents)
	if err != nil {
		return nil, fmt.Errorf("finalizing components: %w", err)
	}

	plan, err := compile.Match(schemaComponents, mp, inst.Metadata.Name)
	if err != nil {
		return nil, err
	}

	return compile.NewModule(k.cueCtx, mp, runtimeName).Execute(ctx, inst, schemaComponents, dataComponents, plan)
}
