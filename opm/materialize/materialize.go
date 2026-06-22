package materialize

import (
	"context"
	"fmt"
	"strings"

	"cuelang.org/go/cue"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

// Materialize realizes a #Platform's path-keyed catalog subscriptions into a
// sealed [MaterializedPlatform]. It walks p's #registry; for each enabled
// subscription it enumerates published versions, narrows them by the
// subscription filter (range ∧ allow ∧ deny), pulls each survivor against the
// supplied registry, and indexes the selected catalogs' #transformers into a
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
// first failing subscription (design.md Q3).
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
		filter := decodeFilter(subVal)

		published, err := enumerateVersions(ctx, env, sub)
		if err != nil {
			return nil, &oerrors.MaterializeError{Kind: oerrors.MaterializeKindCatalog, Subscription: sub, Cause: err}
		}
		if len(published) == 0 {
			return nil, &oerrors.MaterializeError{
				Kind: oerrors.MaterializeKindCatalog, Subscription: sub,
				Cause: fmt.Errorf("no published versions for subscription path"),
			}
		}

		survivors, err := filterVersions(published, filter)
		if err != nil {
			return nil, &oerrors.MaterializeError{Kind: oerrors.MaterializeKindCatalog, Subscription: sub, Cause: err}
		}
		if len(survivors) == 0 {
			return nil, &oerrors.MaterializeError{
				Kind: oerrors.MaterializeKindCatalog, Subscription: sub,
				Cause: fmt.Errorf("filter selected no versions from %v", published),
			}
		}

		for _, ver := range survivors {
			bare := strings.TrimPrefix(ver, "v")
			cv, err := pullCatalog(octx, env, sub, ver)
			if err != nil {
				return nil, &oerrors.MaterializeError{Kind: oerrors.MaterializeKindCatalog, Subscription: sub, Version: bare, Cause: err}
			}
			builds = append(builds, catalogBuild{Subscription: sub, Version: bare, Value: cv})
		}
		// Highest survivor is the resolved version recorded for diagnostics.
		resolved[sub] = strings.TrimPrefix(survivors[len(survivors)-1], "v")
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

// decodeFilter projects a #Subscription's optional filter into a
// *subscriptionFilter, or nil when no filter is authored.
func decodeFilter(sub cue.Value) *subscriptionFilter {
	fv := sub.LookupPath(cue.ParsePath("filter"))
	if !fv.Exists() {
		return nil
	}
	f := &subscriptionFilter{}
	if r := fv.LookupPath(cue.ParsePath("range")); r.Exists() {
		if s, err := r.String(); err == nil {
			f.Range = s
		}
	}
	if a := fv.LookupPath(cue.ParsePath("allow")); a.Exists() {
		_ = a.Decode(&f.Allow)
	}
	if d := fv.LookupPath(cue.ParsePath("deny")); d.Exists() {
		_ = d.Decode(&f.Deny)
	}
	return f
}
