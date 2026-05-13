package synth

import (
	"errors"
	"fmt"
	"strings"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/api"
	"github.com/open-platform-model/library/opm/module"
)

// ReleaseInput is the typed input carried into Release. Required fields:
// Module, Name, Namespace. Optional fields are filled into the release only
// when present (non-nil / non-empty / non-zero); empty values do not displace
// schema-derived fields.
type ReleaseInput struct {
	// Module is the source #Module the release deploys. Required.
	Module *module.Module

	// Name is the release name (metadata.name). Required. Must satisfy the
	// schema's #NameType regex; violations surface as a CUE unification error
	// from Release.
	Name string

	// Namespace is the target namespace (metadata.namespace). Required.
	Namespace string

	// Values is the caller-supplied configuration value unified against the
	// module's #config. The zero cue.Value signals "no values supplied" — the
	// schema's values path is left unfilled and concreteness is enforced
	// downstream by Kernel.ProcessModuleRelease. Release NEVER falls back to
	// Module.debugValues; that is a frontend policy concern.
	Values cue.Value

	// Labels and Annotations layer over the schema's stamped
	// module-release.opmodel.dev/{name,uuid} labels. CUE unification merges
	// caller-supplied entries with schema-stamped ones; caller-supplied keys
	// MUST NOT collide with the schema's reserved keys.
	Labels      map[string]string
	Annotations map[string]string
}

// Sentinel errors. Frontends inspecting the failure surface match on these via
// errors.Is. Synthesis-time errors all wrap one of these so callers can
// distinguish "you forgot a required field" from "the binding/schema is
// broken."
var (
	// ErrMissingModule is returned when ReleaseInput.Module is nil.
	ErrMissingModule = errors.New("synth.Release: Module is required")

	// ErrMissingName is returned when ReleaseInput.Name is empty.
	ErrMissingName = errors.New("synth.Release: Name is required")

	// ErrMissingNamespace is returned when ReleaseInput.Namespace is empty.
	ErrMissingNamespace = errors.New("synth.Release: Namespace is required")

	// ErrSchemaUnavailable is returned when the version binding cannot supply
	// its schema package or the package does not expose #ModuleRelease.
	ErrSchemaUnavailable = errors.New("synth.Release: schema unavailable")
)

// Release builds a #ModuleRelease CUE value by unifying ReleaseInput against
// the schema definition obtained via the version binding's SchemaValue.
//
// The function does NOT validate values against #config and does NOT enforce
// concreteness. Both responsibilities live downstream in
// Kernel.ProcessModuleRelease, which the Kernel.SynthesizeRelease wrapper
// chains onto this call. See package doc for the recommended entry point.
//
// The returned cue.Value carries every schema-derived field automatically:
// metadata.uuid is computed by uuid.SHA1, components is fanned from the
// unified module, opm-secrets is added when #Secret instances are present,
// and the standard module-release.opmodel.dev/{name,uuid} labels are stamped.
// Release stamps only the caller-supplied fields and lets CUE derive the rest.
func Release(ctx *cue.Context, in ReleaseInput) (cue.Value, error) {
	if in.Module == nil {
		return cue.Value{}, ErrMissingModule
	}
	if in.Name == "" {
		return cue.Value{}, ErrMissingName
	}
	if in.Namespace == "" {
		return cue.Value{}, ErrMissingNamespace
	}

	binding, err := api.Lookup(in.Module.APIVersion)
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Release: resolving binding for %q: %w", in.Module.APIVersion, err)
	}

	schemaPkg, err := binding.SchemaValue(ctx)
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Release: loading schema: %w", err)
	}
	if !schemaPkg.LookupPath(cue.ParsePath("#ModuleRelease")).Exists() {
		return cue.Value{}, fmt.Errorf("%w: #ModuleRelease not found in binding %q schema", ErrSchemaUnavailable, in.Module.APIVersion)
	}

	// CUE's Go API rejects FillPath into a closed definition when the
	// definition uses self-referential constraints (apis/core/v1alpha2/module.cue:15
	// declares `modulePath: metadata.modulePath`). Filling a separately-built
	// module value into #ModuleRelease.#module triggers admission checks
	// against a re-evaluated copy of #Module where the self-reference
	// resolves to bottom, so caller-supplied modulePath / version are
	// rejected as "field not allowed".
	//
	// Workaround: build a scope value that extends the schema package with a
	// non-hidden `userModule` field, fill the caller's module into that field,
	// then compile a release expression that references both `#ModuleRelease`
	// and `userModule` via cue.Scope. The user's module enters the
	// compilation as a value (not a re-emitted source fragment), so its full
	// type-embedding chain survives — components, traits, blueprints all
	// retain the schema relationships needed for the schema's `components`
	// comprehension and the autosecrets discovery walk.
	//
	// Caller-supplied Values are pre-merged into the module's #config BEFORE
	// the module enters the scope. The release schema unifies values into
	// #config via `let unifiedModule = #module & {#config: values}`, and the
	// resulting #Image / #Secret closures propagate back to release.values
	// through CUE's bidirectional unification. That propagation conflicts
	// with the defaults that ProcessModuleRelease later folds into the values
	// path: closed res.#Image types declared in the user's registry-loaded
	// module differ at the CUE-runtime level from the same types reached
	// through the embedded schema, so admission of fields like image.pullPolicy
	// fails even though both sides declare them. Pre-merging in Go sidesteps
	// the conflict — the schema sees a fully-formed #config without needing
	// to project values back through the closure boundary.
	preparedModule := in.Module.Package
	if in.Values.Exists() {
		preparedModule = preparedModule.FillPath(binding.Paths().Config, in.Values)
		if err := preparedModule.Err(); err != nil {
			return cue.Value{}, fmt.Errorf("synth.Release: merging values into module #config: %w", err)
		}
	}

	scope, err := buildReleaseScope(ctx, schemaPkg, preparedModule)
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Release: %w", err)
	}

	src := renderReleaseSource(in)
	root := ctx.CompileString(src, cue.Scope(scope), cue.Filename("synth/release.cue"))
	if err := root.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("synth.Release: compiling release source: %w", err)
	}
	spec := root.LookupPath(cue.ParsePath("release"))
	if err := spec.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("synth.Release: unifying inputs with #ModuleRelease: %w", err)
	}
	return spec, nil
}

