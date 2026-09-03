package kernel

import (
	"context"
	"errors"
	"fmt"
	"os"

	"cuelang.org/go/cue/cuecontext"

	"github.com/open-platform-model/library/opm/core"
	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/renderstage"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
)

// SkewPolicy is the caller's response to catalog version skew (enhancement
// 0019 D7/D18): the instance module's cue.mod requiring a NEWER build of an
// OPM-namespace path than the platform module carries. Exactly two responses
// exist; the zero value is the default.
type SkewPolicy int

const (
	// SkewWarn renders against the platform's build and reports the skew on
	// [RenderResult.Warnings]. The default.
	SkewWarn SkewPolicy = iota

	// SkewRefuse fails the render before evaluation with an
	// [*oerrors.SkewError] per skewed path.
	SkewRefuse
)

// RenderInput is the input of [Kernel.Render].
type RenderInput struct {
	// Instance is the validated instance to render. It MUST carry a Source
	// (Kernel.SynthesizeInstance, Kernel.AcquireInstanceFromDir): the render
	// build imports the instance as a package, so an evaluated value alone
	// is never sufficient.
	Instance *module.Instance

	// Platform is the platform to render against, in the D5 shape (registry
	// entries carrying their catalog by import). It MUST carry a Source
	// (Kernel.AcquirePlatformFromDir).
	Platform *platform.Platform

	// RuntimeName identifies the executing runtime; it enters the build as
	// #context.#runtimeName and is stamped on every rendered object.
	RuntimeName string

	// Skew is the response to catalog version skew. Zero is [SkewWarn].
	Skew SkewPolicy
}

// RenderResult is the output of a successful [Kernel.Render].
type RenderResult struct {
	// Compiled is the rendered output, one entry per rendered object, in
	// the build's deterministic pair order, each carrying instance,
	// component and transformer provenance.
	Compiled []*core.Compiled

	// Diagnostics are the matching verdicts and version rows decoded from
	// the build.
	Diagnostics RenderDiagnostics

	// Warnings are advisory, human-readable messages: unhandled optional
	// traits and (under [SkewWarn]) version skew. Non-empty is not failure.
	Warnings []string
}

// RenderPair names one matched (component, transformer) pair.
type RenderPair struct {
	Component   string
	Transformer string
}

// ResolvedVersion is one resolved-versions row (0019 D18): for an
// OPM-namespace path the instance module requires, what it asked for and
// what the platform carries. Plain data with no severity; Newer marks the
// skew case the policy decided.
type ResolvedVersion struct {
	// Path is the major-qualified module path.
	Path string

	// ModuleVersion is the build the instance module's cue.mod requires.
	ModuleVersion string

	// PlatformVersion is the build the platform module's cue.mod carries;
	// empty when the platform does not list the path (the instance's own
	// entry then resolves).
	PlatformVersion string

	// Newer is true when the instance requires a newer build than the
	// platform carries.
	Newer bool
}

// RenderDiagnostics is everything the build reports as data (0019 D10),
// decoded into the kernel's structured types. It is populated on success and
// carried by [*RenderError] on a refusal, so a caller can always read the
// full verdict set.
type RenderDiagnostics struct {
	// Pairs is the matched pair set in build order.
	Pairs []RenderPair

	// Unmatched lists components no transformer matched.
	Unmatched []string

	// Unresolved is every demand the platform failed to resolve (0010 D28):
	// an empty bucket (Disqualified empty, Alternatives naming same-base
	// keys the platform does implement) or every candidate disqualified.
	Unresolved []oerrors.UnresolvedDemand

	// Unify is every candidate the always-unify rung disqualified, one entry
	// per conflicting FQN. Cause names the transformer and the FQN; the
	// verbatim CUE cause is not recoverable from inside the build (D10).
	Unify []oerrors.UnifyError

	// UnhandledTraits maps a component to the effectively-optional traits
	// no matched transformer handles (rendered as warnings).
	UnhandledTraits map[string][]string

	// FailedPairs names matched pairs whose transformer output errored.
	FailedPairs []RenderPair

	// OverSubscribed is every provider-fulfilled contract key that
	// transformers from more than one enabled registry entry require (the
	// single-provider guard, 0010 D32/D37), key-sorted. Any row refuses the
	// render through the gate.
	OverSubscribed []oerrors.OverSubscribedContractError

	// ResolvedVersions holds the per-path version rows, in path order.
	ResolvedVersions []ResolvedVersion
}

