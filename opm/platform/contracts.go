package platform

import (
	"fmt"

	"cuelang.org/go/cue"

	oerrors "github.com/open-platform-model/library/opm/errors"
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
// D2, D5, D18). Providers are counted per registry entry (path plus major),
// the one count the render's single-provider guard reads too: two majors of
// one catalog are two providers, two transformers of one entry are one, and
// a provider counts whether or not an enabled catalog defines the contract.
// Every field is a report: an inventory that is not Fulfilled,
// not Routable or not Discriminated, or that carries Collisions, is still a
// healthy value, and whether a generation step withholds a platform package
// on Routable or Discriminated false is that step's decision, outside this
// type.
//
// A contract key more than one enabled registry entry's catalog lists (two
// majors of one catalog sharing a key, say) is a collision: core folds only
// keys with exactly one enabled definer, so a colliding key is in Collisions
// and CollidingEntries and in none of DefinedBy, RequiredBy, Unfulfilled or
// Comparable. Fulfilled and Discriminated are computed without it and can
// therefore read true while Collisions is non-empty; a caller never reads
// either as safe while Collisions is non-empty. Routable reads false.
//
// `defined` (the listed members themselves) is not part of this view: its
// values are the catalogs' member schemas, non-concrete by construction, so
// there is no Go value a caller could use. A caller that wants one reads it
// off Platform.Package under #contracts.defined; DefinedBy carries the same
// key set.
type ContractInventory struct {
	// DefinedBy maps each contract FQN exactly one enabled catalog lists
	// to the registry key (the catalog's module path) of the catalog
	// listing it: the value every diagnostic prints beside the contract. A
	// colliding key (Collisions) is absent.
	DefinedBy map[string]string `json:"definedBy"`

	// RequiredBy maps each defined contract FQN to the implementation FQNs
	// of every enabled transformer whose requiredResources or
	// requiredTraits name it, in the build's order. Required demands only
	// (0010:D32: optional consumption is tolerance, not fulfilment); a
	// defined contract nothing requires maps to an empty list. A colliding
	// key is not defined, so it is absent however many transformers
	// require it.
	RequiredBy map[string][]string `json:"requiredBy"`

	// ProvidedBy maps every provider-fulfilled contract FQN some enabled
	// transformer requires (defined by an enabled catalog or not) to the
	// sorted registry keys (path@major) of the enabled entries whose
	// transformers require it. OverSubscribed is exactly its keys with two
	// or more entries; a key a defined provider contract lacks is
	// Unfulfilled. It is the field a caller prints to name the supplying
	// entries, including when DefinedBy and RequiredBy lack the key (the
	// defining catalog is disabled or absent).
	ProvidedBy map[string][]string `json:"providedBy"`

	// Unfulfilled lists the provider-fulfilled resources and traits an
	// enabled catalog defines and no enabled registry entry provides (no
	// ProvidedBy key). A report the operator surfaces as a non-gating
	// condition (0015:D18), never a refusal. A colliding key is not
	// defined, so it is never listed here.
	Unfulfilled []string `json:"unfulfilled"`

	// OverSubscribed lists the provider-fulfilled resources and traits
	// required by transformers of two or more enabled registry entries
	// (path plus major), whether or not an enabled catalog defines them:
	// the ProvidedBy keys with two or more entries, and what the generation
	// step refuses on (0010:D37). ProvidedBy names the entries.
	OverSubscribed []string `json:"overSubscribed"`

	// Comparable lists every pair of enabled transformers whose match
	// predicates are comparable over at least one shared catalog-fulfilled
	// contract (0015:D5), in the build's comprehension order. The accessor
	// does not sort; a caller that needs a stable order sorts. Only
	// defined contracts are compared, so no row shares a colliding key,
	// even when two transformers require one under equal predicates.
	Comparable []ComparablePredicates `json:"comparable"`

	// Fulfilled is true exactly when Unfulfilled is empty. It is blind to
	// Collisions: it can read true while Collisions is non-empty.
	Fulfilled bool `json:"fulfilled"`

	// Routable is true exactly when OverSubscribed and Collisions are both
	// empty.
	Routable bool `json:"routable"`

	// Discriminated is true exactly when Comparable is empty. It is blind
	// to Collisions: it can read true while Collisions is non-empty.
	Discriminated bool `json:"discriminated"`

	// Collisions lists, ascending, every contract FQN that two or more
	// enabled registry entries' catalogs list in their contract maps. Such
	// a key is in none of DefinedBy, RequiredBy, Unfulfilled or
	// Comparable, and Routable is false while any exists. A disabled entry
	// never counts as a definer. Empty on a platform pinning a core older
	// than [schema.CollisionsSince], which cannot evaluate a colliding
	// platform at all.
	Collisions []string `json:"collisions"`

	// CollidingEntries maps each Collisions key to the ascending registry
	// keys (path@major) of the enabled entries whose catalogs list it.
	CollidingEntries map[string][]string `json:"collidingEntries"`
}

// Contracts decodes the contract inventory off Package on demand
// (#Platform.#contracts, [schema.Contracts]). Nothing decodes it at
// construction and no kernel verb calls it: the value is already built, and
// it is read only when a caller asks (the operator's readiness loop, a
// platform check), so a render pays nothing for it.
//
// The eleven data fields are read by path, so the decoded set is exactly
// this type's field list; `defined` stays on Package (see
// [ContractInventory]). Reports, never refusals: an over-subscribed,
// colliding or undiscriminated platform returns Routable or Discriminated
// false and a nil error.
//
// A report the value does not carry is never defaulted, in either
// direction: a platform whose #contracts predates a report field is refused
// with a [*oerrors.PlatformCoreTooOldError] naming the missing field, the
// first core release carrying it and the floor to re-pin to
// ([schema.ProvidedBySince]), never returned as a partial inventory
// whose missing verdict would read as a pass. The same refusal covers a
// platform carrying no #contracts at all (Field "#contracts": a value built
// against a core release before 2.0.0-alpha.9, or one that is not a
// #Platform). A missing providedBy is the refusal Kernel.Render returns for
// the same platform.
//
// The one exception is the collision report: an absent `collisions` or
// `collidingEntries` decodes as empty. Every core release carrying
// #contracts before [schema.CollisionsSince] folds definedBy over every
// enabled entry, and registry keys are distinct strings, so two enabled
// definers of one key always conflict there: such a core fails to evaluate
// a colliding platform, and a value that evaluated without the report
// provably has no collision. A collision field that is present but fails
// to decode is still an error.
func (p *Platform) Contracts() (*ContractInventory, error) {
	cv := p.Package.LookupPath(schema.Contracts)
	if !cv.Exists() {
		return nil, &oerrors.PlatformCoreTooOldError{Platform: p.name(), Field: "#contracts", Since: "2.0.0-alpha.9", Require: schema.ProvidedBySince}
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
		{"providedBy", &inv.ProvidedBy, schema.ProvidedBySince},
	} {
		v := cv.LookupPath(cue.ParsePath(f.name))
		if !v.Exists() {
			return nil, &oerrors.PlatformCoreTooOldError{Platform: p.name(), Field: f.name, Since: f.since, Require: schema.ProvidedBySince}
		}
		if err := v.Decode(f.into); err != nil {
			return nil, fmt.Errorf("decoding platform %s.%s: %w", schema.Contracts, f.name, err)
		}
	}
	// The collision report, absent read as empty (see above), never behind
	// a since-guard: a guard would refuse every platform pinning a core
	// between the floor and the report's first release.
	for _, f := range []struct {
		name string
		into any
	}{
		{"collisions", &inv.Collisions},
		{"collidingEntries", &inv.CollidingEntries},
	} {
		v := cv.LookupPath(cue.ParsePath(f.name))
		if !v.Exists() {
			continue
		}
		if err := v.Decode(f.into); err != nil {
			return nil, fmt.Errorf("decoding platform %s.%s: %w", schema.Contracts, f.name, err)
		}
	}
	if inv.Collisions == nil {
		inv.Collisions = []string{}
	}
	if inv.CollidingEntries == nil {
		inv.CollidingEntries = map[string][]string{}
	}
	return inv, nil
}

// name is the platform's metadata.name, or empty when no metadata was
// decoded; it names the platform in a refusal.
func (p *Platform) name() string {
	if p.Metadata == nil {
		return ""
	}
	return p.Metadata.Name
}
