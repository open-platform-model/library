// Package api defines the per-OPM-schema-version binding contract and a
// process-wide registry.
//
// A Binding owns every fact that varies across schema versions: the CUE paths
// the kernel reads or writes, the metadata decoders, the transformer-context
// shape, and the embedded schema filesystem. Per-version packages
// (pkg/api/v1alpha2, pkg/api/<vN>) implement Binding and self-register from
// init() via Register. Callers never construct a Binding directly; they look it
// up by version (Lookup) or by artifact (For).
//
// The registry is read-only after process start. Register panics on duplicate
// to surface misconfiguration at the earliest possible point — which in the
// init() chain means at the first import of a colliding package.
package api

import (
	"io/fs"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/pkg/apiversion"
)

// Paths is the inventory of every CUE path the kernel reads or writes when
// loading, matching, and rendering an OPM artifact for a given schema version.
//
// A binding returns a fully populated Paths from Binding.Paths(). All fields
// MUST be set; the renderer relies on every path being valid.
type Paths struct {
	// Artifact root.
	APIVersion cue.Path // "apiVersion"
	Metadata   cue.Path // "metadata"

	// Module release.
	Components     cue.Path // "components"
	Values         cue.Path // "values"
	Config         cue.Path // "#config"
	Module         cue.Path // "#module" — release's reference to its source #Module
	ModuleMetadata cue.Path // "#moduleMetadata" — release-side projection of #module.metadata

	// Module-internal field. DebugValues is a Module field — NOT a separate
	// kernel artifact. Frontends that want a debug overlay read the value at
	// this path from Module.Package and decide whether to layer it into the
	// values stack; the kernel never receives debugValues as a parameter.
	// See enhancement 001-kernel-redesign-around-platform D6.
	DebugValues cue.Path // "debugValues"

	// Provider.
	Transformers cue.Path // "#transformers"

	// Platform. The five Platform paths point at #registry and the four
	// CUE-computed views over it. Bindings expose them so kernel callers
	// (today: tests; slice 09: the matcher) can read the views via
	// Package.LookupPath without hand-spelling path literals.
	Registry             cue.Path // "#registry"
	KnownResources       cue.Path // "#knownResources"
	KnownTraits          cue.Path // "#knownTraits"
	ComposedTransformers cue.Path // "#composedTransformers"
	Matchers             cue.Path // "#matchers"
	MatchersResources    cue.Path // "#matchers.resources"
	MatchersTraits       cue.Path // "#matchers.traits"

	// Transformer body and matching predicates.
	Transform                    cue.Path // "#transform"
	TransformerRequiredLabels    cue.Path // "requiredLabels"
	TransformerRequiredResources cue.Path // "requiredResources"
	TransformerRequiredTraits    cue.Path // "requiredTraits"
	TransformerOptionalTraits    cue.Path // "optionalTraits"

	// Inside #transform.
	Component cue.Path // "#component"
	Context   cue.Path // "#context"
	Output    cue.Path // "output"

	// Sub-paths of #context filled per (release, component, transformer) pair.
	ContextModuleReleaseMetadata cue.Path // "#context.#moduleReleaseMetadata"
	ContextComponentMetadata     cue.Path // "#context.#componentMetadata"
	ContextRuntimeName           cue.Path // "#context.#runtimeName"

	// Component sub-paths.
	//
	// Blueprints are deliberately omitted: a Blueprint is a composition
	// template — its composedResources / composedTraits unify into a
	// Component's spec at CUE-evaluation time (see component.cue _allFields).
	// By the time the renderer sees a Component, blueprints are already
	// merged. Walking #blueprints separately would double-count.
	MetadataLabels      cue.Path // "metadata.labels"
	MetadataAnnotations cue.Path // "metadata.annotations"
	ComponentResources  cue.Path // "#resources"
	ComponentTraits     cue.Path // "#traits"
}

