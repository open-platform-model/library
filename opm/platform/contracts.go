package platform

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/schema"
)

// ContractInventory is the decoded view of #Platform.#contracts, the
// inventory core derives from the enabled registry entries' contract maps
// and the required demands of #composedTransformers (enhancement 0015 D1,
// D2, D18). Every field is a report: an inventory that is not Fulfilled or
// not Routable is still a healthy value, and whether a generation step
// withholds a platform package on Routable false is that step's decision,
// outside this type.
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
	// (0010 D32: optional consumption is tolerance, not fulfilment); a
	// defined contract nothing requires maps to an empty list.
	RequiredBy map[string][]string `json:"requiredBy"`

	// Unfulfilled lists the provider-fulfilled resources and traits
	// required by nothing on the platform. A report the operator surfaces
	// as a non-gating condition (D18), never a refusal.
	Unfulfilled []string `json:"unfulfilled"`

	// OverSubscribed lists the provider-fulfilled resources and traits
	// required by transformers from more than one catalog: what the
	// generation step refuses on (0010 D37).
	OverSubscribed []string `json:"overSubscribed"`

	// Fulfilled is true exactly when Unfulfilled is empty.
	Fulfilled bool `json:"fulfilled"`

	// Routable is true exactly when OverSubscribed is empty.
	Routable bool `json:"routable"`
}

// Contracts decodes the contract inventory off Package on demand
// (#Platform.#contracts, [schema.Contracts]). Nothing decodes it at
// construction and no kernel verb calls it: the value is already built, and
// it is read only when a caller asks (the operator's readiness loop, a
// platform check), so a render pays nothing for it.
//
// The six data fields are read by path, so the decoded set is exactly this
// type's field list; `defined` stays on Package (see [ContractInventory]).
// Reports, never refusals: an over-subscribed platform returns Routable
// false and a nil error. The one error is a platform carrying no #contracts
// (a value built against a core release before 2.0.0-alpha.9, or one that
// is not a #Platform at all), named by the missing field.
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
	}{
		{"definedBy", &inv.DefinedBy},
		{"requiredBy", &inv.RequiredBy},
		{"unfulfilled", &inv.Unfulfilled},
		{"overSubscribed", &inv.OverSubscribed},
		{"fulfilled", &inv.Fulfilled},
		{"routable", &inv.Routable},
	} {
		v := cv.LookupPath(cue.ParsePath(f.name))
		if !v.Exists() {
			return nil, fmt.Errorf("platform %s carries no %q field", schema.Contracts, f.name)
		}
		if err := v.Decode(f.into); err != nil {
			return nil, fmt.Errorf("decoding platform %s.%s: %w", schema.Contracts, f.name, err)
		}
	}
	return inv, nil
}