// RenderError is a refusal after the build: the fail-closed gate (an
// unresolved demand, an unmatched component or an over-subscribed
// provider-fulfilled contract), a failed pair, or a non-concrete pair output.
// Diagnostics carries everything the build reported; Err carries the typed
// causes ([*oerrors.UnresolvedDemandsError], [*compile.UnmatchedComponentsError],
// [oerrors.OverSubscribedContractError], [*oerrors.TransformError]), reachable
// through errors.As.
type RenderError struct {
	Diagnostics RenderDiagnostics
	Err         error
}

func (e *RenderError) Error() string { return "render refused: " + e.Err.Error() }

func (e *RenderError) Unwrap() error { return e.Err }

// Render renders an instance against a platform as ONE CUE build (enhancement
// 0019 D9): it stages a generated render module in a per-render temporary
// directory (the promoted cue.mod, D13; directory replacements bringing both
// inputs in; the embedded matching and execution glue), verifies the
// promoted list covers every OPM-namespace path either input requires,
// applies the skew policy (D7/D18), builds the module once in a fresh
// cue.Context that is dropped when Render returns (D8), and decodes
// `diagnostics` and `rendered` off the built value.
//
// The Kernel's own context is not used and no built value survives the call
// except the returned output; repeated renders share nothing. The staging
// directory is removed on return, success or failure. Registry resolution
// for the platform's catalog imports uses [WithRegistry] when set, else the
// process CUE_REGISTRY, plumbed through the load configuration only.
//
// Refusals before evaluation (missing Source, uncovered OPM path, skew under
// [SkewRefuse]) return plain errors; refusals after evaluation return a
// [*RenderError] carrying the decoded diagnostics.
func (k *Kernel) Render(ctx context.Context, in RenderInput) (*RenderResult, error) {
	if in.Instance == nil {
		return nil, errors.New("RenderInput.Instance is required")
	}
	if in.Instance.Source == nil {
		return nil, fmt.Errorf("instance %q carries no Source: the render build imports the instance as a package (acquire it with SynthesizeInstance or AcquireInstanceFromDir)", in.Instance.Metadata.Name)
	}
	if in.Platform == nil {
		return nil, errors.New("RenderInput.Platform is required")
	}
	if in.Platform.Source == nil {
		return nil, fmt.Errorf("platform %q carries no Source: the render build imports the platform as a package (acquire it with AcquirePlatformFromDir)", platformName(in.Platform))
	}
	if in.RuntimeName == "" {
		return nil, errors.New("RenderInput.RuntimeName must be non-empty")
	}
	if in.Skew != SkewWarn && in.Skew != SkewRefuse {
		return nil, fmt.Errorf("RenderInput.Skew %d is not a SkewPolicy", in.Skew)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	dir, err := os.MkdirTemp("", "opm-render-")
	if err != nil {
		return nil, fmt.Errorf("creating render staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	staged, err := renderstage.Stage(dir, in.Instance.Source, in.Platform.Source, in.RuntimeName)
	if err != nil {
		return nil, fmt.Errorf("staging render module: %w", err)
	}

	rows := make([]ResolvedVersion, 0, len(staged.Skew))
	var warnings []string
	var refusals []error
	for _, r := range staged.Skew {
		rows = append(rows, ResolvedVersion(r))
		if !r.Newer {
			continue
		}
		switch in.Skew {
		case SkewRefuse:
			refusals = append(refusals, &oerrors.SkewError{Path: r.Path, ModuleVersion: r.ModuleVersion, PlatformVersion: r.PlatformVersion})
		default:
			warnings = append(warnings, fmt.Sprintf(
				"version skew on %q: module requires %s, platform carries %s; rendering against the platform's build",
				r.Path, r.ModuleVersion, r.PlatformVersion))
		}
	}
	if len(refusals) > 0 {
		return nil, fmt.Errorf("render refused before evaluation: %w", errors.Join(refusals...))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// One build, one context, dropped with the render (D8).
	built, err := renderstage.Build(cuecontext.New(), staged, renderstage.RegistryEnv(k.registry))
	if err != nil {
		return nil, fmt.Errorf("building render module: %w", err)
	}

	diag, err := decodeRenderDiagnostics(built, rows)
	if err != nil {
		return nil, err
	}
	if gate := gateErrors(diag); gate != nil {
		return nil, &RenderError{Diagnostics: diag, Err: gate}
	}
	compiled, err := decodeRendered(built, diag, in.Instance.Metadata.Name)
	if err != nil {
		return nil, &RenderError{Diagnostics: diag, Err: err}
	}

	warnings = append(warnings, unhandledTraitWarnings(diag.UnhandledTraits)...)
	if warnings == nil {
		warnings = []string{}
	}
	return &RenderResult{Compiled: compiled, Diagnostics: diag, Warnings: warnings}, nil
}

func platformName(p *platform.Platform) string {
	if p != nil && p.Metadata != nil {
		return p.Metadata.Name
	}
	return "<unnamed>"
}
