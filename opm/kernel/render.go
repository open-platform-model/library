package kernel

import (
	"context"
	"errors"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/cueenv"
	"github.com/open-platform-model/library/opm/internal/renderstage"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
)

// SkewPolicy is the caller's response to catalog version skew
// (0019:D7/D18): the instance module's cue.mod requiring a NEWER build of an
// OPM-namespace path than the platform module carries. Exactly two responses
// exist; the zero value is the default.
type SkewPolicy int

// The skew policies a caller can choose.
const (
	// SkewWarn renders against the platform's build and marks that path's
	// row on [RenderDiagnostics.ResolvedVersions] as Newer. The default;
	// the wording of any advisory is the frontend's.
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

	// Platform is the platform to render against, in the 0019:D5 shape (registry
	// entries carrying their catalog by import). It MUST carry a Source
	// (Kernel.AcquirePlatformFromDir).
	Platform *platform.Platform

	// RuntimeName identifies the executing runtime; it enters the build as
	// #context.#runtimeName and is stamped on every rendered object.
	RuntimeName string

	// Skew is the response to catalog version skew. Zero is [SkewWarn].
	Skew SkewPolicy

	// LocalReplacements enables an input's own cue.mod/local-module.cue
	// replacements (a developer redirecting a dependency to a directory or
	// another module) for this render. Off, the default, refuses an input
	// whose file carries a replacement rather than silently rendering
	// against the published pin. On, the replacements are promoted into
	// the render module under the precedence dependencies get (the
	// platform's whole, the instance's only for paths the platform does
	// not name) and reported on [RenderDiagnostics.Replacements]. This is
	// the one switch that lets a render read a directory an artifact names;
	// a frontend sets it for a developer's checkout, never for an artifact
	// it did not author.
	LocalReplacements bool

	// SkipUnprovided renders what the platform can when a component demands
	// a provider-fulfilled contract that no enabled catalog provides (zero
	// providers). Off, the default, refuses such a render as always. On, a
	// skipped trait demand leaves its component rendering every pair it
	// matched; a skipped resource demand omits the whole component (a partly
	// satisfied component never renders) without reporting it unmatched.
	// Every skipped demand is a row on [RenderDiagnostics.Skipped]. Every
	// other refusal stands: a catalog-fulfilled unresolved demand, a
	// provider that exists but did not match, an over-subscribed contract,
	// an unmatched component. The decision is made inside the build, so the
	// render module's own gate agrees with the kernel under both values.
	SkipUnprovided bool
}

// Compiled is the terminal output of an OPM render: [Kernel.Render] emits
// *Compiled values carrying the rendered CUE value plus OPM provenance. It
// carries no platform-native fields — keeping platform vocabulary out of the
// kernel keeps it platform-neutral, and each consumer wraps *Compiled in its
// own resource type. A Kubernetes consumer uses opm/k8s/object.Resource, the
// Kubernetes tier's wrapper; the kernel never imports it.
type Compiled struct {
	// Value is the CUE value produced by the transformer. Concrete and
	// fully evaluated — safe to encode directly to YAML or JSON.
	Value cue.Value

	// Instance is the name of the ModuleInstance that produced this resource.
	Instance string

	// Component is the source component name within the instance.
	Component string

	// Transformer is the FQN of the transformer that produced this resource.
	Transformer string
}

// RenderResult is the output of a successful [Kernel.Render].
type RenderResult struct {
	// Compiled is the rendered output, one entry per rendered object, in
	// the build's deterministic pair order, each carrying instance,
	// component and transformer provenance.
	Compiled []*Compiled

	// Diagnostics are the matching verdicts and version rows decoded from
	// the build.
	Diagnostics RenderDiagnostics
}

// RenderPair names one matched (component, transformer) pair.
type RenderPair struct {
	Component   string
	Transformer string
}

// ResolvedVersion is one resolved-versions row (0019:D18): for an
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

