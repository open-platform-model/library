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

// ValidateReleaseValues is a typed shortcut for
// [Kernel.ValidateConfig](r.ConfigSchema(), values). It resolves the
// embedded source module's #config schema for the caller.
func (k *Kernel) ValidateReleaseValues(r *module.Release, values cue.Value) (cue.Value, error) {
	return k.ValidateConfig(r.ConfigSchema(), values)
}

// ValidateReleaseValuesPartial is the partial-mode counterpart of
// [Kernel.ValidateReleaseValues].
func (k *Kernel) ValidateReleaseValuesPartial(r *module.Release, values cue.Value) (cue.Value, error) {
	return k.ValidateConfigPartial(r.ConfigSchema(), values)
}

// ValidateReleaseValuesDetailed is the layered counterpart of
// [Kernel.ValidateReleaseValues] — see [Kernel.ValidateConfigDetailed].
func (k *Kernel) ValidateReleaseValuesDetailed(r *module.Release, sources []Source, opts ...ValidateOption) (cue.Value, error) {
	return k.ValidateConfigDetailed(r.ConfigSchema(), sources, opts...)
}
