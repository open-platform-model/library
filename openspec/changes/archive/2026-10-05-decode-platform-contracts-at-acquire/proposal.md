## Why

A platform's core floor and its contract inventory are read off `Platform.Package` every time
someone asks. `Kernel.Render` looks up `#contracts.providedBy` on the shared `Package` before
staging (`opm/kernel/render.go:364-376`), and `Platform.Contracts()` decodes eleven fields of
`#contracts` on every call (`opm/platform/contracts.go:164-223`). The operator calls `Contracts()`
on every reconcile of a held platform (`internal/controller/platform_controller.go:205`), and two
controllers call it on one shared platform, so the decode repeats and every call reads a shared
`cue.Value`.

These two reads are also what keeps `Render` and `Contracts()` tied to `Package`. While they read
it, `Package` cannot be dropped from `Platform`. Dropping it is a later change, and it is where the
memory saving is.

The decision: decode the core floor and the contract inventory once, at acquire, into plain Go
fields on `Platform`; `Render` and `Contracts()` read those fields; the refusal stays before
staging. That is what makes dropping `Package` possible.

## What Changes

- `platform.NewPlatformFromValue`, and so `Kernel.AcquirePlatformFromDir`, records three facts in
  unexported fields of the new `Platform`, read from the value once:
  - the core floor: whether `#contracts.providedBy` exists;
  - the decoded `ContractInventory`;
  - or, in place of the inventory, its refusal: the typed `PlatformCoreTooOldError` for a missing
    field, or the decode error.
  Construction never fails because of contracts. An older-core platform still acquires, and the
  recorded refusal is returned later by `Contracts()` and `Render`, exactly as today.
- `Platform.Contracts()` returns the recorded inventory as a deep copy, or the recorded refusal.
  It no longer reads `Package` on a platform built by the constructor.
- New `(*Platform).CoreFloor() error`. It returns nil when the platform's `#contracts` carries
  `providedBy`. Otherwise it returns the typed `*PlatformCoreTooOldError` (field `providedBy`,
  release `schema.ProvidedBySince`). `Kernel.Render` calls it in place of the `Package` lookup, at
  the same point (after the input and context checks, before staging), with the same
  `render refused before staging: %w` wrap. `Render` then reads only `Metadata` and `Source` from
  the platform.
- A `Platform` that the constructor did not build (a struct literal, or the zero value) has no
  recorded facts. Its first `Contracts()` or `CoreFloor()` call decodes them from `Package` once,
  guarded by a `sync.Once`, and every later call reads the result. So a hand-built platform behaves
  exactly as it does today, and a zero `Platform` still returns the `#contracts`
  `PlatformCoreTooOldError` the operator's tests expect. The artifact-types "Package Is Source of
  Truth" requirement stays true: the recorded facts are a cache stamped at construction, like
  `Metadata`.
- Docs that say the inventory is "on demand" and "never at construction", or that `Render` reads
  `Package`, are corrected: the `Platform`, `Contracts`, `ContractInventory` and package docs, the
  `opm/schema` path comments, the `opm/catalog` comments that cite `Platform.Contracts` as their
  precedent, the `opm/kernel` package doc and the `Render` doc, and a dated amendment line in ADR-007.
  AGENTS.md stays as it is: its lines remain true, and the `Platform` godoc is the contract's home.

Not in this change:

- Dropping `Platform.Package`. That is a later change, which also brings the platform overlay mode and
  lets the operator delete its lease system. `Package` still pins the platform's build, so this
  change frees no memory, and the PR claims none.
- Any change to what the inventory reports or refuses. The decode moves; its rules do not.
- A frontend change. Neither cli nor opm-operator needs one: they call `Contracts()` as before,
  and their `&platform.Platform{}` tests keep their result.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `platform-artifact`: MODIFIED "Platform Type Shape" (the inventory is decoded once at
  construction, not on demand; every scenario kept); ADDED "A platform records its core floor and
  contract inventory at construction".
- `single-build-render`: MODIFIED "A render refuses a platform whose core predates the provider
  count" (the floor reads the recorded fact through `CoreFloor()` and reads no `Package`; every
  scenario kept, one added).
- `schema-dispatch`: MODIFIED "Path inventory exposed as package-level vars" (the readers of
  `Contracts` and `ContractsProvidedBy` are now the construction-time decode; every scenario
  kept).

## Impact

- Packages: `opm/platform` (`platform.go`, `contracts.go`, `doc.go` and tests), `opm/kernel`
  (`render.go`, `doc.go`, `render_core_floor_test.go`, and comments in `render_collision_test.go`),
  `opm/schema` (`paths.go` comments), `opm/catalog` (`catalog.go` and `provides.go` comments).
  Also `adr/007-shares-nothing-verbs.md`. `Taskfile.yml` moves `opm/platform` under the race
  detector in `task test`.
- Public surface: additive. The new method is `(*Platform).CoreFloor() error`. `Platform` gains
  unexported fields, including a `sync.Once`, so `go vet`'s copylocks check now flags a copy of a
  `Platform` value. No such copy exists in the library, cli or opm-operator at `origin/main` (design
  D2), and the `Platform` doc says not to copy one.
- Behaviour: `AcquirePlatformFromDir` decodes `#contracts` once. That takes about 1-2 ms, measured
  against 370-730 ms for the acquisition. `Contracts()` reads Go fields. A struct-literal or zero
  `Platform` behaves as before. A caller that mutates `Package` after construction no longer
  changes what `Contracts()` or `Render`'s floor see. The constructor docs already say not to do
  that without re-running the constructor.
- Downstream: cli (`internal/cmd/platform/check.go:125`) and opm-operator
  (`platform_controller.go:205, :271`, `transformerregistration_controller.go:227`) keep building
  and keep their behaviour. The operator's repeated `Contracts()` calls stop re-decoding.
- SemVer: MINOR (additive). Release class `feat`.