// buildReleaseScope produces a cue.Value combining the binding's schema
// package with a `userModule` field bound to the supplied module. The
// returned value is used as cue.Scope when compiling the release source.
//
// The schema package is open at the file level, so unifying it with the
// overlay `{userModule: _}` introduces a new field without violating any
// closure. The combined value is then used as a *scope* for compilation,
// not a value to be filled — references resolve against it lexically.
func buildReleaseScope(ctx *cue.Context, schemaPkg, userModule cue.Value) (cue.Value, error) {
	overlay := ctx.CompileString("userModule: _")
	if err := overlay.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("building scope overlay: %w", err)
	}
	combined := schemaPkg.Unify(overlay)
	if err := combined.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("unifying schema with scope overlay: %w", err)
	}
	combined = combined.FillPath(cue.ParsePath("userModule"), userModule)
	if err := combined.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("binding module into scope: %w", err)
	}
	return combined, nil
}

// renderReleaseSource emits the CUE source compiled by Release. The source
// declares `release: { #ModuleRelease, ... }` and references `userModule`
// from cue.Scope.
func renderReleaseSource(in ReleaseInput) string {
	var sb strings.Builder
	sb.WriteString("release: {\n")
	sb.WriteString("\t#ModuleRelease\n")
	sb.WriteString("\tmetadata: {\n")
	fmt.Fprintf(&sb, "\t\tname:      %q\n", in.Name)
	fmt.Fprintf(&sb, "\t\tnamespace: %q\n", in.Namespace)
	writeStringMap(&sb, "\t\t", "labels", in.Labels)
	writeStringMap(&sb, "\t\t", "annotations", in.Annotations)
	sb.WriteString("\t}\n")
	sb.WriteString("\t#module: userModule\n")
	// values is left to ProcessModuleRelease to fill — synth pre-merges values
	// into userModule's #config so the schema's components fan-out sees them,
	// but the release-level values path stays open so downstream concreteness
	// validation can run without colliding with res.#Image / res.#Secret
	// closures inherited through `#config: values`.
	sb.WriteString("}\n")
	return sb.String()
}

// writeStringMap writes a "<field>: { ... }" block when m is non-empty.
func writeStringMap(sb *strings.Builder, indent, field string, m map[string]string) {
	if len(m) == 0 {
		return
	}
	fmt.Fprintf(sb, "%s%s: {\n", indent, field)
	for k, v := range m {
		fmt.Fprintf(sb, "%s\t%q: %q\n", indent, k, v)
	}
	fmt.Fprintf(sb, "%s}\n", indent)
}
