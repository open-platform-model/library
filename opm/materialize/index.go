package materialize

import (
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/schema"
)

// optionalResources is the one transformer map path opm/schema does not yet
// export a constant for (it has Required{Resources,Traits} and
// OptionalTraits). The reverse index covers required ∪ optional primitives.
var optionalResources = cue.ParsePath("optionalResources")

// catalogBuild is one pulled catalog value tagged with the subscription path
// and resolved version it came from, for MaterializeError attribution.
type catalogBuild struct {
	Subscription string
	Version      string // bare SemVer
	Value        cue.Value
}

// indexCatalogs reads each catalog build's #transformers map and produces two
// CUE values built with octx: the composed transformer map (FQN →
// #ComponentTransformer) and the #matchers reverse index
// ({resources,traits}: primitive FQN → [transformers]).
//
// Transformers sharing an FQN across builds collapse via CUE unification when
// their bodies agree; divergent bodies surface as a MaterializeError wrapping
// the CUE conflict (spec: Transformer Indexing). Output ordering is stable
// (FQN-sorted) so repeated materializations are byte-identical.
func indexCatalogs(octx *cue.Context, builds []catalogBuild) (composed cue.Value, matchers cue.Value, err error) {
	// 1. Build the composed map, collapsing / conflicting on shared FQNs.
	composedByFQN := map[string]cue.Value{}
	for _, b := range builds {
		txs := b.Value.LookupPath(schema.Transformers)
		if !txs.Exists() {
			continue
		}
		it, ferr := txs.Fields()
		if ferr != nil {
			return cue.Value{}, cue.Value{}, &oerrors.MaterializeError{
				Kind: oerrors.MaterializeKindCatalog, Subscription: b.Subscription, Version: b.Version,
				Cause: fmt.Errorf("reading #transformers: %w", ferr),
			}
		}
		for it.Next() {
			fqn := it.Selector().Unquoted()
			tx := it.Value()
			existing, seen := composedByFQN[fqn]
			if !seen {
				composedByFQN[fqn] = tx
				continue
			}
			// Same FQN seen in another build: unify and keep on agreement
			// (the spec's "identical builds collapse"), conflict on divergence.
			// Through the real #Catalog shape this collapse path is largely
			// defensive — the #Catalog pattern stamps each transformer's
			// metadata.modulePath to "<catalogPath>/transformers", so two
			// builds sharing an FQN key necessarily come from different paths
			// and diverge on modulePath; the same path cannot yield two builds
			// at one FQN (distinct versions → distinct FQNs). It is exercised
			// directly by indexCatalogs unit tests with synthetic builds.
			unified := existing.Unify(tx)
			if vErr := unified.Validate(cue.Concrete(false)); vErr != nil {
				return cue.Value{}, cue.Value{}, &oerrors.MaterializeError{
					Kind: oerrors.MaterializeKindCatalog, Subscription: b.Subscription, Version: b.Version,
					Cause: fmt.Errorf("transformer %q diverges across selected builds: %w", fqn, vErr),
				}
			}
			composedByFQN[fqn] = unified
		}
	}

	// 2. Build the reverse index from the deduped composed map: each
	// transformer's required ∪ optional primitive FQNs map back to it.
	resources := map[string][]cue.Value{}
	traits := map[string][]cue.Value{}
	for _, fqn := range sortedKeys(composedByFQN) {
		tx := composedByFQN[fqn]
		for _, rfqn := range mapKeys(tx, schema.TransformerRequiredResources) {
			resources[rfqn] = append(resources[rfqn], tx)
		}
		for _, rfqn := range mapKeys(tx, optionalResources) {
			resources[rfqn] = append(resources[rfqn], tx)
		}
		for _, tfqn := range mapKeys(tx, schema.TransformerRequiredTraits) {
			traits[tfqn] = append(traits[tfqn], tx)
		}
		for _, tfqn := range mapKeys(tx, schema.TransformerOptionalTraits) {
			traits[tfqn] = append(traits[tfqn], tx)
		}
	}

	// 3. Emit CUE values.
	composed = octx.CompileString("{}")
	for _, fqn := range sortedKeys(composedByFQN) {
		composed = composed.FillPath(cue.MakePath(cue.Str(fqn)), composedByFQN[fqn])
		if composed.Err() != nil {
			return cue.Value{}, cue.Value{}, fmt.Errorf("building composed transformer map at %q: %w", fqn, composed.Err())
		}
	}

	matchers = octx.CompileString(`{resources: {}, traits: {}}`)
	for _, rfqn := range sortedKeys(resources) {
		matchers = matchers.FillPath(cue.MakePath(cue.Str("resources"), cue.Str(rfqn)), octx.NewList(resources[rfqn]...))
	}
	for _, tfqn := range sortedKeys(traits) {
		matchers = matchers.FillPath(cue.MakePath(cue.Str("traits"), cue.Str(tfqn)), octx.NewList(traits[tfqn]...))
	}
	if matchers.Err() != nil {
		return cue.Value{}, cue.Value{}, fmt.Errorf("building #matchers reverse index: %w", matchers.Err())
	}

	return composed, matchers, nil
}

// mapKeys returns the concrete string field labels of the map at path on v,
// or nil when the field is absent. Non-string labels are skipped.
func mapKeys(v cue.Value, path cue.Path) []string {
	m := v.LookupPath(path)
	if !m.Exists() {
		return nil
	}
	it, err := m.Fields()
	if err != nil {
		return nil
	}
	var keys []string
	for it.Next() {
		sel := it.Selector()
		if sel.LabelType() != cue.StringLabel {
			continue
		}
		keys = append(keys, sel.Unquoted())
	}
	return keys
}

// sortedKeys returns the keys of m in ascending order, for deterministic
// emission.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
