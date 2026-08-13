package synth

import (
	"errors"
	"fmt"
	"strings"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/schema"
)

// PlatformInput is the typed input carried into Platform. Required fields:
// Name, Type, SchemaCache. Optional fields are rendered into the platform
// only when present (non-nil / non-empty); empty values do not displace
// schema-derived fields. It is the #Platform peer of [InstanceInput].
type PlatformInput struct {
	// Name is the platform name (metadata.name). Required. Must satisfy the
	// schema's #NameType regex; violations surface as a CUE unification error
	// from Platform.
	Name string

	// Type is the platform type discriminator (the top-level #Platform.type
	// field). Required by the schema. Today it is an informational
	// discriminator the matcher does not consult.
	Type string

	// SchemaCache supplies the OPM core schema used to unify against
	// #Platform. REQUIRED. Typically the value of kernel.SchemaCache() from
	// the caller's Kernel — passing the kernel's cache preserves the
	// one-Cache-per-process invariant and avoids a duplicate schema fetch.
	// Platform returns an error when this field is nil.
	SchemaCache *schema.Cache

	// Description, Labels, and Annotations layer onto metadata. Each is
	// rendered only when non-empty.
	Description string
	Labels      map[string]string
	Annotations map[string]string

	// Subscriptions is the optional typed registry of catalog subscriptions.
	// The map key is the catalog's CUE module path (e.g.
	// "opmodel.dev/catalogs/opm"); a key that violates #ModulePathType
	// surfaces as a CUE unification error from Platform. An empty/nil map
	// leaves #registry empty.
	Subscriptions map[string]SubscriptionSpec
}

// SubscriptionSpec is the typed form of a single #registry subscription.
type SubscriptionSpec struct {
	// Enable maps onto #Subscription.enable. It is a pointer so an omitted
	// value (nil) DEFERS to the schema's `*true` default rather than forcing
	// `false`: nil → schema default, non-nil → the explicit bool. This mirrors
	// the unset-vs-supplied distinction synth.Instance draws for its inputs.
	Enable *bool

	// Version maps onto #Subscription.version — the single build the
	// subscription materializes (0010 D14: the platform file IS the
	// resolution). REQUIRED: Platform returns an error naming the
	// subscription path when it is empty, because the kernel reads this
	// scalar — a versionless subscription would fail at materialize.
	Version string
}

// Platform sentinel errors. These mirror the Instance set but carry
// synth.Platform: wording so error messages name the failing artifact. They
// are distinct package-level vars from the Instance sentinels (D4) — the
// instance sentinels stay untouched to avoid churn on synth.Instance callers.
var (
	// ErrMissingType is returned when PlatformInput.Type is empty. It has no
	// Instance counterpart (only #Platform carries a required type).
	ErrMissingType = errors.New("synth.Platform: Type is required")

	// ErrPlatformMissingName is returned when PlatformInput.Name is empty.
	ErrPlatformMissingName = errors.New("synth.Platform: Name is required")

	// ErrPlatformMissingSchemaCache is returned when PlatformInput.SchemaCache
	// is nil. Callers typically pass kernel.SchemaCache() from their Kernel.
	ErrPlatformMissingSchemaCache = errors.New("synth.Platform: SchemaCache is required")

	// ErrPlatformSchemaUnavailable is returned when the caller-supplied
	// SchemaCache resolves but does not expose #Platform.
	ErrPlatformSchemaUnavailable = errors.New("synth.Platform: schema unavailable")

	// ErrSubscriptionMissingVersion is returned when a SubscriptionSpec
	// carries an empty Version. Early validation at the write side (the only
	// in-repo producer of the subscription shape) beats a materialize-time
	// failure at the read side: the kernel resolves exactly the authored
	// version (0010 D14), so a versionless subscription can never materialize.
	ErrSubscriptionMissingVersion = errors.New("synth.Platform: subscription Version is required")
)