// ModuleMetadata is the canonical decoded module-level metadata shape returned
// by Binding.DecodeModuleMetadata. Bindings whose schema carries richer
// per-version data MAY keep the richer data internal; the value returned to
// kernel callers MUST conform to this shape.
type ModuleMetadata struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	ModulePath  string            `json:"modulePath"`
	FQN         string            `json:"fqn"`
	Version     string            `json:"version"`
	UUID        string            `json:"uuid"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ReleaseMetadata is the canonical decoded release-level metadata.
type ReleaseMetadata struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	UUID        string            `json:"uuid"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ProviderMetadata is the canonical decoded provider-level metadata.
type ProviderMetadata struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Version     string            `json:"version,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PlatformMetadata is the canonical decoded platform-level metadata returned
// by Binding.DecodePlatformMetadata. Type is the top-level #Platform.type
// field hoisted into the metadata projection so callers see one Go-level
// identity record per Platform artifact.
type PlatformMetadata struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ReleaseView is the read-only view of a module release that the binding needs
// to construct a transformer context. It exists so that BuildTransformerContext
// stays decoupled from pkg/module; any caller-supplied type that exposes these
// accessors can drive context construction (e.g. tests).
type ReleaseView interface {
	ReleaseName() string
	Namespace() string
	ReleaseUUID() string
	ModuleFQN() string
	ModuleVersion() string
	Labels() map[string]string
	Annotations() map[string]string
}

// Binding is the per-schema-version contract. Every field of Paths and every
// method MUST be implemented by a concrete binding. Bindings register
// themselves from init() — see package doc.
type Binding interface {
	// Version is the apiVersion literal this binding handles. It MUST equal the
	// key used to register the binding.
	Version() apiversion.Version

	// Paths returns the CUE path inventory. The returned struct is treated as
	// immutable by callers; bindings SHOULD return the same value on every call
	// (typically a package-level variable).
	Paths() Paths

	// DecodeModuleMetadata extracts ModuleMetadata from a #Module artifact
	// value. The input v is the artifact root (the value whose apiVersion field
	// matches Version()).
	DecodeModuleMetadata(v cue.Value) (*ModuleMetadata, error)

	// DecodeReleaseMetadata extracts ReleaseMetadata from a #ModuleRelease
	// artifact value. The input v is the artifact root.
	DecodeReleaseMetadata(v cue.Value) (*ReleaseMetadata, error)

	// DecodeProviderMetadata extracts ProviderMetadata from a #Provider
	// artifact value. fallbackName is used when the artifact's metadata.name
	// field is absent — typically the config map key under which the provider
	// was loaded.
	DecodeProviderMetadata(v cue.Value, fallbackName string) (*ProviderMetadata, error)

	// DecodePlatformMetadata extracts PlatformMetadata from a #Platform
	// artifact value. The input v is the artifact root (the value whose
	// apiVersion field matches Version()). The returned struct merges
	// metadata.{name,description,labels,annotations} with the top-level
	// #Platform.type field.
	DecodePlatformMetadata(v cue.Value) (*PlatformMetadata, error)

	// BuildTransformerContext constructs the per-(release, component, transformer)
	// #context value. The returned cue.Value MUST be filled by the renderer at
	// Paths().Context. The renderer owns the surrounding plumbing (fill, error
	// propagation); the binding owns the shape of the context.
	//
	// schemaComp is the schema-preserving component value (the one with
	// definition fields such as #resources still present). runtimeName is
	// validated as non-empty by the public render entry point.
	//
	// Returned warnings are non-fatal annotations the renderer surfaces to the
	// caller (e.g. malformed metadata that decoded as empty).
	BuildTransformerContext(
		cueCtx *cue.Context,
		rel ReleaseView,
		compName string,
		schemaComp cue.Value,
		runtimeName string,
	) (cue.Value, []string, error)

	// EmbeddedSchema returns the read-only filesystem holding this version's
	// CUE schema source files. Used for offline/deterministic schema validation.
	// Returns nil when no embedded schema is wired (treated as a soft failure
	// by callers that prefer an embed but can fall back to registry resolution).
	EmbeddedSchema() fs.FS
}
