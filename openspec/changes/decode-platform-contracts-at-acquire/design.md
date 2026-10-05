## Context

Line numbers are at `origin/main` `b2d51d7` (library v1.0.0-beta.6).

**Where the platform is read today.**

| Reader | Where | What it reads |
| --- | --- | --- |
| `Kernel.Render` core floor | `opm/kernel/render.go:364-376` | `Package.LookupPath(schema.ContractsProvidedBy).Exists()`, after the input checks and the `ctx.Err()` check (`:361`), before `os.MkdirTemp` (`:378`) |
| `Platform.Contracts()` | `opm/platform/contracts.go:164-223` | `#contracts` decoded on every call: nine required fields with their since-releases, then the two optional collision fields |
| `NewPlatformFromValue` | `opm/platform/platform.go:63-72` | `metadata` and `type` only |

`AcquirePlatformFromDir` (`opm/kernel/acquire.go:274-290`) is the only acquire path. It calls
`NewPlatformFromValue` and then stamps `Source`.

**Callers of `Contracts()`** at `origin/main`: cli `internal/cmd/platform/check.go:125`;
opm-operator `internal/controller/platform_controller.go:205` (the skip path, on every reconcile
of a held platform) and `:271`, and `internal/controller/transformerregistration_controller.go:227`.

**Platforms not built by the constructor.** The operator's tests build `&platform.Platform{}`:
`internal/controller/platform_inventory_test.go:367` calls `Contracts()` on it and expects a
`*PlatformCoreTooOldError` with field `#contracts`, and `store_test.go`,
`platform_controller_test.go:71` and `test/integration/reconcile/platform_gate_test.go:165` store
one. The library's own `opm/kernel/render_core_floor_test.go:86` builds
`&platform.Platform{Metadata: md, Package: acquired.Package, Source: acquired.Source}` and asserts
that `Render` and `Contracts()` refuse it with the same error. `render_collision_test.go:212` builds
a platform through `NewPlatformFromValue` whose `#contracts` fails to evaluate (`conflicting
values`), and expects construction and the render floor to pass so that the build fails.

**Cost.** Measured with memprobe at `origin/main` during planning: `AcquirePlatformFromDir` takes
370-730 ms, and it already evaluates `#contracts`. A following `Contracts()` takes 0-2 ms on both
the two-catalog and the opm-only platform. Decoding at construction therefore adds about 1-2 ms
to an acquisition, and the cli, which acquires a platform on every render, pays nothing
noticeable.

## Goals / Non-Goals

**Goals:**

- The core floor and the contract inventory (or its refusal) are decoded once, when the platform
  is constructed, into plain Go fields (owner decision h4).
- `Render` and `Contracts()` read those fields. `Render`'s floor refusal stays where it is: before
  staging, with the same typed error and wrap.
- After this change, `Render` and `Contracts()` on a constructed platform never read `Package`, so
  g5 part B can drop it.
- No behaviour change for any existing caller, including struct-literal and zero platforms.

**Non-Goals:**

- Dropping `Package`, and any memory claim. That is g5 part B.
- Changing what the inventory reports or refuses.
- Any cli or opm-operator change.

## Decisions

### D1. Record three facts at construction, never fail on them

`Platform` gains unexported fields:

```go
type Platform struct {
	Metadata *PlatformMetadata `json:"metadata"`
	Package  cue.Value         `json:"-"`
	Source   *Source           `json:"-"`

	// recorded is the construction-time decode of the core floor and the
	// contract inventory. once guards it for a Platform the constructor did
	// not build (D2).
	once     sync.Once
	recorded facts
}

// facts is what decode reads off Package, once.
type facts struct {
	providedBy bool               // #contracts.providedBy exists: the core floor
	inv        *ContractInventory // nil when refused
	tooOld     *tooOld            // a missing #contracts or report field (D3)
	err        error              // #contracts did not evaluate, or a field failed to decode
}
```

`decode(v cue.Value) facts` is the current `Contracts()` body (`contracts.go:165-222`). It
records the outcome in place of returning it, and it also records
`v.LookupPath(schema.ContractsProvidedBy).Exists()`. `NewPlatformFromValue` decodes the metadata
first, as today. A metadata failure still returns a nil `*Platform`. It then runs
`p.once.Do(func() { p.recorded = decode(v) })`. A contracts refusal or decode error is stored and
never returned from construction, so an older-core platform acquires
(`render_core_floor_test.go:67-68`), and so does the hand-built colliding platform of
`render_collision_test.go:212`.

