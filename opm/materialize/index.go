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
//
// The walk also runs the single-provider guard (0010 D32/D37): for every
// required contract key any embedded copy declares `fulfilment: "provider"`,
// at most one subscribed catalog may supply a transformer for it, and
// embedded copies must agree on a key's fulfilment. See providerGuard.
func indexCatalogs(octx *cue.Context, builds []catalogBuild) (composed cue.Value, matchers cue.Value, err error) {
	// 1. Build the composed map, collapsing / conflicting on shared FQNs.
	composedByFQN := map[string]cue.Value{}
	guard := newProviderGuard()
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
			if gerr := guard.observe(b, tx); gerr != nil {
				return cue.Value{}, cue.Value{}, gerr
			}
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
	if gerr := guard.check(); gerr != nil {
		return cue.Value{}, cue.Value{}, gerr
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

// providerGuard is the single-provider guard (0010 D32 as corrected by D37).
// It keys on a contract's DECLARED fulfilment, read off the provider's
// embedded required copy — the only place materialize can reach a contract's
// definition, since #Catalog exposes no primitive maps. Counted are required
// demands only (requiredResources / requiredTraits); optional consumption is
// not supply. Catalog provenance is structural (catalogBuild.Subscription),
// never parsed back out of an FQN.
//
// Open acknowledgment (recorded in the change design): the embedded copy is
// the provider's CLAIM about the contract, not the declaring catalog's word —
// a lying provider shows the guard the lie. The disagreement error below is
// the partial mitigation until core exposes catalog primitive maps.
type providerGuard struct {
	// fulfilment records each key's stated fulfilment; source records the
	// subscription whose copy first stated it, for divergence attribution.
	fulfilment map[string]string
	source     map[string]string

	// providers maps each provider-fulfilled key to the set of subscribed
	// catalogs supplying a transformer that requires it.
	providers map[string]map[string]bool
}

func newProviderGuard() *providerGuard {
	return &providerGuard{
		fulfilment: map[string]string{},
		source:     map[string]string{},
		providers:  map[string]map[string]bool{},
	}
}

// observe reads the declared fulfilment off every REQUIRED embedded contract
// copy of tx, failing fast on copies that disagree for one key (divergent
// contract definitions — unifying them would mask a catalog bug).
func (g *providerGuard) observe(b catalogBuild, tx cue.Value) error {
	for _, path := range []cue.Path{schema.TransformerRequiredResources, schema.TransformerRequiredTraits} {
		m := tx.LookupPath(path)
		if !m.Exists() {
			continue
		}
		it, err := m.Fields()
		if err != nil {
			continue
		}
		for it.Next() {
			key := it.Selector().Unquoted()
			declared, stated := declaredFulfilment(it.Value())
			if !stated {
				// No concrete fulfilment on this copy (schema-bypassing
				// synthetic builds): nothing to guard on.
				continue
			}
			if prev, seen := g.fulfilment[key]; seen && prev != declared {
				return &oerrors.MaterializeError{
					Kind: oerrors.MaterializeKindCatalog, Subscription: b.Subscription, Version: b.Version,
					Cause: fmt.Errorf(
						"contract %q: embedded copies disagree on fulfilment: %q (from %q) vs %q (from %q)",
						key, prev, g.source[key], declared, b.Subscription),
				}
			} else if !seen {
				g.fulfilment[key] = declared
				g.source[key] = b.Subscription
			}
			if declared == "provider" {
				if g.providers[key] == nil {
					g.providers[key] = map[string]bool{}
				}
				g.providers[key][b.Subscription] = true
			}
		}
	}
	return nil
}

// check refuses a materialization in which transformers from more than one
// subscribed catalog supply one provider-fulfilled contract key. Iteration is
// key-sorted so the refusal is deterministic.
func (g *providerGuard) check() error {
	for _, key := range sortedKeys(g.providers) {
		subs := sortedKeys(g.providers[key])
		if len(subs) < 2 {
			continue
		}
		return &oerrors.MaterializeError{
			Kind: oerrors.MaterializeKindCatalog, Subscription: subs[1],
			Cause: fmt.Errorf(
				"contract %q declares fulfilment \"provider\" but is supplied by transformers from %d catalogs (%q and %q): a platform must carry exactly one provider for it",
				key, len(subs), subs[0], subs[1]),
		}
	}
	return nil
}

// declaredFulfilment reads the concrete fulfilment off an embedded contract
// copy, resolving the schema's default (*"catalog" | "provider"). stated is
// false when the field is absent or non-concrete with no default.
func declaredFulfilment(copy cue.Value) (fulfilment string, stated bool) {
	v := copy.LookupPath(cue.ParsePath("fulfilment"))
	if !v.Exists() {
		return "", false
	}
	if d, ok := v.Default(); ok {
		v = d
	}
	s, err := v.String()
	if err != nil {
		return "", false
	}
	return s, true
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
