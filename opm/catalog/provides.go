package catalog

import (
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/internal/corepath"
	"github.com/open-platform-model/library/opm/internal/modversion"
	"github.com/open-platform-model/library/opm/schema"
)

// Provides returns the provider-fulfilled contracts this catalog implements:
// every contract required by one of the catalog's own #transformers
// ([corepath.Transformers]) whose value carries `fulfilment: "provider"`.
// Required demands only — `optionalResources` and `optionalTraits` are
// tolerance, not fulfilment, the same rule core applies when it folds a
// platform's inventory.
//
// Core derives that set itself since [schema.ProvidesSince], as the
// catalog's `provides` field ([schema.CatalogProvides]), and Provides reads
// it from there. A catalog whose committed cue.mod pins an older core is
// answered by a deprecated fallback that walks #transformers in Go, the way
// every release before this one did, even when the catalog authors a
// `provides` beside its embedded #Catalog: an older core does not derive
// that field, so an authored one is not the catalog's derived set. When the
// catalog carries no Source (one built by [NewCatalogFromValue]), its core is
// unknown, and the field is read when present and the fallback runs when it
// is absent.
//
// Both paths read the demand entry's own value rather than looking the
// contract up in this catalog's #resources / #traits: a provider catalog
// implements contracts ANOTHER catalog defines (that is what makes it a
// provider), so its own member maps do not list them.
//
// Nothing decodes this at construction and no kernel verb calls it: the value
// is already built, and it is read only when a caller asks, so acquiring a
// catalog pays nothing for it.
//
// Reports, never refusals. The result is deterministically ordered and
// deduplicated, so two derivations of one catalog compare equal element for
// element without the caller sorting — the comparison a consumer makes
// against a claimed list is then a plain slice equality. A catalog that
// implements no provider-fulfilled contract, or ships no transformers at
// all, returns a non-nil empty slice and a nil error: implementing none is a
// fact about the catalog, not a malformed value.
//
// A provider set that cannot be read is an error, with no partial set, on
// either path: a #transformers that does not evaluate; a `provides` that is
// not a concrete list of strings; a demand's `fulfilment` present but not a
// concrete string (the catalog's contracts were built against a core that
// means something else by the field); and, for a catalog carrying a Source,
// a committed module file that cannot be read or names an invalid core
// version. An entry carrying no `fulfilment` at all is simply not
// provider-fulfilled.
func (c *Catalog) Provides() ([]string, error) {
	if c == nil {
		return nil, fmt.Errorf("catalog is nil")
	}
	// ADR-012: the per-catalog provider set is the second derived rule moved
	// into core (core v2.0.0-beta.3), after #Platform.#contracts.providedBy.
	// The fold stays only for catalogs published against an older core.

	// Shared by both paths: core guards each demand map with `!= _|_`, so a
	// #transformers that does not evaluate could drop out of `provides`
	// where the fold would report it.
	if transformers := c.Package.LookupPath(corepath.Transformers); transformers.Exists() {
		if err := transformers.Err(); err != nil {
			return nil, fmt.Errorf("catalog %s did not evaluate: %w", corepath.Transformers, err)
		}
	}

	old, err := c.pinsCoreBeforeProvides()
	if err != nil {
		return nil, err
	}
	field := c.Package.LookupPath(schema.CatalogProvides)
	if old || !field.Exists() {
		return c.providesFold()
	}

	var fqns []string
	if err := field.Decode(&fqns); err != nil {
		return nil, fmt.Errorf("reading catalog %s: %w", schema.CatalogProvides, err)
	}
	return sortedUnique(fqns), nil
}

// pinsCoreBeforeProvides reports whether the catalog's committed cue.mod pins
// a core older than [schema.ProvidesSince]. It reports false, with no error,
// when the core is unknown: no Source, no core requirement, or a requirement
// carrying no version (a local replacement). A module file Requires refuses,
// or a core version that is not SemVer, is an error: acquisition already
// read that file, so failing to read it now is a defect, not an unknown.
func (c *Catalog) pinsCoreBeforeProvides() (bool, error) {
	if c.Source == nil || c.Source.Root == "" {
		return false, nil
	}
	reqs, err := c.Requires()
	if err != nil {
		return false, fmt.Errorf("reading the catalog's core pin: %w", err)
	}
	version := reqs[modversion.CorePath]
	if version == "" {
		return false, nil
	}
	// Defensive: Requires reads the file through the module-file parser,
	// which already refuses a version that is not canonical SemVer.
	cmp, err := modversion.Compare(version, schema.ProvidesSince)
	if err != nil {
		return false, fmt.Errorf("reading the catalog's core pin %s: %w", modversion.CorePath, err)
	}
	return cmp < 0, nil
}

// providesFold derives the provider-fulfilled set by walking #transformers
// in Go.
//
// Deprecated: providesFold is the fallback for catalogs built against a core
// older than [schema.ProvidesSince], which carry no derived `provides`. It is
// removed before GA, after catalog_opm is republished against a core that
// derives `provides`. [Catalog.Provides] is the entry point.
func (c *Catalog) providesFold() ([]string, error) {
	found := map[string]struct{}{}

	transformers := c.Package.LookupPath(corepath.Transformers)
	if !transformers.Exists() {
		return []string{}, nil
	}
	if err := transformers.Err(); err != nil {
		return nil, fmt.Errorf("catalog %s did not evaluate: %w", corepath.Transformers, err)
	}

	iter, err := transformers.Fields()
	if err != nil {
		return nil, fmt.Errorf("reading catalog %s: %w", corepath.Transformers, err)
	}
	for iter.Next() {
		impl := iter.Selector().Unquoted()
		for _, demands := range []cue.Path{corepath.RequiredResources, corepath.RequiredTraits} {
			if err := collectProviders(iter.Value(), impl, demands, found); err != nil {
				return nil, err
			}
		}
	}

	out := make([]string, 0, len(found))
	for fqn := range found {
		out = append(out, fqn)
	}
	sort.Strings(out)
	return out, nil
}

// sortedUnique returns fqns sorted and deduplicated, as a non-nil slice. On
// core's own output it is the identity; it keeps the ordering promise for a
// value core did not build.
func sortedUnique(fqns []string) []string {
	out := make([]string, 0, len(fqns))
	seen := make(map[string]struct{}, len(fqns))
	for _, fqn := range fqns {
		if _, dup := seen[fqn]; dup {
			continue
		}
		seen[fqn] = struct{}{}
		out = append(out, fqn)
	}
	sort.Strings(out)
	return out
}

// collectProviders adds every contract FQN the demand map at path requires
// whose value carries fulfilment "provider" into found. An absent demand map
// contributes nothing: both maps are optional on #ComponentTransformer.
func collectProviders(transformer cue.Value, impl string, path cue.Path, found map[string]struct{}) error {
	demands := transformer.LookupPath(path)
	if !demands.Exists() {
		return nil
	}
	iter, err := demands.Fields()
	if err != nil {
		return fmt.Errorf("reading %s.%s of transformer %q: %w", corepath.Transformers, path, impl, err)
	}
	for iter.Next() {
		fqn := iter.Selector().Unquoted()
		fulfilment := iter.Value().LookupPath(corepath.Fulfilment)
		if !fulfilment.Exists() {
			continue
		}
		s, err := fulfilment.String()
		if err != nil {
			return fmt.Errorf("reading %s of contract %q required by transformer %q: %w", corepath.Fulfilment, fqn, impl, err)
		}
		if s == "provider" {
			found[fqn] = struct{}{}
		}
	}
	return nil
}