// Platform builds a #Platform CUE value by unifying PlatformInput against the
// #Platform definition obtained from the caller-supplied SchemaCache. It is a
// peer of [Instance].
//
// Unlike Instance, Platform needs no userModule scope dance: #Platform has no
// nested closed-artifact input — all inputs are plain scalars, maps, and
// lists — so Platform renders a CUE source string and CompileStrings it with
// the resolved schema package as cue.Scope (to resolve #Platform /
// #Subscription references). See design D2.
//
// The returned cue.Value carries the caller-supplied identity and subscription
// fields and leaves the kernel-filled materialization slots
// (#composedTransformers, #matchers) unset — those are populated later by
// Materialize, not by synthesis.
func Platform(ctx *cue.Context, in PlatformInput) (cue.Value, error) {
	if in.Name == "" {
		return cue.Value{}, ErrPlatformMissingName
	}
	if in.Type == "" {
		return cue.Value{}, ErrMissingType
	}
	if in.SchemaCache == nil {
		return cue.Value{}, ErrPlatformMissingSchemaCache
	}
	for path, sub := range in.Subscriptions {
		if sub.Version == "" {
			return cue.Value{}, fmt.Errorf("%w: subscription %q", ErrSubscriptionMissingVersion, path)
		}
	}

	schemaPkg, err := in.SchemaCache.Get(ctx)
	if err != nil {
		return cue.Value{}, fmt.Errorf("synth.Platform: loading schema: %w", err)
	}
	if !schemaPkg.LookupPath(cue.ParsePath("#Platform")).Exists() {
		return cue.Value{}, fmt.Errorf("%w: #Platform not found in resolved schema", ErrPlatformSchemaUnavailable)
	}

	src := renderPlatformSource(in)
	root := ctx.CompileString(src, cue.Scope(schemaPkg), cue.Filename("synth/platform.cue"))
	if err := root.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("synth.Platform: compiling platform source: %w", err)
	}
	spec := root.LookupPath(cue.ParsePath("platform"))
	if err := spec.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("synth.Platform: unifying inputs with #Platform: %w", err)
	}
	return spec, nil
}

// renderPlatformSource emits the CUE source compiled by Platform. The source
// declares `platform: { #Platform, ... }` and references #Platform /
// #Subscription from cue.Scope.
func renderPlatformSource(in PlatformInput) string {
	var sb strings.Builder
	// Unify against #Platform with `&` rather than embedding it as a struct
	// field (`{ #Platform; ... }`). Embedding a closed definition into an open
	// struct literal relaxes its closedness, so an invalid #registry key
	// (one that violates #ModulePathType's pattern constraint) would be
	// silently accepted. `#Platform & {...}` preserves the closedness, so a
	// bad key surfaces as a "field not allowed" unification error — the
	// behavior the spec requires.
	sb.WriteString("platform: #Platform & {\n")
	sb.WriteString("\tmetadata: {\n")
	fmt.Fprintf(&sb, "\t\tname: %q\n", in.Name)
	if in.Description != "" {
		fmt.Fprintf(&sb, "\t\tdescription: %q\n", in.Description)
	}
	writeStringMap(&sb, "\t\t", "labels", in.Labels)
	writeStringMap(&sb, "\t\t", "annotations", in.Annotations)
	sb.WriteString("\t}\n")
	fmt.Fprintf(&sb, "\ttype: %q\n", in.Type)
	writeRegistry(&sb, in.Subscriptions)
	sb.WriteString("}\n")
	return sb.String()
}

// writeRegistry writes the "#registry: { ... }" block when subs is non-empty.
// Each subscription emits `enable` only when explicitly set (Enable != nil),
// letting the schema's `*true` default stand otherwise, and always emits the
// required `version` scalar (Platform refused an empty one before rendering).
func writeRegistry(sb *strings.Builder, subs map[string]SubscriptionSpec) {
	if len(subs) == 0 {
		return
	}
	sb.WriteString("\t#registry: {\n")
	for path, sub := range subs {
		fmt.Fprintf(sb, "\t\t%q: {\n", path)
		if sub.Enable != nil {
			fmt.Fprintf(sb, "\t\t\tenable: %t\n", *sub.Enable)
		}
		fmt.Fprintf(sb, "\t\t\tversion: %q\n", sub.Version)
		sb.WriteString("\t\t}\n")
	}
	sb.WriteString("\t}\n")
}
