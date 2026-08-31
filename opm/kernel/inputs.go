package kernel

import (
	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/module"
)

// MatchInput is the input for [Kernel.Match]. The instance artifact is the
// sole module-side handle and is matched as processed: values validation and
// filling happen in [Kernel.ProcessModuleInstance] before either phase runs.
type MatchInput struct {
	// ModuleInstance supplies the components value via
	// [module.Instance.MatchComponents]. Required.
	ModuleInstance *module.Instance

	// Platform is the materialized platform whose #composedTransformers and
	// #matchers index drive the matcher. Required. Callers MUST Materialize a
	// *platform.Platform before invoking these phases.
	Platform *materialize.MaterializedPlatform
}

// CompileInput is the input for [Kernel.Compile]. The instance artifact is
// the sole module-side handle and is rendered as processed: values validation
// and filling happen in [Kernel.ProcessModuleInstance], and Compile performs
// no validation pass of its own.
type CompileInput struct {
	// ModuleInstance supplies instance-level metadata and components.
	// Required.
	ModuleInstance *module.Instance

	// Platform is the materialized platform whose #composedTransformers and
	// #matchers index drive the matcher. Required. Callers MUST Materialize a
	// *platform.Platform before invoking these phases.
	Platform *materialize.MaterializedPlatform

	// RuntimeName identifies the runtime executing this compile (e.g.
	// "opm-cli", "opm-controller"). MUST be non-empty.
	RuntimeName string
}