The floor and the inventory are recorded separately on purpose. The floor is a presence test that
stays true for a platform whose `#contracts` fails to evaluate. That is the
`render_collision_test.go` case: the floor passes and the build fails on the conflict. If
`Render` read the inventory refusal instead, that test's error would change.

### D2. A Platform the constructor did not build decodes lazily, once

This change is additive. `Platform` has exported fields, and both the library and
the operator build it without the constructor. So when `recorded` was never set, the first
`Contracts()` or `CoreFloor()` call runs the same `once.Do` against the `Package` it finds, and
every later call reads the result. Concretely:

- A struct literal over a real platform value behaves exactly as today. That includes the
  `render_core_floor_test.go:86` literal with `nil` or empty metadata, which still names the
  platform `<unnamed>` in both refusals.
- A zero `Platform` reads a zero `cue.Value`, finds no `#contracts`, and returns the
  `PlatformCoreTooOldError{Field: "#contracts"}` the operator's test expects. Its floor is unmet,
  and `Render` refuses it before that anyway, because it has no `Source`.

The `Platform` doc keeps "Package is the source of truth": the recorded facts are a cache stamped
at construction, like `Metadata`. The doc says that a caller that changes `Package` re-runs the
constructor. That keeps the artifact-types "Package Is Source of Truth" requirement true, so the
requirement needs no delta.

**Copying.** A `sync.Once` makes `Platform` a type that must not be copied after first use, and
`go vet` (copylocks) flags a value copy. At `origin/main`, the library, cli and opm-operator hold
`*platform.Platform` everywhere and never copy the value. Planning searched for dereference copies
and value-typed `platform.Platform` uses, and found only the `reflect.TypeOf(platform.Platform{})`
in the field-set test. The `Platform` doc gains one sentence: use it through a pointer, never copy
it. Task 3.2's consumer build runs `go vet` on both frontends, so a copy there would show.

### D3. Contracts() hands out copies; refusals are built per call

`Contracts()` returns a deep copy of the recorded inventory: every map and slice is cloned, and so
is each `Comparable` row's `Contracts` slice. The clone keeps nil and empty apart (`maps.Clone` and
`slices.Clone` do), so `reflect.DeepEqual` and JSON (`null` against `[]`) see what the decode produced. Callers on several goroutines (the operator's two
controllers) therefore never share a mutable value, and a caller that edits its result cannot
change the next call's result. A missing-field refusal is recorded as data (`tooOld{field, since}`)
and built into a fresh `*oerrors.PlatformCoreTooOldError` on each call, naming the platform with
`p.name()` exactly as today. A decode error is an immutable `fmt` error and is returned as
recorded.

### D4. CoreFloor() is the floor's one reader; Render reads no Package

```go
// CoreFloor reports whether the platform's #contracts carries providedBy,
// the provider count Kernel.Render's glue reads. It returns nil when it
// does, and otherwise the *oerrors.PlatformCoreTooOldError (Field
// "providedBy", Since and Require schema.ProvidedBySince) that Kernel.Render
// refuses the platform with before staging.
func (p *Platform) CoreFloor() error
```

`render.go:364-376` becomes:

```go
if err := in.Platform.CoreFloor(); err != nil {
	return none, nil, fmt.Errorf("render refused before staging: %w", err)
}
```

The position is unchanged, so `TestRender_OlderCorePlatformRefusedBeforeStaging` still sees no
staging directory. The error is the one `Render` builds today. Its `Platform` field comes from
`p.name()`, the same raw `metadata.name` that `platformMetadataName` returns, so the
"Render and Contracts() refuse alike" test (`render_core_floor_test.go:79-104`) holds unchanged. If
no other reader of `platformMetadataName` is left, the helper is deleted.

`kernel` and `platform` are separate packages, so the floor needs an exported reader. Two other
options were considered:

- `Render` calls `Contracts()` and reads its refusal. Rejected, because the semantics differ: the
  inventory also refuses on other missing fields and on a `#contracts` that does not evaluate (D1),
  and the floor must not.
- An internal bridge package that the `platform` package fills in `init`. Rejected: it adds hidden
  coupling to avoid one documented method. The frontends can also use `CoreFloor()`, for example
  the cli's platform check.

`CoreFloor` is the one new exported symbol. It has a doc comment (AGENTS.md: every exported symbol
needs one), and `task api:diff` should list it as a compatible addition.

### D5. Docs move with the code they describe

