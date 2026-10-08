package platform

import (
	"fmt"
	"maps"
	"slices"

	"cuelang.org/go/cue"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/corepath"
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
	// requiredTraits name it, in the build's order. Required demands only:
	// optional consumption is tolerance, not fulfilment (0010:D32). A
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
	// than core 2.0.0-alpha.13, which cannot evaluate a colliding
	// platform at all.
	Collisions []string `json:"collisions"`

	// CollidingEntries maps each Collisions key to the ascending registry
	// keys (path@major) of the enabled entries whose catalogs list it.
	CollidingEntries map[string][]string `json:"collidingEntries"`
}

// Contracts returns the contract inventory (#Platform.#contracts) that was
// decoded once, when the platform was
// constructed (see [Platform]), or the refusal recorded in its place. It
// reads no Package on a constructed platform, and no kernel verb calls it.
// Each call returns its own copy: a caller may change the maps and slices
// it gets without changing what the next call returns, and callers on
// several goroutines share nothing.
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
// the same platform (see [Platform.CoreFloor]).
//
// The one exception is the collision report: an absent `collisions` or
// `collidingEntries` decodes as empty. Every core release carrying
// #contracts before core 2.0.0-alpha.13 folds definedBy over every
// enabled entry, and registry keys are distinct strings, so two enabled
// definers of one key always conflict there: such a core fails to evaluate
// a colliding platform, and a value that evaluated without the report
// provably has no collision. A collision field that is present but fails
// to decode is still an error.
func (p *Platform) Contracts() (*ContractInventory, error) {
	f := p.facts()
	if f.tooOld != nil {
		return nil, &oerrors.PlatformCoreTooOldError{Platform: p.name(), Field: f.tooOld.field, Since: f.tooOld.since, Require: schema.ProvidedBySince}
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.inv.clone(), nil
}

// CoreFloor reports whether the platform's #contracts carries providedBy,
// the provider count Kernel.Render's glue reads. It returns nil when it
// does, and otherwise the *oerrors.PlatformCoreTooOldError (Field
// "providedBy", Since and Require [schema.ProvidedBySince]) that
// Kernel.Render refuses the platform with before staging. Like
// [Platform.Contracts] it reads the fact recorded at construction, not
// Package. The floor is a presence test only: it holds for a platform whose
// #contracts carries providedBy but fails to evaluate, which Contracts
// refuses and a render fails in its build.
func (p *Platform) CoreFloor() error {
	if p.facts().providedBy {
		return nil
	}
	return &oerrors.PlatformCoreTooOldError{Platform: p.name(), Field: "providedBy", Since: schema.ProvidedBySince, Require: schema.ProvidedBySince}
}

// facts is what decodeFacts reads off a platform value, once.
type facts struct {
	// providedBy is the core floor: #contracts.providedBy exists.
	providedBy bool
	// inv is the decoded inventory, nil when tooOld or err is set.
	inv *ContractInventory
	// tooOld records a missing #contracts or report field as data, so
	// each Contracts call builds a fresh typed error naming the platform.
	tooOld *tooOld
	// err is a #contracts that did not evaluate or a field that failed to
	// decode. It is immutable and returned as recorded.
	err error
}

// tooOld is a missing #contracts or report field and the first core release
// carrying it.
type tooOld struct {
	field string
	since string
}

// facts returns the recorded facts, decoding them from Package on the first
// call. The constructor makes that first call; on a Platform it did not
// build, the first Contracts or CoreFloor call does.
func (p *Platform) facts() *facts {
	p.once.Do(func() { p.recorded = decodeFacts(p.Package) })
	return &p.recorded
}

// decodeFacts reads the core floor and decodes the contract inventory off a
// platform value. It never fails: a refusal is recorded in place of the
// inventory.
func decodeFacts(v cue.Value) facts {
	f := facts{providedBy: v.LookupPath(corepath.ContractsProvidedBy).Exists()}
	cv := v.LookupPath(corepath.Contracts)
	if !cv.Exists() {
		f.tooOld = &tooOld{field: "#contracts", since: "2.0.0-alpha.9"}
		return f
	}
	if err := cv.Err(); err != nil {
		f.err = fmt.Errorf("platform %s did not evaluate: %w", corepath.Contracts, err)
		return f
	}
	inv := &ContractInventory{}
	for _, d := range []struct {
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
		fv := cv.LookupPath(cue.ParsePath(d.name))
		if !fv.Exists() {
			f.tooOld = &tooOld{field: d.name, since: d.since}
			return f
		}
		if err := fv.Decode(d.into); err != nil {
			f.err = fmt.Errorf("decoding platform %s.%s: %w", corepath.Contracts, d.name, err)
			return f
		}
	}
	// The collision report, absent read as empty (see Contracts), never
	// behind a since-guard: a guard would refuse every platform pinning a
	// core between the floor and the report's first release.
	for _, d := range []struct {
		name string
		into any
	}{
		{"collisions", &inv.Collisions},
		{"collidingEntries", &inv.CollidingEntries},
	} {
		fv := cv.LookupPath(cue.ParsePath(d.name))
		if !fv.Exists() {
			continue
		}
		if err := fv.Decode(d.into); err != nil {
			f.err = fmt.Errorf("decoding platform %s.%s: %w", corepath.Contracts, d.name, err)
			return f
		}
	}
	if inv.Collisions == nil {
		inv.Collisions = []string{}
	}
	if inv.CollidingEntries == nil {
		inv.CollidingEntries = map[string][]string{}
	}
	f.inv = inv
	return f
}

// clone deep-copies the inventory: every map, every slice, and each
// Comparable row's Contracts. It keeps nil and empty apart, so a copy
// compares (reflect.DeepEqual, JSON null against []) exactly as the decoded
// value does.
func (inv *ContractInventory) clone() *ContractInventory {
	out := *inv
	out.DefinedBy = maps.Clone(inv.DefinedBy)
	out.RequiredBy = cloneListMap(inv.RequiredBy)
	out.ProvidedBy = cloneListMap(inv.ProvidedBy)
	out.Unfulfilled = slices.Clone(inv.Unfulfilled)
	out.OverSubscribed = slices.Clone(inv.OverSubscribed)
	out.Comparable = slices.Clone(inv.Comparable)
	for i := range out.Comparable {
		out.Comparable[i].Contracts = slices.Clone(out.Comparable[i].Contracts)
	}
	out.Collisions = slices.Clone(inv.Collisions)
	out.CollidingEntries = cloneListMap(inv.CollidingEntries)
	return &out
}

// cloneListMap copies a map of lists and each list, keeping nil and empty
// apart at both levels.
func cloneListMap(m map[string][]string) map[string][]string {
	if m == nil {
		return nil
	}
	out := make(map[string][]string, len(m))
	for k, v := range m {
		out[k] = slices.Clone(v)
	}
	return out
}

// name is the platform's metadata.name, or empty when no metadata was
// decoded; it names the platform in a refusal.
func (p *Platform) name() string {
	if p.Metadata == nil {
		return ""
	}
	return p.Metadata.Name
}
