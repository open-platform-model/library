// Package materialize realizes a #Platform's path-keyed catalog
// subscriptions into a sealed [MaterializedPlatform].
//
// A #Platform authored against opmodel.dev/core@v0.3.0 carries only a
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
// The result is a [MaterializedPlatform] whose Package answers the exact
// LookupPath calls the matcher already makes (schema.ComposedTransformers,
// schema.MatchersResources, schema.MatchersTraits), so the downstream match
// rewrite consumes it with a minimal diff.
//
// Materialize performs I/O (registry enumeration + OCI pulls) and is
// explicit and caller-driven: the kernel holds no cache (Principle I).
// Consumers that want memoization wire their own via opm/materialize/cache.
package materialize
