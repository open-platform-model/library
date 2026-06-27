// Package materialize realizes a #Platform's path-keyed catalog
// subscriptions into a sealed [MaterializedPlatform].
//
// A #Platform authored against opmodel.dev/core@v1.0.0-alpha.1 carries only a
// #registry of subscriptions; its #composedTransformers / #matchers slots
// are empty (the schema marks them optional, kernel-filled).
//
// Materialize is the kernel step that fills those slots. For each enabled
// subscription it:
//
//  1. enumerates the published versions of the subscribed catalog path;
//  2. narrows them Go-side by the subscription filter — range ∧ allow ∧ deny,
//     parsed as SemVer constraints because CUE cannot evaluate range syntax;
//  3. pulls each surviving version through cue/load against the configured
//     OCI registry;
//  4. reads each build's #Catalog.#transformers map; and
//  5. indexes every transformer by its stamped FQN into a composed transformer
//     map, plus a #matchers reverse index over the primitive FQNs those
//     transformers reference.
//
// The result is a [MaterializedPlatform] that exposes the composed transformer
// map and the #matchers reverse index as native first-class fields —
// Transformers (FQN → #ComponentTransformer) and Matchers ({resources, traits})
// — built in the owner *cue.Context by indexCatalogs. They are NOT filled onto
// the closed c.#Platform (ADR-003): the matcher and executor read them off the
// native fields, and a #transform read off Transformers renders concrete
// because no closed twin is ever constructed. The closed platform spec stays
// reachable as Source.Package for #registry / metadata / diagnostics.
//
// Materialize performs I/O (registry enumeration + OCI pulls) and is
// explicit and caller-driven: the kernel holds no cache (Principle I).
// Consumers that want memoization wire their own via opm/materialize/cache.
package materialize
