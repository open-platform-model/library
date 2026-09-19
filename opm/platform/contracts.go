package platform

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/schema"
)

// ComparablePredicates is one row of the comparable-predicate report: two
// enabled transformers whose match predicates are comparable over at least
// one shared catalog-fulfilled contract (0015:D5). A
// transformer's predicate is every required demand it declares — resources,
// traits and label key-value pairs together — and a pair is comparable when
// one predicate contains the other, so Broader matches every component
// Narrower matches and the two are never told apart by a component's shape.
// A report row, never a refusal: the generation step refuses.
type ComparablePredicates struct {
	// Broader is the implementation FQN of the transformer whose predicate
	// contains the other's: it matches every component Narrower matches.
	Broader string `json:"broader"`

	// Narrower is the implementation FQN of the contained transformer.
	Narrower string `json:"narrower"`

	// Contracts lists the catalog-fulfilled contract FQNs both predicates
	// require, the bucket the pair is undiscriminated within.
	Contracts []string `json:"contracts"`
}

// ContractInventory is the decoded view of #Platform.#contracts, the
// inventory core derives from the enabled registry entries' contract maps
// and the required demands of #composedTransformers (0015:D1,
// D2, D5, D18). Every field is a report: an inventory that is not Fulfilled,
// not Routable or not Discriminated is still a healthy value, and whether a
// generation step withholds a platform package on Routable or Discriminated
// false is that step's decision, outside this type.
//
// `defined` (the listed members themselves) is not part of this view: its
// values are the catalogs' member schemas, non-concrete by construction, so
// there is no Go value a caller could use. A caller that wants one reads it
// off Platform.Package under #contracts.defined; DefinedBy carries the same
// key set.
type ContractInventory struct {
	// DefinedBy maps each contract FQN an enabled catalog lists to the
	// registry key (the catalog's module path) of the catalog listing it:
	// the value every diagnostic prints beside the contract.
	DefinedBy map[string]string `json:"definedBy"`

	// RequiredBy maps each defined contract FQN to the implementation FQNs
	// of every enabled transformer whose requiredResources or
	// requiredTraits name it, in the build's order. Required demands only
	// (0010:D32: optional consumption is tolerance, not fulfilment); a
	// defined contract nothing requires maps to an empty list.
	RequiredBy map[string][]string `json:"requiredBy"`

	// Unfulfilled lists the provider-fulfilled resources and traits
	// required by nothing on the platform. A report the operator surfaces
	// as a non-gating condition (0015:D18), never a refusal.
	Unfulfilled []string `json:"unfulfilled"`

	// OverSubscribed lists the provider-fulfilled resources and traits
	// required by transformers from more than one catalog: what the
	// generation step refuses on (0010:D37).
	OverSubscribed []string `json:"overSubscribed"`

	// Comparable lists every pair of enabled transformers whose match
	// predicates are comparable over at least one shared catalog-fulfilled
	// contract (0015:D5), in the build's comprehension order. The accessor
	// does not sort; a caller that needs a stable order sorts.
	Comparable []ComparablePredicates `json:"comparable"`

	// Fulfilled is true exactly when Unfulfilled is empty.
	Fulfilled bool `json:"fulfilled"`

	// Routable is true exactly when OverSubscribed is empty.
	Routable bool `json:"routable"`

	// Discriminated is true exactly when Comparable is empty.
	Discriminated bool `json:"discriminated"`
}

// Contracts decodes the contract inventory off Package on demand
// (#Platform.#contracts, [schema.Contracts]). Nothing decodes it at
// construction and no kernel verb calls it: the value is already built, and
// it is read only when a caller asks (the operator's readiness loop, a
// platform check), so a render pays nothing for it.
//
// The eight data fields are read by path, so the decoded set is exactly
// this type's field list; `defined` stays on Package (see
// [ContractInventory]). Reports, never refusals: an over-subscribed or
// undiscriminated platform returns Routable or Discriminated false and a
// nil error.
//
// A report the value does not carry is never defaulted, in either
// direction: a platform whose #contracts predates a report field is refused
// with an error naming the missing field and the first core release
// carrying it, never returned as a partial inventory whose missing verdict
// would read as a pass. The same refusal covers a platform carrying no
// #contracts at all (a value built against a core release before
// 2.0.0-alpha.9, or one that is not a #Platform).
func (p *Platform) Contracts() (*ContractInventory, error) {
	cv := p.Package.LookupPath(schema.Contracts)
	if !cv.Exists() {
		return nil, fmt.Errorf("platform carries no %s field (the contract inventory core derives from release 2.0.0-alpha.9 on)", schema.Contracts)
	}
	if err := cv.Err(); err != nil {
		return nil, fmt.Errorf("platform %s did not evaluate: %w", schema.Contracts, err)
	}
	inv := &ContractInventory{}
	for _, f := range []struct {
		name string
		into any
		// since is the first core release deriving the field, named in the
		// refusal so an older platform module knows what to re-pin to.
		since string
	}{
		{"definedBy", &inv.DefinedBy, "2.0.0-alpha.9"},
		{"requiredBy", &inv.RequiredBy, "2.0.0-alpha.9"},
		{"unfulfilled", &inv.Unfulfilled, "2.0.0-alpha.9"},
		{"overSubscribed", &inv.OverSubscribed, "2.0.0-alpha.9"},
		{"comparable", &inv.Comparable, "2.0.0-alpha.10"},
		{"fulfilled", &inv.Fulfilled, "2.0.0-alpha.9"},
		{"routable", &inv.Routable, "2.0.0-alpha.9"},
		{"discriminated", &inv.Discriminated, "2.0.0-alpha.10"},
	} {
		v := cv.LookupPath(cue.ParsePath(f.name))
		if !v.Exists() {
			return nil, fmt.Errorf("platform %s carries no %q field (core derives it from release %s on)", schema.Contracts, f.name, f.since)
		}
		if err := v.Decode(f.into); err != nil {
			return nil, fmt.Errorf("decoding platform %s.%s: %w", schema.Contracts, f.name, err)
		}
	}
	return inv, nil
}
