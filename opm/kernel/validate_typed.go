package kernel

import (
	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/module"
)

// ValidateModuleValues is a typed shortcut for
// [Kernel.ValidateConfig](m.ConfigSchema(), values). It exists so callers
// holding a *module.Module reach the validation surface without looking up
// the #config schema themselves.
func (k *Kernel) ValidateModuleValues(m *module.Module, values cue.Value) (cue.Value, error) {
	return k.ValidateConfig(m.ConfigSchema(), values)
}

// ValidateModuleValuesPartial is the partial-mode counterpart of
// [Kernel.ValidateModuleValues].
func (k *Kernel) ValidateModuleValuesPartial(m *module.Module, values cue.Value) (cue.Value, error) {
	return k.ValidateConfigPartial(m.ConfigSchema(), values)
}

// ValidateModuleValuesDetailed is the layered counterpart of
// [Kernel.ValidateModuleValues] — see [Kernel.ValidateConfigDetailed].
func (k *Kernel) ValidateModuleValuesDetailed(m *module.Module, sources []Source, opts ...ValidateOption) (cue.Value, error) {
	return k.ValidateConfigDetailed(m.ConfigSchema(), sources, opts...)
}

// ValidateInstanceValues is a typed shortcut for
// [Kernel.ValidateConfig](r.ConfigSchema(), values). It resolves the
// embedded source module's #config schema for the caller.
//
// Was: ValidateReleaseValues
func (k *Kernel) ValidateInstanceValues(r *module.Instance, values cue.Value) (cue.Value, error) {
	return k.ValidateConfig(r.ConfigSchema(), values)
}

// ValidateInstanceValuesPartial is the partial-mode counterpart of
// [Kernel.ValidateInstanceValues].
//
// Was: ValidateReleaseValuesPartial
func (k *Kernel) ValidateInstanceValuesPartial(r *module.Instance, values cue.Value) (cue.Value, error) {
	return k.ValidateConfigPartial(r.ConfigSchema(), values)
}

// ValidateInstanceValuesDetailed is the layered counterpart of
// [Kernel.ValidateInstanceValues] — see [Kernel.ValidateConfigDetailed].
//
// Was: ValidateReleaseValuesDetailed
func (k *Kernel) ValidateInstanceValuesDetailed(r *module.Instance, sources []Source, opts ...ValidateOption) (cue.Value, error) {
	return k.ValidateConfigDetailed(r.ConfigSchema(), sources, opts...)
}
