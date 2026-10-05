## Context

See proposal.md, Why. Design-local decisions are numbered KO1 to KO11 so they collide with no
other numbering. Line references are at library `origin/main` `a8bfc76` (after lib-consolidate
and lib-c2), cli `origin/main` `0b37e3f2` and opm-operator `origin/main` `9b83611`, all fetched
2026-10-05. Re-checked at cli `origin/main` `1338e700` (after cli#316, cli#317 and cli#318):
`pkg/core` and `pkg/resourceorder` are unchanged since `0b37e3f2`, and the KO2 consumer lines
still hold. `0b37e3f2` stays the provenance commit for the ported code (the weight table last
changed in `75e5f268`). Evidence comes from the wave-2 research entries e2 and e5
(`claude-stuff/kernel-plan-beta1/wave2-plan-result.json`), re-checked at those heads. One thing
changed since the research: lib-consolidate merged and `go.mod` now carries `golang.org/x/mod`.

## Goals / Non-Goals

**Goals:**
- Add a dependency-free label vocabulary.
- Add one `Resource` wrapper and a one-export conversion that serves digest, apply objects and
  inventory.
- Add the cli's weight table, unchanged, with sort and stages.
- Add the duplicate-identity check in its tier home.
- Mark the helper copy deprecated.
- Raise the Go floor to match both frontends.
- Make the "planned" lines true.

**Non-Goals:**
- Any frontend edit.
- Removing `opm/helper/objectset`.
- Digests, inventory entries, ownership verdicts, health and deletion plans (later tier
  changes).
- The g1 bytes switch.
- Changing a weight.
- A release rule for apimachinery bumps.

## Research & Decisions

### KO1: Go 1.26.0 and apimachinery v0.36.4; the lint pin follows if it must

**Context**: `k8s.io/apimachinery v0.36.4`, the version both frontends pin (cli `go.mod:21`,
opm-operator `go.mod:21`), declares `go 1.26.0`. The library declares `go 1.25.0`.
**Explored**: The research offered two options: raise the directive, or pin apimachinery 0.35.x
(go 1.25.0) and let MVS pick 0.36.4 in the frontends. The cli declares `go 1.26.0` and the
operator `go 1.26.2`. CI's `lint.yml` installs golangci-lint `2.8.0`. golangci-lint refuses to
lint a module whose `go` directive is newer than the Go it was built with, and 2.8.0 predates
Go 1.26. The cli's CI pins `v2.11.3`.
**Decision**: Supervisor decision SD10 applies: `go 1.26.0` and `k8s.io/apimachinery v0.36.4`.
The directive moves in section 1, and apimachinery arrives in section 3 with its first
importer, because `go mod tidy` drops a requirement nothing imports. Section 1 checks the lint
pin: it runs golangci-lint 2.8.0 from a scratch install against the raised directive. If 2.8.0
refuses, `lint.yml` moves to 2.11.3, the cli's pin, with the sha256 from that release's
`checksums.txt`, both values together as the step's comment requires.
**Outcome (section 1)**: golangci-lint 2.8.0 (built with go1.25.5) refuses the raised
directive ("the Go language version (go1.25) used to build golangci-lint is lower than the
targeted Go version (1.26.0)"), so `lint.yml` moves to 2.11.3 with the sha256 of
`golangci-lint-2.11.3-linux-amd64.tar.gz` from that release's checksums file, checked against
the downloaded archive.
**Rationale**: This matches both frontends, so the floor costs neither of them anything. It is
an assumption until section 1 has run, so section 1 is the spike.

### KO2: opm/k8s/labels: names without stutter, every constant kept, stdlib only

**Context**: The frontends' `pkg/core/labels.go` has nine constants and `IsOPMManagedBy`.
**Explored**: Every constant has a non-test consumer at `origin/main`. `LabelComponent` is used
by cli `internal/inventory/legacy.go:68`. `LabelModuleInstanceNamespace` is used by cli
`internal/inventory/store.go:338` and `internal/operator/migration_proof.go:36,138`.
`LabelManagedByLegacyValue` is used inside `IsOPMManagedBy`. Nothing may be dropped.
**Decision**:

```go
package labels // imports nothing

const (
	ManagedBy           = "app.kubernetes.io/managed-by"
	ManagedByCLI        = "opm-cli"             // core's #runtimeName for the cli
	ManagedByController = "opm-controller"      // core's #runtimeName for the operator
	ManagedByLegacy     = "open-platform-model" // recognised, never stamped
	Component           = "opmodel.dev/component" // the OPM object category, not a component name
	ComponentName           = "component.opmodel.dev/name"
	ModuleInstanceName      = "module-instance.opmodel.dev/name"
	ModuleInstanceNamespace = "module-instance.opmodel.dev/namespace"
	ModuleInstanceUUID      = "module-instance.opmodel.dev/uuid"
)

func IsOPMManagedBy(value string) bool // ManagedByCLI, ManagedByController or ManagedByLegacy
```

Each doc comment says who writes the label. Core's CUE stamps managed-by, the instance name and
UUID, and the component name. Only the cli writes `Component` and `ModuleInstanceNamespace`,
and only on its own inventory objects. `Component` keeps the name its key carries, so a reader
matches it to `opmodel.dev/component` at a glance, but its doc comment opens with "the OPM
object category (the cli writes \"inventory\"), not a component name" and points at
`ComponentName` for the component a rendered object came from. A rename to `ObjectCategory`
was weighed and not taken: it would hide the key, and the doc comment carries the warning where
an IDE shows it. The package doc states the rule: it names and
recognises labels, and it never stamps them on rendered objects (0012:D6). The values are
byte-equal to the frontends' copies, and a table test pins each one. lib-e4 later adds the
adopt annotation key here.
**Rationale**: `labels.LabelManagedBy` stutters. The frontends rewrite every import line when
they adopt anyway (ADR-011 item 3, no aliases), so the rename costs them nothing extra.

### KO3: Resource wraps the kernel's value; constructors replace the adapters

**Context**: Owner g1: "e2 ships alone; opm/k8s/object.Resource wraps the Value for now".
**Decision**:

```go
type Resource struct {
	Value       cue.Value // concrete, fully evaluated; pins its build (holder-bounded, ADR-007)
	Instance    string
	Component   string
	Transformer string
}

func NewResource(c *kernel.Compiled) *Resource          // nil in, nil out
func Resources(compiled []*kernel.Compiled) []*Resource // nil entries skipped
// Kind, Name, Namespace, APIVersion, GVK, Labels, Annotations, String,
// MarshalJSON, ToUnstructured: ported verbatim from pkg/core.
```

The fields stay exported and in the same order, so the frontends' many test literals
(`&core.Resource{Value: ..., Component: ...}`) migrate by import path alone. The behaviour of
each accessor, including the best-effort empty string and nil maps, ports with the frontends'
`resource_test.go` cases. The doc states the retention rule. A `Resource` keeps its whole build
alive. A long-lived caller exports and then drops its Resources. The library never drops them
for the caller.
**Rationale**: This is a literal reading of g1. Keeping the field set lets
`ResourceFromCompiled` and the cli's inline loop collapse into `NewResource` and `Resources`.

### KO4: Export is the one CUE export per resource

**Context**: The operator's `convertRender` exports each value once, hashes the bytes, decodes
the same bytes into unstructured objects and drops `result.Resources`. The cli exports twice
(`inventory/digest.go:38`, then `ToUnstructured`). x6 requires that the operator can derive
inventory entries from the converted set.
**Decision**:

```go
type Exported struct {
	JSON        []byte                     // the CUE export, field order as CUE emits it
	Object      *unstructured.Unstructured // decoded from JSON, not exported again
	Instance    string
	Component   string
	Transformer string
}

// Export exports each resource once, in input order (index-aligned). It never
// mutates or drops the input; the caller drops its Resources after.
func Export(resources []*Resource) ([]Exported, error)

type ExportStep int
const (
	ExportMarshal ExportStep = iota // the CUE value would not export to JSON
	ExportDecode                    // the JSON would not decode to an object
)

type ExportError struct {
	Index    int
	Resource string // Resource.String() of the failing input
	Step     ExportStep
	Err      error
}
```

`Export` stops at the first failure. Decoding is into `map[string]any`, so any JSON value that
is not an object fails with `ExportDecode`: a concrete list or string fails in `json.Unmarshal`,
and a concrete `null`, which `json.Unmarshal` accepts as a nil map, is refused by an explicit
nil check, so `Exported.Object.Object` is never nil. `ToUnstructured` stays a verbatim port and
does not gain that check. `ExportError.Error()` names the resource and the step, and
`Unwrap` returns `Err`. The operator needs `Step`, because it maps the two failures to different
condition reasons today (`RenderFailedReason` for the export, `ApplyFailedReason` for the
decode). Tests assert four things: `Exported.JSON` equals the resource's own `MarshalJSON`
bytes, `Exported.Object` equals what those bytes decode to, the input slice and its
Resources are untouched, and a list, a string and `null` each fail with `ExportDecode` and the
right `Index`. That each value is exported once is structural: `Export` calls
`MarshalJSON` once per item and decodes from its result. It is checked in review, because no
public seam can count CUE exports.
**Rationale**: The digest (lib-e3), apply objects and inventory entries all read `Exported`, so
no caller exports twice. When g1 lands, `Export` becomes a byte copy inside this package and
its signature holds.

### KO5: The weight table is ported unchanged from the cli

**Context**: Owner e5. The supervisor note requires that the order match what cli#289 (a1) and
the operator use today, and that the cli copy be cited.
**Decision**: Copy `cli/pkg/resourceorder/weights.go` and `sort.go` at cli `origin/main`
`0b37e3f2` (last touched by `75e5f268`, cli#289) into `opm/k8s/object`, values unchanged:
- the 25 `Weight*` constants;
- the GVK table and the kind-only fallback table;
- `WeightDefault` 1000;
- `Direction` with `Ascending` and `Descending`;
- the stable generic `Sort[T]`.

The only rename is `GetWeight` to `Weight`. The cli's `weights_test.go` and `sort_test.go` port
with it. A guard test pins every constant's value and every table entry, so a later edit is
deliberate. The file header comment cites the source commit once.
**Rationale**: "Matches what cli#289 uses" holds by construction. The operator has no table of
its own today (0012:D5 deleted it). It uses Flux's order, which KO7 documents.

### KO6: Stages cut an apply set by the order (SD11)

**Context**: SD11 says the operator applies one library weight group per fluxssa `ApplyAll`,
because Flux re-sorts every call with its own table (ssa v0.77.0 `manager_apply.go:207`). The
cli's apply has two stages: the cluster definitions (`IsProtectedKind`: a
CustomResourceDefinition in `apiextensions.k8s.io`, a Namespace in the core group), an
Established wait, then the rest.
**Decision**:

```go
type Stage[T any] struct {
	ClusterDefinitions bool // CRDs and core-group Namespaces; wait for them before the next stage
	Items              []T
}

// Stages returns a sorted copy of items cut into apply stages: the cluster
// definitions first (one stage, omitted when empty), then one stage per
// distinct weight of the rest, ascending. Stable within a stage; the input
// slice is not reordered; no empty stage is returned.
func Stages[T any](items []T, gvkOf func(T) schema.GroupVersionKind) []Stage[T]
```

The definition stage spans two weights, -100 (CustomResourceDefinition) and 0 (Namespace), so
it is not one library weight group in SD11's literal sense. It is still safe to hand to one
`ApplyAll` call, because Flux v0.77.0 ranks CustomResourceDefinition before Namespace too
(`sort.go` `ReconcileOrder.First[0]` and `[1]`), so Flux's re-sort inside the call keeps the
library's order. `flux_order_test.go` asserts that this pair is no contradiction.

The cluster definitions are matched on group and kind, so a `Namespace` kind in another API
group is not one. It still takes the kind-only fallback weight 0 in its weight stage. Flux's own
definition stage also holds ClusterRoles. The library's does not, because the cli's does not.
A ClusterRole lands in the weight-5 stage that follows. The operator's adoption change decides
whether it waits on that stage. Deletion uses `Sort(..., Descending)` and needs no stages.
**Rationale**: SD11 needs exactly this cut. The cli can flatten every stage after the first and
keep its two-stage apply. One function then defines "stage" for both frontends.

### KO7: The Flux comparison is a documentation test, not a dependency

**Context**: The e5 research found that Flux's `ReconcileOrder` and the library table disagree,
and that the disagreement is invisible today.
**Decision**: `flux_order_test.go` lists the 25 kinds of `ReconcileOrder.First` and `.Last`
from fluxcd/pkg/ssa v0.77.0 `sort.go` as string literals, with that version in a comment. Flux
itself is denied by depguard and is not imported. The test's universe is Flux's 25 kinds, the
26 kinds of the library's kind table and one representative custom kind (`Widget`). Flux ranks
`First` kinds -23 to -1 and `Last` kinds 1 and 2 (`computeKind2index`), and every kind it does
not list, including `Widget`, ranks 0 (ties then break by group and kind, which the test does
not model). The test resolves each kind to a library weight through the kind-only lookup. It
counts a pair only when the Flux ranks differ and the library weights are strictly opposite. It asserts that the set equals a committed list,
so adding or removing a contradiction fails the test and needs a reviewed edit of the list.
Known members include PersistentVolume and PersistentVolumeClaim after Deployment in Flux
(unlisted, rank 0) but before it in the library, the `*Class` kinds early in Flux but at the default weight in the
library, and the webhook configurations last in Flux but at 500, before custom resources at
1000, in the library (the custom resource ranks 0 in Flux, between the two). It also asserts that
CustomResourceDefinition before Namespace is no contradiction, the pair KO6's definition stage
rests on. The test decides nothing. Under SD11 these contradictions cannot appear
inside one `ApplyAll` call, because each call holds a single weight.
**Rationale**: This makes the e1 rule "an engine may refine, never contradict" checkable in
review without importing the engine.

### KO8: Duplicates is copied into the tier; the helper copy is deprecated and frozen

**Context**: ADR-011 item 9 says objectset moves. SD1 forbids deleting it while the frontends
import it. The fences forbid forwarding in either direction. `opm/helper` may not import
`opm/k8s` (rule `nothing-imports-k8s-tier`), and `opm/k8s` may not import `opm/helper` (rule
`k8s-tier-imports-no-runtime`). Test files are included in both rules.
**Decision**: Copy `objectset.go` into `opm/k8s/object/duplicates.go`. The exported names and
the signature `Duplicates(compiled []*kernel.Compiled) []Duplicate` stay the same. The
identity read, the group-only key, row order and error wording do not change. The test file is
ported whole. The helper package keeps its code, and each exported symbol and the package doc
gain a `Deprecated: use opm/k8s/object.<Name>` paragraph. The in-repo importers outside the helper
get `//nolint:staticcheck` with the reason "SA1019: the deprecated copy is tested until its
removal" on every line staticcheck reports, the import and each use:
`opm/kernel/flow_integration_test.go` (the import and the `objectset.Duplicates` call) and
`opm/kernel/render_test.go` (the import and the `objectset.Duplicates` call), which may not
import the tier. The helper's own external test package `objectset_test` gets none, because
staticcheck (v0.7.0 `sa1019.go`) does not report a package's own external test package using
it. The removal change deletes the kernel test uses.
The ported test carries the same synthetic transformer id
(`opmodel.dev/catalogs/opm/transformer-registration-transformer@4.4.0`) as the helper test, so
`opm/k8s/object/duplicates_test.go` gets its own `.cascade-frozen` entry with the pin
`opmodel.dev/catalogs/opm@v4`, appended at the end of the file.
**Rationale**: Two identical copies for one release window is the price of SD1. The copy is
mechanical and the identical test suites pin it. No cross-package parity test is possible,
because the fences forbid it.

### KO9: A strict import allow list for the tier pays

**Context**: ADR-011 item 2 left one question to this change: "Any other third-party import
into the tier is held by review until add-kubernetes-object-packages decides whether a strict
allow list pays."
**Decision**: Yes. Two depguard rules:
- `k8s-labels-imports-only-stdlib`: strict, files `**/opm/k8s/labels/**` except `_test.go`,
  allow `$gostd`.
- `k8s-tier-allow-list`: strict, files `**/opm/k8s/**` except `_test.go`, allow `$gostd`,
  `cuelang.org/go/cue`, `k8s.io/apimachinery` and `github.com/open-platform-model/library/opm`.

The existing deny rules still refuse `opm/internal`, `opm/helper`, Flux and every other
Kubernetes module, so the broad library prefix admits only the kernel's exported packages and
tier siblings. Test files stay outside the strict rules, so testify is allowed, and the
existing deny rules still cover them. Section 1 proves the two assumptions this rests on with
mutation checks: that depguard v2 applies every matching rule (a strict allow and a lax deny
both bite), and that `$gostd` works in this golangci-lint version.
The spec states this as a MODIFIED of the existing requirement "The Kubernetes tier imports no
cluster client or controller framework", which already named the allowed set, rather than a
second requirement owning the same rule.
**Outcome (section 1)**: every matching rule applies and `$gostd` works in golangci-lint
2.11.3. Depguard names the first rule that refuses an import, one finding per import. In a
non-test tier file `opm/internal/modversion` is refused by the lax `k8s-tier-imports-no-runtime`
although the strict list's library prefix admits it, and `k8s.io/client-go` and
`github.com/google/uuid` are refused by `k8s-tier-allow-list`. In a tier test file, outside the
strict list, `k8s.io/client-go` and `k8s.io/utils` are refused by
`k8s-tier-imports-only-apimachinery` and `opm/helper` by `k8s-tier-imports-no-runtime`.
`opm/k8s/labels` refuses `k8s.io/apimachinery`. apimachinery, `cuelang.org/go/cue` and
`opm/kernel` pass in `opm/k8s/object`, and `strings` passes in `opm/k8s/labels`.
**Rationale**: The list is four entries and costs one lint rule. Without it, a new third-party
import into a mandatory tier, which reaches every Kubernetes frontend's module graph, depends on
a reviewer noticing.

### KO10: The deprecation is a real marker, and its lint cost is stated

**Context**: SD1 says the API is "Deprecated". Both frontends enable staticcheck with every
check on (cli `.golangci.yml` `checks: [all, -ST1000, -ST1003]`, operator default), and SA1019
reports imports of a deprecated package.
**Decision**: Use the standard `Deprecated:` paragraph, as SD1 says. Both frontends run a
full golangci-lint with staticcheck on every PR (the cli through golangci-lint-action with no
`only-new-issues`), so the cascade's library bump PR in each frontend would turn lint red. Each
frontend's bump PR therefore carries the mechanical import swap `opm/helper/objectset` to
`opm/k8s/object` (cli `internal/workflow/render/render.go`, `validation.go` and their tests;
operator `internal/reconcile/resolution.go`, `internal/render/kernel_module_renderer.go` and
their tests). `Duplicates`, `Duplicate`, `Identity`, `Producer` and
`DuplicateIdentitiesError` keep their names and signatures, so the swap is the import line
and the package qualifier. cli-e2e5 and op-e2e5 still do the rest of the adoption. Section 5
measures the SA1019 count against both frontends as evidence.
**Rationale**: A prose-only "superseded" note would avoid the lint finding but would not be the
deprecation SD1 asks for. The bump still compiles without the swap, which is SD1's purpose; the
swap keeps the bump PR's lint green too.

### KO11: The docs say what exists

**Decision**: `README.md` (§ Helper boundary, the Layout tree and the bullet that names
`opm/helper/objectset` for Kubernetes vocabulary), `AGENTS.md` (rules line and layout) and
`CONSTITUTION.md` (Principle III and the helper list that names objectset) drop "planned" for
`opm/k8s/` and name `labels` and `object`.
They locate edits by text, because lib-c2 moved lines. Every layout and list that names
`objectset` marks it Deprecated with its replacement. `AGENTS.md` adds `k8s` to the commit scopes and one bullet under "CUE toolchain
pin": `k8s.io/apimachinery` in `go.mod` (v0.36.4) is, by MVS, every embedder's apimachinery
floor, like `cuelang.org/go`. It states no bump policy (SD24). The ADR-011 Status gains an
"Amended 2026-10-05 by add-kubernetes-object-packages" sentence covering three things:
- the first packages are `labels` and `object`;
- item 9 is carried out as copy, deprecate, then remove (SD1);
- item 2's allow-list question is answered.

The `kernel.Compiled` doc comment ("each consumer wraps *Compiled in its own resource type")
names `opm/k8s/object.Resource` as the Kubernetes wrapper. That is prose only, with no import.
`opm/helper/doc.go` describes objectset as deprecated in favour of `opm/k8s/object`.

## Risks / Trade-offs

- **Two duplicate-identity copies.** A fix to one must reach the other until removal. Mitigation:
  the helper copy is frozen (Deprecated), the ported tests are identical, and removal follows the
  frontends' adoption.
- **SA1019 in frontend lint on the bump PR** (KO10). Mitigation: each bump PR carries the
  mechanical import swap; the count is measured in section 5 and reported.
- **Go floor 1.26.0** shuts out an embedder on Go 1.25. Mitigation: the only embedders are on
  1.26. 0012:D2 accepted the apimachinery floor.
- **Weight-table contradictions with Flux** stay (KO7). Mitigation: they are documented and
  pinned, and SD11 keeps any one `ApplyAll` call within a single weight.
- **Export holds all JSON and objects at once.** The operator already holds both. The peak is
  the CUE build, not these bytes (j2/g1 research: bytes under 1 MB, the build up to GBs).

## Migration Plan

None for embedders: everything is additive. Frontends adopt in cli-e2e5 and op-e2e5 after the
first library release that contains this change. The objectset removal is a later library
change.

## Open Questions

None blocking. The supervisor's report decisions are KO1 (lint pin), KO9 (allow list) and KO10
(SA1019 on bump PRs).
