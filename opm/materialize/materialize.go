package materialize

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

// Materialize realizes a #Platform's path-keyed catalog subscriptions into a
// sealed [MaterializedPlatform]. It walks p's #registry; for each enabled
// subscription it reads the authored `version!` scalar (0010 D14: the platform
// file IS the resolution), checks the named build sits in the subscription
// key's major, pulls exactly that build against the supplied registry,
// verifies the pulled catalog's declared identity against the subscription
// coordinate (D11/D9), and indexes the catalogs' #transformers into a
// composed transformer map plus a #matchers reverse index. Both are exposed as
// native first-class fields ([MaterializedPlatform.Transformers] /
// [MaterializedPlatform.Matchers]) — they are NOT filled onto the closed
// c.#Platform (ADR-003: doing so corrupts output-local hidden fields).
//
// owner supplies the *cue.Context used for both the platform value and every
// catalog build (the native surfaces share one context with the platform).
// registry is the CUE_REGISTRY mapping for catalog (and schema) resolution; an
// empty string inherits the process CUE_REGISTRY. The process environment is
// never mutated.
//
// Inputs are not mutated: p.Package is read-only and never filled.
// Failures surface as [oerrors.MaterializeError] (Kind "catalog") naming the
// offending subscription path and version. Materialize fails fast on the
// first failing subscription (design.md Q3). An identity mismatch carries an
// [oerrors.IdentityError] as Cause, reachable via errors.As.
func Materialize(ctx context.Context, owner CueContextOwner, registry string, p *platform.Platform) (*MaterializedPlatform, error) {
	if owner == nil || owner.CueContext() == nil {
		return nil, fmt.Errorf("materialize: nil cue.Context owner")
	}
	if p == nil {
		return nil, fmt.Errorf("materialize: nil *platform.Platform")
	}
	octx := owner.CueContext()
	env := resolverEnv(registry)

	registryVal := p.Package.LookupPath(schema.Registry)
	if !registryVal.Exists() {
		return nil, fmt.Errorf("materialize: platform has no #registry")
	}
	it, err := registryVal.Fields()
	if err != nil {
		return nil, fmt.Errorf("materialize: reading #registry: %w", err)
	}

	var builds []catalogBuild
	resolved := map[string]string{}
	for it.Next() {
		sub := it.Selector().Unquoted()
		subVal := it.Value()

		if !subscriptionEnabled(subVal) {
			continue
		}

		ver, err := subscriptionVersion(subVal)
		if err != nil {
			return nil, &oerrors.MaterializeError{Kind: oerrors.MaterializeKindCatalog, Subscription: sub, Cause: err}
		}
		if err := majorAgrees(sub, ver); err != nil {
			return nil, &oerrors.MaterializeError{Kind: oerrors.MaterializeKindCatalog, Subscription: sub, Version: ver, Cause: err}
		}

		cv, err := pullCatalog(octx, env, sub, "v"+ver)
		if err != nil {
			return nil, &oerrors.MaterializeError{
				Kind: oerrors.MaterializeKindCatalog, Subscription: sub, Version: ver,
				Cause: pullFailureDiagnostic(ctx, env, sub, ver, err),
			}
		}
		if err := verifyCatalogIdentity(cv, sub, ver); err != nil {
			return nil, &oerrors.MaterializeError{Kind: oerrors.MaterializeKindCatalog, Subscription: sub, Version: ver, Cause: err}
		}

		builds = append(builds, catalogBuild{Subscription: sub, Version: ver, Value: cv})
		resolved[sub] = ver
	}

	composed, matchers, err := indexCatalogs(octx, builds)
	if err != nil {
		return nil, err
	}

	// Federate the native surfaces (ADR-003): expose the open composed map and
	// reverse index as first-class fields and do NOT FillPath them onto the
	// closed c.#Platform. The closed twin is never built, so there is no surface
	// from which reading a #transform corrupts output-local hidden fields. The
	// original spec stays reachable as Source.Package for #registry/metadata.
	return &MaterializedPlatform{Source: p, Transformers: composed, Matchers: matchers, Resolved: resolved}, nil
}