Each section corrects the statements its code makes false, so every commit leaves `main`
consistent:

- Section 1 (`opm/platform`): the `Platform` doc (`platform.go:12-21`), the `Contracts` and
  `ContractInventory` docs (`contracts.go:133-163`, and the `defined` paragraph, which still
  points at `Package`), the package doc (`platform/doc.go:13-16`), the `schema.Contracts` path
  comment (`paths.go:52-58`, "never platform construction"), and the two `opm/catalog` comments
  that cite `Platform.Contracts` as their on-demand precedent (`catalog.go:33-36`,
  `provides.go:36-39`). Those comments keep their own rule and drop the precedent claim.
- Section 2 (`opm/kernel`): the `Render` doc (`render.go:324-331`), the floor comment, the
  `opm/kernel` package doc (`doc.go:55-61`, "with one exception: Render reads ... Package"; the
  Goroutine safety sentence at `:80-86`, "reads the shared Package only for the core floor", which
  becomes "reads no Package: the floor reads the fact recorded at construction"; and `:185-189`), the `schema.ContractsProvidedBy` comment (`paths.go:60-66`), the race-test comments
  (`render_core_floor_test.go:106-110` and the cold sibling), ADR-007's Status paragraph (a new
  dated amendment sentence naming this change and owner decision h4, in the form of the 2026-09-30
  sentence, which stays as history), and the `acquireOlderCorePlatform` helper comment and require
  message (`render_core_floor_test.go:54-55, :68`), which say acquisition does not read the
  inventory.

AGENTS.md is not edited. Its `platform/` line ("Render's sole platform input") and its parity
tripwire line stay true, and AGENTS.md "Where a statement lives" puts a runtime contract in one
home, here the `Platform` godoc.

## Research & Decisions

### Where to decode

**Context**: Owner decision h4 says "at acquire". `AcquirePlatformFromDir` is the only acquire
verb, and it builds the platform through `NewPlatformFromValue`.
**Explored**: decoding in `AcquirePlatformFromDir` after construction, or in the constructor.
**Decision**: in the constructor.
**Rationale**: every constructed platform then carries the facts, including one built from a bare
value (`render_collision_test.go:212`, cli tests), and the kernel needs no write access to
unexported fields.

### Keeping a hand-built platform working

**Context**: recording only at construction would break a
struct-literal `Platform` (`render_core_floor_test.go:86`). That would be a behaviour break filed
as `feat`.
**Explored**: (a) `feat!` with a migration note, and a REMOVED+ADDED of the artifact-types
requirement; (b) a lazy `sync.Once` fallback; (c) a lazy fallback with no caching, which decodes
`Package` on every call for a literal, as today.
**Decision**: (b).
**Rationale**: it keeps the change additive, keeps "Package Is Source of Truth" true, and
decodes a literal once rather than per call. The one `Once` also covers the constructor path, so
`Contracts()` and `CoreFloor()` have one code path. The cost is the no-copy rule in D2, and no
caller breaks it.

### Spec delta shape

**Context**: OpenSpec 1.12 refuses a MODIFIED requirement that drops a main-spec scenario. The
repo rule is REMOVED+ADDED under a new name where a requirement or scenario name has gone stale.
**Decision**: MODIFIED for "Platform Type Shape", for the render core-floor requirement and for
the schema-dispatch path inventory. No requirement or scenario name becomes false: only body
sentences do ("SHALL NOT be decoded at construction", "readable on demand", "SHALL only read the
platform's `Package`"). Every scenario is kept by name. The new construction-time behaviour is an
ADDED requirement in `platform-artifact`.

## Risks / Trade-offs

- [Acquisition does a little more work] → about 1-2 ms on 370-730 ms. A `#contracts` that fails
  to evaluate is stored and surfaced by `Contracts()`, never thrown at acquire, so the
  older-core and hand-built-colliding tests keep passing.
- [No memory saving yet] → `Package` stays, so the build stays pinned (about 96 MiB idle for the
  two-catalog platform). Only g5 part B frees it. The PR says so.
- [A mutated `Package` is no longer seen] → the constructor docs already say to re-run the
  constructor after changing `Package`. No caller in the library, cli or opm-operator mutates it.
- [Copying a `Platform`] → D2: no copy exists today, and vet's copylocks now flags one.
- [Shared mutable inventory] → D3: every `Contracts()` call returns a deep copy.

## Verification

Filled in by the implement stage (tasks 3.1-3.3).