// RenderDiagnostics is everything the build reports as data (0019:D10),
// decoded into the kernel's structured types. It is populated on success and
// carried by [*RenderError] on a refusal, so a caller can always read the
// full verdict set. Every field is a row the build emitted, in the build's
// order; the kernel derives, joins and re-sorts nothing.
//
// It also holds the three advisory facts a render can report, as rows rather
// than as messages: an unhandled optional trait is on UnhandledTraits, a
// module requiring a newer build than the platform carries is a
// ResolvedVersions row with Newer set, and a demand skipped under
// [RenderInput.SkipUnprovided] is a Skipped row. A frontend words all three.
type RenderDiagnostics struct {
	// Pairs is the matched pair set in build order.
	Pairs []RenderPair

	// Unmatched lists components no transformer matched, each carrying
	// every candidate the demand walk reached for it.
	Unmatched []oerrors.UnmatchedComponent

	// Unresolved is every demand the platform failed to resolve (0010:D28):
	// an empty bucket (Disqualified empty, Alternatives naming same-base
	// keys the platform does implement) or every candidate disqualified.
	Unresolved []oerrors.UnresolvedDemand

	// Skipped is every demand the render skipped under
	// [RenderInput.SkipUnprovided], in build order (every component's
	// resource rows, then every component's trait rows). Always empty when
	// the switch is off. Advisory: a skipped demand never refuses, and a
	// frontend words it.
	Skipped []SkippedDemand

	// Unify is every candidate the always-unify rung disqualified, one row
	// per (component, transformer) carrying the FQNs it conflicted at. The
	// verbatim CUE cause is not recoverable from inside the build (0019:D10).
	Unify []oerrors.UnifyRefusal

	// UnhandledTraits maps a component to the effectively-optional traits
	// no matched transformer handles. Advisory: a frontend formats it.
	UnhandledTraits map[string][]string

	// FailedPairs names the matched pairs whose transformer output is an
	// error, in pair order. It is filled on every RenderError raised after
	// the build, a gate refusal included, and is empty on a successful
	// render. An output that is only non-concrete is not listed.
	FailedPairs []RenderPair

	// OverSubscribed is every provider-fulfilled contract key that
	// transformers of two or more enabled registry entries (path plus
	// major) require (the single-provider guard, 0010:D32/D37), key-sorted,
	// each row naming the registry keys core's #contracts.providedBy holds
	// for it. The count is core's, the one the platform's Contracts() reads,
	// so the rows are exactly its OverSubscribed. Any row refuses the render
	// through the gate.
	OverSubscribed []oerrors.OverSubscribedContract

	// Collisions is every contract key two or more enabled registry
	// entries' catalogs list (core's #contracts.collisions), key-sorted,
	// each row naming the registry keys core's #contracts.collidingEntries
	// holds for it: exactly the platform inventory's CollidingEntries. Any
	// row refuses the render through the gate, first among the causes,
	// whatever the instance and whatever [RenderInput.SkipUnprovided] says.
	// Empty on a platform pinning a core without the report, which cannot
	// evaluate a colliding platform at all.
	Collisions []oerrors.ContractCollision

	// Routable is core's #contracts.routable as decoded: false when the
	// platform carries an over-subscribed or colliding contract. A false
	// verdict with neither row refuses the render with
	// [*oerrors.NotRoutableError].
	Routable bool

	// RequiredContracts is every contract key a component of the instance
	// requires: each component's #resources and #traits keys, sorted in
	// byte order and deduplicated (0013:D24). It counts every component of
	// the render, one omitted under [RenderInput.SkipUnprovided] included,
	// and it is not narrowed to provider-fulfilled contracts. Empty, never
	// nil, for an instance with no components.
	//
	// It is computed in the build and set on every RenderResult and every
	// [*RenderError]. A render that returns a plain error (a refusal before
	// evaluation, a build error or a diagnostics decode failure) carries no
	// demand, so a caller tracking it keeps its last known list there.
	//
	// It fails closed: a component whose #resources is missing or does not
	// evaluate fails the render with a plain error instead of reading as no
	// demand. #traits is read behind the matcher's presence test, so an
	// absent #traits contributes nothing; a #traits that is a top-level
	// conflict never reaches a successful render, because the instance's
	// components do not evaluate.
	RequiredContracts []string

	// ResolvedVersions holds the per-path version rows, in path order.
	ResolvedVersions []ResolvedVersion

	// Replacements holds one row per local replacement the render honoured
	// under [RenderInput.LocalReplacements], in path order; nil otherwise.
	// A replaced path keeps its pinned versions on ResolvedVersions; the
	// row says where its bytes were served from. Advisory data a frontend
	// words (an instance replacement the platform's list made inert is not
	// here, so a frontend computes the inert set from its own file).
	Replacements []Replacement
}

