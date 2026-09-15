package catalog

import (
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/schema"
)

// Provides derives the provider-fulfilled contracts this catalog implements:
// every contract required by one of the catalog's own #transformers
// ([schema.Transformers]) whose value carries `fulfilment: "provider"`.
// Required demands only — `optionalResources` and `optionalTraits` are
// tolerance, not fulfilment, the same rule core applies when it folds a
// platform's inventory.
//
// The fold reads the demand entry's own value rather than looking the
// contract up in this catalog's #resources / #traits. That is not an
// optimization: a provider catalog implements contracts ANOTHER catalog
// defines (that is what makes it a provider), so its own member maps do not
// list them and a lookup there would derive the empty set for exactly the
// catalogs this method exists to describe.
//
// Nothing decodes this at construction and no kernel verb calls it: the value
// is already built, and it is read only when a caller asks, so acquiring a
// catalog pays nothing for it. That is platform.Platform.Contracts's rule,
// and so is the next one.
//
// Reports, never refusals. The result is deterministically ordered and
// deduplicated, so two derivations of one catalog compare equal element for
// element without the caller sorting — the comparison a consumer makes
// against a claimed list is then a plain slice equality. A catalog that
// implements no provider-fulfilled contract, or ships no transformers at
// all, returns a non-nil empty slice and a nil error: implementing none is a
// fact about the catalog, not a malformed value.
//
// The one error is a catalog whose demand entries do not evaluate: a
// `fulfilment` present but not readable as a concrete string means the
// catalog's contracts were built against a core that means something else by
// the field, and that is reported rather than silently folded away. An entry
// carrying no `fulfilment` at all is simply not provider-fulfilled.
func (c *Catalog) Provides() ([]string, error) {
	if c == nil {
		return nil, fmt.Errorf("catalog is nil")
	}

	found := map[string]struct{}{}

	transformers := c.Package.LookupPath(schema.Transformers)
	if !transformers.Exists() {
		return []string{}, nil
	}
	if err := transformers.Err(); err != nil {
		return nil, fmt.Errorf("catalog %s did not evaluate: %w", schema.Transformers, err)
	}

	iter, err := transformers.Fields()
	if err != nil {
		return nil, fmt.Errorf("reading catalog %s: %w", schema.Transformers, err)
	}
	for iter.Next() {
		impl := iter.Selector().Unquoted()
		for _, demands := range []cue.Path{schema.RequiredResources, schema.RequiredTraits} {
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
		return fmt.Errorf("reading %s.%s of transformer %q: %w", schema.Transformers, path, impl, err)
	}
	for iter.Next() {
		fqn := iter.Selector().Unquoted()
		fulfilment := iter.Value().LookupPath(schema.Fulfilment)
		if !fulfilment.Exists() {
			continue
		}
		s, err := fulfilment.String()
		if err != nil {
			return fmt.Errorf("reading %s of contract %q required by transformer %q: %w", schema.Fulfilment, fqn, impl, err)
		}
		if s == "provider" {
			found[fqn] = struct{}{}
		}
	}
	return nil
}
