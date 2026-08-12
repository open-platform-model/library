// Package materialize realizes a #Platform's path-keyed catalog
// subscriptions into a sealed [MaterializedPlatform].
//
// A #Platform authored against opmodel.dev/core@v2 carries only a
// #registry of subscriptions; its #composedTransformers / #matchers slots
// are empty (the schema marks them optional, kernel-filled).
//
// Materialize is the kernel step that fills those slots. For each enabled
// subscription it:
//
//  1. reads the authored `version!` scalar — the single build the
//     subscription materializes (0010 D14: catalog selection is a pure
//     function of committed source; the platform file IS the resolution) —
//     and checks it sits in the subscription key's major;
//  2. pulls exactly that build through cue/load against the configured OCI
//     registry (the happy path makes no enumeration round-trip; when the
//     pull fails, the published list is enumerated lazily to enrich the
//     error);
//  3. verifies the pulled catalog's declared identity — metadata.modulePath
//     against the subscription key, metadata.version against the pulled tag
//     (D11/D9: the kernel is the version label's verifier, never its
//     source);
//  4. reads the build's #Catalog.#transformers map; and
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