// SkippedDemand is one provider-fulfilled demand skipped under
// [RenderInput.SkipUnprovided] because nothing on the platform provides it.
// Data, like every diagnostics row.
type SkippedDemand struct {
	// Component is the demanding component.
	Component string

	// FQN is the demanded contract key.
	FQN string

	// Kind is "resource" or "trait".
	Kind string

	// DefinedBy is the registry key of the enabled catalog listing the key
	// in its contract maps; empty when none does.
	DefinedBy string

	// Alternatives is the same-base contract-key set the platform does
	// implement at another apiVersion, in the build's ladder order.
	Alternatives []string

	// ComponentOmitted is true on every skipped row of a component that
	// rendered nothing because it has a skipped resource demand, trait rows
	// included, so a frontend can say "not rendered" once per component.
	ComponentOmitted bool
}

// Replacement is one honoured local replacement: the replaced major-qualified
// module path, its target (an absolute directory, or a module path for a
// module replacement) and the input whose cue.mod/local-module.cue supplied
// it, "platform" or "instance".
type Replacement struct {
	Path   string
	Target string
	By     string
}

// RenderError is a refusal after the build: the fail-closed gate (a contract
// collision, an unresolved demand, an over-subscribed provider-fulfilled
// contract, an unmatched component, or a not-routable platform no row
// explains), a failed pair, or a non-concrete pair output. Diagnostics
// carries everything the build reported; Err carries the typed causes
// ([*oerrors.ContractCollisionsError], [*oerrors.UnresolvedDemandsError],
// [*oerrors.OverSubscribedContractsError],
// [*oerrors.UnmatchedComponentsError], [*oerrors.NotRoutableError],
// [*oerrors.TransformError]), reachable through errors.As; the gate causes
// are joined in that order.
type RenderError struct {
	Diagnostics RenderDiagnostics
	Err         error
}

// Error returns "render refused: " followed by the message of Err.
func (e *RenderError) Error() string { return "render refused: " + e.Err.Error() }

// Unwrap returns Err, so errors.As and errors.Is reach the typed causes it
// carries.
func (e *RenderError) Unwrap() error { return e.Err }

// Render renders an instance against a platform as ONE CUE build
// (0019:D9): it stages a generated render module in memory, under a
// synthetic root that exists nowhere on disk (the promoted cue.mod,
// 0019:D13; directory replacements bringing both inputs in, an on-disk input
// in place and an overlay-mode input from memory; the embedded matching and
// execution glue), verifies the promoted list covers
// every OPM-namespace path either input requires, applies the skew policy
// (0019:D7/D18), builds the module once in a fresh cue.Context whose
// references the kernel drops when Render returns (0019:D8), and decodes
// `diagnostics` and `rendered` off the built value.
//
// The Kernel holds no context of its own, and no built value survives the
// call except the returned output; repeated renders share nothing. A render
// writes no staging file, success or failure: the generated module reaches
// the build through the load overlay. The CUE module cache under
// CUE_CACHE_DIR is filled when the build fetches a dependency, as for any
// load. Registry resolution
// for the platform's catalog imports uses [WithRegistry] when set, else the
// process CUE_REGISTRY, plumbed through the load configuration only.
//
// Refusals before evaluation (missing Source, a platform whose core predates
// the provider count, uncovered OPM path, skew under [SkewRefuse]) return
// plain errors; refusals after evaluation return a [*RenderError] carrying
// the decoded diagnostics. A platform whose #contracts carries no
// providedBy (a platform module pinning core older than
// [schema.ProvidedBySince]) is refused before staging, with nothing staged,
// by the error [platform.Platform.CoreFloor] returns,
// wrapping [*oerrors.PlatformCoreTooOldError]: the render never falls back
// to a provider count of its own. The floor is the fact the platform
// recorded at construction; Render reads no Package, only the platform's
// Metadata and Source. That holds for a platform the constructor built; a
// Platform built as a struct literal decodes its Package once, on its first
// CoreFloor or Contracts call (see [platform.Platform]).
func (k *Kernel) Render(ctx context.Context, in RenderInput) (*RenderResult, error) {
	_, res, err := k.render(ctx, in)
	return res, err
}