// subscriptionEnabled reports whether a #Subscription value is enabled. The
// schema defaults enable to true (bool | *true); a concrete false skips the
// subscription. A missing or non-concrete enable field is treated as enabled.
func subscriptionEnabled(sub cue.Value) bool {
	en := sub.LookupPath(cue.ParsePath("enable"))
	if !en.Exists() {
		return true
	}
	b, err := en.Bool()
	if err != nil {
		return true
	}
	return b
}

// subscriptionVersion reads a #Subscription's required `version!` scalar —
// the single build the subscription materializes (D14). Core v2's schema
// requires it; an absent or non-concrete version is an authoring error.
func subscriptionVersion(sub cue.Value) (string, error) {
	v := sub.LookupPath(cue.ParsePath("version"))
	if !v.Exists() {
		return "", fmt.Errorf("subscription has no version (core v2's #Subscription requires version!)")
	}
	s, err := v.String()
	if err != nil {
		return "", fmt.Errorf("subscription version is not a concrete string: %w", err)
	}
	return s, nil
}

// majorAgrees checks the named build sits in the subscription key's major —
// the one rule D14 leaves in resolution, mirroring the target schema's
// _majorAgrees (majors compared as strings, no numeric ordering). key is the
// subscription's #ModulePathType key; version is the authored bare SemVer. A
// key with no @vN suffix is an error: core v2's #ModulePathType requires the
// suffix, so this is defensive, not a supported path.
func majorAgrees(key, version string) error {
	_, major, ok := ast.SplitPackageVersion(key)
	if !ok {
		return fmt.Errorf("subscription key %q carries no @vN major suffix", key)
	}
	want := strings.TrimPrefix(major, "v")
	if got, _, _ := strings.Cut(version, "."); got != want {
		return fmt.Errorf("authored version %q is outside the subscription key's major %q", version, major)
	}
	return nil
}

// verifyCatalogIdentity compares the pulled catalog's declared identity
// against the coordinate it was fetched by (D11; version clause D9):
// metadata.modulePath must equal the subscription key (both @vN-suffixed —
// direct string comparison, no recomposition) and metadata.version must equal
// the version just pulled. Partially redundant with majorAgrees by design —
// that check validates the author's consistency before any I/O; this one
// validates the artifact's honesty after. A mismatch returns an
// [oerrors.IdentityError]; the caller wraps it in a *MaterializeError.
func verifyCatalogIdentity(cv cue.Value, sub, ver string) error {
	coordinate := sub + " v" + ver

	declaredPath, err := cv.LookupPath(cue.ParsePath("metadata.modulePath")).String()
	if err != nil {
		return fmt.Errorf("reading catalog metadata.modulePath of %s: %w", coordinate, err)
	}
	if declaredPath != sub {
		return oerrors.IdentityError{
			Artifact:   "catalog",
			Field:      "path",
			Declared:   declaredPath,
			Fetched:    sub,
			Coordinate: coordinate,
		}
	}

	declaredVersion, err := cv.LookupPath(cue.ParsePath("metadata.version")).String()
	if err != nil {
		return fmt.Errorf("reading catalog metadata.version of %s: %w", coordinate, err)
	}
	if declaredVersion != ver {
		return oerrors.IdentityError{
			Artifact:   "catalog",
			Field:      "version",
			Declared:   declaredVersion,
			Fetched:    ver,
			Coordinate: coordinate,
		}
	}

	return nil
}

// pullFailureDiagnostic enriches a failed pull of a named build with what IS
// published (D14 keeps `published` as a diagnostic surface; the happy path
// makes no enumeration round-trip). Enumeration runs only here — lazily, on
// failure. The published list is scoped to the subscription key's major
// (enumerateVersions' documented scoping). When enumeration itself fails, or
// the named build IS in the list (the pull failed for another reason), the
// original pull error stands alone.
func pullFailureDiagnostic(ctx context.Context, env []string, sub, ver string, pullErr error) error {
	published, enumErr := enumerateVersions(ctx, env, sub)
	if enumErr != nil {
		return pullErr
	}
	if !slices.Contains(published, "v"+ver) {
		return fmt.Errorf("named build \"v%s\" is not published; published in this major: %v: %w", ver, published, pullErr)
	}
	return pullErr
}
