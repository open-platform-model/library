package kernel

import (
	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/module"
)

// ValidateInput is the input for [Kernel.Validate].
type ValidateInput struct {
	// Module supplies the `#config` schema via its Package. Required.
	Module *module.Module

	// ModuleInstance provides the instance context (name, namespace) used
	// in diagnostic messages. Required.
	ModuleInstance *module.Instance

	// Values is the user-supplied values cue.Value to validate. The zero
	// cue.Value is treated as "no values" and Validate returns nil without
	// running schema checks.
	Values cue.Value
}

// MatchInput is the input for [Kernel.Match]. The instance artifact is the
// sole module-side handle: the source module, when needed, is reachable via
// `ModuleInstance.Package` at the binding's `Paths().Module`.
type MatchInput struct {
	// ModuleInstance supplies the components value via
	// [module.Instance.MatchComponents]. Required.
	ModuleInstance *module.Instance

	// Platform is the materialized platform whose #composedTransformers and
	// #matchers index drive the matcher. Required. Callers MUST Materialize a
	// *platform.Platform before invoking these phases.
	Platform *materialize.MaterializedPlatform
}

// PlanInput is the input for [Kernel.Plan]. The instance artifact is the sole
// module-side handle: the `#config` schema and module-level metadata are
// reachable via `ModuleInstance.ConfigSchema()` and the binding's
// `Paths().ModuleMetadata`.
type PlanInput struct {
	// ModuleInstance supplies instance-level metadata and components.
	// Required.
	ModuleInstance *module.Instance

	// Values is the user-supplied values cue.Value to validate. Optional;
	// the zero cue.Value skips validation.
	Values cue.Value

	// Platform is the materialized platform whose #composedTransformers and
	// #matchers index drive the matcher. Required. Callers MUST Materialize a
	// *platform.Platform before invoking these phases.
	Platform *materialize.MaterializedPlatform

	// RuntimeName identifies the runtime executing this plan (e.g.
	// "opm-cli", "opm-controller"). MUST be non-empty.
	RuntimeName string
}

// CompileInput is the input for [Kernel.Compile]. The instance artifact is
// the sole module-side handle: the `#config` schema and module-level metadata
// are reachable via `ModuleInstance.ConfigSchema()` and the binding's
// `Paths().ModuleMetadata`.
type CompileInput struct {
	// ModuleInstance supplies instance-level metadata and components.
	// Required.
	ModuleInstance *module.Instance

	// Values is the user-supplied values cue.Value to validate. Optional;
	// the zero cue.Value skips validation.
	Values cue.Value

	// Platform is the materialized platform whose #composedTransformers and
	// #matchers index drive the matcher. Required. Callers MUST Materialize a
	// *platform.Platform before invoking these phases.
	Platform *materialize.MaterializedPlatform

	// RuntimeName identifies the runtime executing this compile (e.g.
	// "opm-cli", "opm-controller"). MUST be non-empty.
	RuntimeName string
}
