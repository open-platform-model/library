package platform

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/module"
	pkgplatform "github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

// CueContextOwner is the minimal contract [Compose] needs from the kernel:
// access to the CUE context that owns the shell Platform's [cue.Value]. The
// [opm/kernel.Kernel] type satisfies this interface, so call sites pass
// `*kernel.Kernel` directly. Tests that build values outside a kernel may
// pass any type exposing a [*cue.Context]. The interface lives here so
// opm/helper/platform does not import opm/kernel and we avoid a cycle with
// the kernel wrapper that delegates back to this package.
type CueContextOwner interface {
	CueContext() *cue.Context
}

// Compose returns a fresh [*pkgplatform.Platform] built by FillPath-injecting
// each Module into shell.Package at schema.Registry[<id>], where <id> is the
// Module's metadata.name. enabled is set to true explicitly even though the
// schema's default is true — the explicit value records intent and survives
// any future default change.
//
// Inputs are not mutated. Calling Compose twice with the same inputs is
// idempotent: the returned Platforms are semantically identical.
//
// Transformer-FQN collisions across registered Modules unify naturally at
// the #composedTransformers map: identical bodies are no-ops, divergent
// bodies surface as a CUE evaluation error that Compose returns verbatim.
//
// The shell is expected to carry the schema's #Platform definition so the
// computed views (#composedTransformers, #matchers) re-evaluate after
// FillPath; a hand-rolled shell with concrete empty views will populate
// #registry correctly but its views will not update.
func Compose(owner CueContextOwner, shell *pkgplatform.Platform, modules []*module.Module) (*pkgplatform.Platform, error) {
	if owner == nil {
		return nil, fmt.Errorf("kernel is required")
	}
	if shell == nil {
		return nil, fmt.Errorf("shell platform is required")
	}
	regSels := schema.Registry.Selectors()

	composed := shell.Package
	for i, m := range modules {
		if m == nil {
			return nil, fmt.Errorf("modules[%d] is nil", i)
		}
		if m.Metadata == nil || m.Metadata.Name == "" {
			return nil, fmt.Errorf("modules[%d] has no metadata.name; cannot derive registry id", i)
		}
		entryPath := cue.MakePath(append(append([]cue.Selector{}, regSels...), cue.Str(m.Metadata.Name))...)
		registration := buildRegistration(owner.CueContext(), m.Package)
		composed = composed.FillPath(entryPath, registration)
	}

	if err := composed.Validate(cue.Concrete(false)); err != nil {
		return nil, fmt.Errorf("composing platform: %w", err)
	}

	return pkgplatform.NewPlatformFromValue(owner, composed)
}

// buildRegistration constructs the [cue.Value] for a single
// #ModuleRegistration entry: { #module: <module value>, enabled: true }.
// The value is built in the supplied context (typically the kernel's) so
// the FillPath call below operates on values bound to the same context as
// the shell. Module values bound to a different context are imported into
// this one transparently by CUE.
func buildRegistration(ctx *cue.Context, modVal cue.Value) cue.Value {
	base := ctx.CompileString(`enabled: true`)
	return base.FillPath(cue.MakePath(cue.Def("module")), modVal)
}
