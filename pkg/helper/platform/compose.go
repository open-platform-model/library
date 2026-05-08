package platform

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/api"
	"github.com/open-platform-model/library/pkg/module"
	pkgplatform "github.com/open-platform-model/library/pkg/platform"
)

// CueContextOwner is the minimal contract [Compose] needs from the kernel:
// access to the CUE context that owns the shell Platform's [cue.Value]. The
// [pkg/kernel.Kernel] type satisfies this interface, so call sites pass
// `*kernel.Kernel` directly. Tests that build values outside a kernel may
// pass any type exposing a [*cue.Context]. The interface lives here so
// pkg/helper/platform does not import pkg/kernel and we avoid a cycle with
// the kernel wrapper that delegates back to this package.
type CueContextOwner interface {
	CueContext() *cue.Context
}

// Compose returns a fresh [*pkgplatform.Platform] built by FillPath-injecting
// each Module into shell.Package at binding.Paths().Registry[<id>], where
// <id> is the Module's metadata.name (catalog 014 D16). enabled is set to
// true explicitly even though the schema's default is true — the explicit
// value records intent and survives any future default change.
//
// Inputs are not mutated. Calling Compose twice with the same inputs is
// idempotent: the returned Platforms are semantically identical.
//
// If the composed value violates the multi-fulfiller constraint
// (catalog 014 D13 — at most one transformer may fulfil any given
// primitive FQN at the platform layer), Compose returns
// [*MultiFulfillerError]. The error carries parsed FQN/module/transformer
// attribution when available; otherwise it wraps the raw CUE diagnostic so
// callers can still surface it.
//
// The shell Platform's APIVersion field selects the binding used to
// resolve the registry path; bindings missing the v1alpha2-equivalent
// Paths().Registry fail eagerly via [api.Lookup]. The shell is expected
// to carry the schema's #Platform definition so the computed views
// (#composedTransformers, #matchers, #knownResources, #knownTraits)
// re-evaluate after FillPath; a hand-rolled shell with concrete empty
// views will populate #registry correctly but its views will not update.
func Compose(owner CueContextOwner, shell *pkgplatform.Platform, modules []*module.Module) (*pkgplatform.Platform, error) {
	if owner == nil {
		return nil, fmt.Errorf("kernel is required")
	}
	if shell == nil {
		return nil, fmt.Errorf("shell platform is required")
	}
	b, err := api.Lookup(shell.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("resolving binding for shell apiVersion %q: %w", shell.APIVersion, err)
	}
	paths := b.Paths()
	regSels := paths.Registry.Selectors()

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
		if mf := classifyMultiFulfiller(err); mf != nil {
			return nil, mf
		}
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

// classifyMultiFulfiller inspects a CUE evaluation error for the signature
// of catalog 014's _noMultiFulfiller hard-fail constraint. A positive match
// returns a [*MultiFulfillerError] wrapping the raw error; structured
// attribution (FQN, modules, transformers) is left empty when the diagnostic
// does not carry enough surface to extract them safely. Returns nil when
// the error is not a multi-fulfiller violation.
//
// The detection is intentionally conservative: catalog 014 names the
// constraint _noMultiFulfiller and the resulting diagnostic includes that
// field name in the failure path. Frontends that need richer attribution
// can re-evaluate against #PlatformBase (which exposes #matchers._invalid
// without the constraint short-circuit), but that is out of scope here.
func classifyMultiFulfiller(err error) *MultiFulfillerError {
	if err == nil {
		return nil
	}
	if !strings.Contains(err.Error(), "_noMultiFulfiller") {
		return nil
	}
	return &MultiFulfillerError{rawErr: err}
}