// render is Render with the built value exposed. The value is returned on
// every path that reached the build, refusal included, so a test can assert
// that the render module's own `gate` agrees with the kernel's verdict; the
// exported verb drops it, since no built value survives a render (0019:D8).
func (k *Kernel) render(ctx context.Context, in RenderInput) (cue.Value, *RenderResult, error) {
	var none cue.Value
	if in.Instance == nil {
		return none, nil, errors.New("RenderInput.Instance is required")
	}
	if in.Instance.Source == nil {
		return none, nil, fmt.Errorf("instance %q carries no Source: the render build imports the instance as a package (acquire it with SynthesizeInstance or AcquireInstanceFromDir)", in.Instance.Metadata.Name)
	}
	if in.Platform == nil {
		return none, nil, errors.New("RenderInput.Platform is required")
	}
	if in.Platform.Source == nil {
		return none, nil, fmt.Errorf("platform %q carries no Source: the render build imports the platform as a package (acquire it with AcquirePlatformFromDir)", platformName(in.Platform))
	}
	if in.RuntimeName == "" {
		return none, nil, errors.New("RenderInput.RuntimeName must be non-empty")
	}
	if in.Skew != SkewWarn && in.Skew != SkewRefuse {
		return none, nil, fmt.Errorf("RenderInput.Skew %d is not a SkewPolicy", in.Skew)
	}
	if err := ctx.Err(); err != nil {
		return none, nil, err
	}
	// The core floor: the render build evaluates the platform module's own
	// core pin, the one its Package was built from, so a Package lacking
	// providedBy is a build whose glue would lack it. The platform recorded
	// the floor at construction; Render reads that fact, not Package.
	if err := in.Platform.CoreFloor(); err != nil {
		return none, nil, fmt.Errorf("render refused before staging: %w", err)
	}

	staged, err := renderstage.Stage(in.Instance.Source, in.Platform.Source, in.RuntimeName, renderstage.StageOptions{
		LocalReplacements: in.LocalReplacements,
		SkipUnprovided:    in.SkipUnprovided,
	})
	if err != nil {
		return none, nil, fmt.Errorf("staging render module: %w", err)
	}

	rows := make([]ResolvedVersion, 0, len(staged.Skew))
	var refusals []error
	for _, r := range staged.Skew {
		rows = append(rows, ResolvedVersion(r))
		if r.Newer && in.Skew == SkewRefuse {
			refusals = append(refusals, &oerrors.SkewError{Path: r.Path, ModuleVersion: r.ModuleVersion, PlatformVersion: r.PlatformVersion})
		}
	}
	if len(refusals) > 0 {
		return none, nil, fmt.Errorf("render refused before evaluation: %w", errors.Join(refusals...))
	}
	if err := ctx.Err(); err != nil {
		return none, nil, err
	}

	// One build, one context, dropped with the render (0019:D8).
	built, err := renderstage.Build(cuecontext.New(), staged, cueenv.Override(k.registry, ""))
	if err != nil {
		return none, nil, fmt.Errorf("building render module: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return none, nil, err
	}

	var replacements []Replacement
	for _, r := range staged.Replacements {
		replacements = append(replacements, Replacement(r))
	}
	diag, err := decodeRenderDiagnostics(built, rows, replacements)
	if err != nil {
		return built, nil, err
	}
	if gate := gateErrors(diag); gate != nil {
		diag.FailedPairs = failedPairs(built, diag.Pairs)
		return built, nil, &RenderError{Diagnostics: diag, Err: gate}
	}
	compiled, failed, err := decodeRendered(built, diag, in.Instance.Metadata.Name)
	if err != nil {
		diag.FailedPairs = failed
		return built, nil, &RenderError{Diagnostics: diag, Err: err}
	}

	return built, &RenderResult{Compiled: compiled, Diagnostics: diag}, nil
}

func platformName(p *platform.Platform) string {
	if p != nil && p.Metadata != nil {
		return p.Metadata.Name
	}
	return "<unnamed>"
}
