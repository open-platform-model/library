// Package inventory is the runtime-neutral, shared inventory contract for OPM.
//
// "Inventory" answers one question: which Kubernetes objects does a rendered
// release own, and which of those are now stale relative to a fresh render.
// Historically this logic existed twice — once in the operator
// (opm-operator/internal/inventory) and once in the CLI (cli/pkg/inventory) —
// and the two copies drifted, computing different prune sets for the same
// release. This package is the single canonical implementation both frontends
// consume so that entry identity, content digests, and the stale/prune set are
// computed by the same code, which is the correctness precondition for the
// enhancement 0006 CLI↔operator handoff.
//
// This is kernel core, NOT opt-in helper code: it lives under opm/ (not
// opm/helper/) because it defines a contract every OPM frontend MUST compute
// identically, not a convenience a frontend MAY skip.
//
// Neutrality constraints (enhancement 0006 D13):
//
//   - The package and its types MUST NOT import sigs.k8s.io/controller-runtime,
//     any github.com/fluxcd/* package, or k8s.io/apiextensions-apiserver.
//     Importing the operator's CRD InventoryEntry would transitively drag that
//     whole stack into every consumer; this package's InventoryEntry is a plain
//     value type with none of it.
//   - Its only non-standard-library dependency is k8s.io/apimachinery, used for
//     unstructured access and object-identity primitives.
//   - All functions are pure: no cluster I/O, no logging. Fetching live cluster
//     state and reacting to verdicts are caller (CLI/operator) responsibilities,
//     per the kernel-neutrality constitution (I/O at the edges, no direct
//     logging).
//
// Consumers map their own entry shapes to and from InventoryEntry at their
// boundary: the operator maps api/v1alpha1.InventoryEntry, the CLI maps its own
// struct. The field set is identical across all three, so the mapping is
// mechanical and total.
package inventory
