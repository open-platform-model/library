package kernel

import (
	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/module"
	"github.com/open-platform-model/library/pkg/provider"
)

// ValidateInput is the input for [Kernel.Validate].
type ValidateInput struct {
	// Module supplies the `#config` schema via its Package. Required.
	Module *module.Module

	// ModuleRelease provides the release context (name, namespace) used
	// in diagnostic messages. Required.
	ModuleRelease *module.Release

	// Values is the user-supplied values cue.Value to validate. The zero
	// cue.Value is treated as "no values" and Validate returns nil without
	// running schema checks.
	Values cue.Value
}

// MatchInput is the input for [Kernel.Match].
type MatchInput struct {
	// Module is the source module. May be nil.
	Module *module.Module

	// ModuleRelease supplies the components value via
	// [module.Release.MatchComponents]. Required.
	ModuleRelease *module.Release

	// Provider supplies the transformer registry. Required.
	Provider *provider.Provider
}

// PlanInput is the input for [Kernel.Plan].
type PlanInput struct {
	// Module supplies the `#config` schema and module-level metadata.
	// Required.
	Module *module.Module

	// ModuleRelease supplies release-level metadata and components.
	// Required.
	ModuleRelease *module.Release

	// Values is the user-supplied values cue.Value to validate. Optional;
	// the zero cue.Value skips validation.
	Values cue.Value

	// Provider supplies the transformer registry. Required.
	Provider *provider.Provider

	// RuntimeName identifies the runtime executing this plan (e.g.
	// "opm-cli", "opm-controller"). MUST be non-empty.
	RuntimeName string
}

// CompileInput is the input for [Kernel.Compile].
type CompileInput struct {
	// Module supplies the `#config` schema and module-level metadata.
	// Required.
	Module *module.Module

	// ModuleRelease supplies release-level metadata and components.
	// Required.
	ModuleRelease *module.Release

	// Values is the user-supplied values cue.Value to validate. Optional;
	// the zero cue.Value skips validation.
	Values cue.Value

	// Provider supplies the transformer registry. Required.
	Provider *provider.Provider

	// RuntimeName identifies the runtime executing this compile (e.g.
	// "opm-cli", "opm-controller"). MUST be non-empty.
	RuntimeName string
}
