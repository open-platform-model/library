## Why

ADR-011 placed the Kubernetes decisions OPM makes in a tier beside the kernel, `opm/k8s/`, and
fenced it in lint before any package existed. This change lands the first two packages. Both
come from the beta-1 kernel-plan walkthrough (owner, 2026-10-02/03), where they are tasks e2
and e5:

- **e2.** The cli and the operator each carry a `pkg/core` package. Both copies are
  byte-identical at their `origin/main` (cli `0b37e3f2`, re-checked unchanged at `1338e700`;
  opm-operator `9b83611`): a `Resource`
  that wraps a rendered `cue.Value` with its instance, component and transformer provenance,
  `MarshalJSON` and `ToUnstructured` on it, and the OPM label vocabulary with `IsOPMManagedBy`.
  The operator also carries `ResourceFromCompiled`, and the cli copies the same fields inline
  in its render workflow. The owner decided: "Move cli/opm-operator pkg/core into library
  opm/k8s/labels (no deps) and opm/k8s/object (apimachinery) with a Resource wrapper type".
  The later g1 decision is: "e2 ships alone; opm/k8s/object.Resource wraps the Value for now"
  (the JSON-bytes switch, g1, is deferred).
- **e5.** The kind-class weight table exists only in `cli/pkg/resourceorder`. Since cli#289
  (walkthrough task a1, `75e5f268`) the cli applies by it. The operator applies through Flux's
  own staging and has no copy. The owner decided: "Weight table moves from
  cli/pkg/resourceorder to opm/k8s/object with e2". This is the library half. The frontends
  delete their copies in their own adoption changes.
- **The objectset move.** ADR-011 item 9 and the helper doc say that `opm/helper/objectset`
  (duplicate apply identities) moves into `opm/k8s/object` with the first tier package. Both
  frontends import it at `origin/main`. Supervisor decision SD1 (deprecate, then remove) keeps
  the helper package, marks it Deprecated and points it at `opm/k8s/object`. Its removal is a
  later change that merges after both frontends have migrated, so a deps-cascade bump of either
  frontend keeps compiling.

The wave-1 operator memory work adds one requirement. The operator exports each rendered
resource from CUE once inside its render slot, feeds both its digest and its apply objects from
those bytes, and then drops the CUE values (`internal/reconcile/converted.go`, `convertRender`).
The cli exports twice. The shared conversion must offer the single export, or the operator
regresses when it adopts the package. Follow-up x6 adds that the operator must be able to
derive inventory entries from that one converted set.

## What Changes

- **`opm/k8s/labels`** (new, standard library only): the label keys and values both frontends
  use, with `IsOPMManagedBy`. Every constant in the frontends' `pkg/core/labels.go` has a
  consumer at `origin/main`, so all of them move. Names drop the package stutter (design KO2).
  The package names and recognises labels and never stamps them: stamping stays in CUE at render
  (0012:D6). A depguard rule holds the package to the standard library.
- **`opm/k8s/object`** (new, on `k8s.io/apimachinery`):
  - `Resource` is the frontends' wrapper, field for field (`Value cue.Value` plus `Instance`,
    `Component` and `Transformer`). It keeps the accessors (`Kind`, `Name`, `Namespace`,
    `APIVersion`, `GVK`, `Labels`, `Annotations`, `String`) and the conversions (`MarshalJSON`,
    `ToUnstructured`). `labels.Component` keeps its key's name, and its doc says it is the OPM
    object category, not a component name (design KO2). `NewResource` and `Resources` build it from `*kernel.Compiled`, which
    replaces `ResourceFromCompiled` and the cli's inline copy.
  - `Export` exports each resource from CUE exactly once. It returns, index-aligned with its
    input, the JSON bytes, the decoded `*unstructured.Unstructured` and the provenance. A caller
    can then compute a digest, build apply objects and derive inventory entries from one export
    and drop the CUE values straight after. `*ExportError` names the failing resource and
    whether the CUE export or the JSON decode failed; a value that exports to JSON that is not
    an object (a list, a string or `null`) is a decode failure.
  - The kind-class weight table (`Weight` plus the `Weight*` constants), `Sort` and `Direction`
    are ported unchanged from `cli/pkg/resourceorder` at cli `origin/main` `0b37e3f2`.
    `GetWeight` becomes `Weight`.
  - `Stages` cuts an apply set into the stages the order implies: the cluster definitions
    (CustomResourceDefinitions and core Namespaces) first, then one stage per distinct weight,
    ascending. This is the input SD11 needs ("apply one library weight group per fluxssa
    ApplyAll"). It is the same definition the cli's two-stage apply uses.
  - `Duplicates`, `Duplicate`, `Identity`, `Producer` and `DuplicateIdentitiesError` are copied
    from `opm/helper/objectset` with their behaviour and wording unchanged.
  - A documentation-only test lists Flux's `ReconcileOrder` kinds (fluxcd/pkg/ssa v0.77.0, the
    operator's pin) as literals, with no Flux import. Over those kinds, the library's table
    kinds and one custom kind (unlisted kinds rank 0 in Flux), it records every pair the library
    orders strictly the other way, so a table change that adds or removes a contradiction shows up in review.
- **`opm/helper/objectset`** stays and is frozen. Its package doc and every exported symbol
  carry `Deprecated:` with the replacement in `opm/k8s/object` (SD1). Its behaviour does not
  change.
- **`go.mod`**: `go 1.26.0` and `k8s.io/apimachinery v0.36.4`, matching both frontends (SD10).
  If the CI lint pin cannot lint a Go 1.26 module, it moves to the release the cli pins (design
  KO1).
- **Lint**: one rule holds `opm/k8s/labels` to the standard library. ADR-011 item 2 left one
  question to this change, whether a strict allow list for the tier pays. This change answers
  yes (design KO9): a second rule allows only the standard library, `cuelang.org/go/cue`,
  `k8s.io/apimachinery` and the library's own `opm/` packages in non-test tier files. The
  existing denials stay.
- **Docs**: the "planned" `opm/k8s/` lines in `README.md`, `AGENTS.md` and `CONSTITUTION.md`
  become the real layout. The helper doc and the `AGENTS.md` layout mark `objectset` as
  Deprecated. The ADR-011 Status records the first packages, the copy-then-remove reading of
  item 9 and the allow-list answer. `AGENTS.md` gains the `k8s` commit scope and a note that
  `k8s.io/apimachinery` in `go.mod` sets every embedder's floor, as `cuelang.org/go` does.
- **Specs**: `kubernetes-tier` gains the label, conversion, order, stage and allow-list
  requirements. `duplicate-object-identities` moves its home to `opm/k8s/object` and states
  the deprecated copy. `helper-packages` marks `objectset` as Deprecated.

Not **BREAKING** (SD1). Every addition is new API. Nothing is removed or renamed. Two things are
observable. The Go floor rises from 1.25.0 to 1.26.0, a floor both frontends already declare.
Importers of `opm/helper/objectset` (cli and opm-operator) now see a deprecation, which
staticcheck's SA1019 reports in their lint. Both frontends run a full golangci-lint with
staticcheck on every PR, so each frontend's cascade bump PR carries the mechanical import swap
`opm/helper/objectset` to `opm/k8s/object` (cli `internal/workflow/render/render.go`,
`validation.go` and their tests; operator `internal/reconcile/resolution.go`,
`internal/render/kernel_module_renderer.go` and their tests). `Duplicates`, `Duplicate`,
`Identity`, `Producer` and `DuplicateIdentitiesError` keep identical names and signatures, so
the swap is a few lines and keeps the bump's lint green. Without it the bump still compiles.
cli-e2e5 and op-e2e5 do the rest of the adoption.

SemVer class: MINOR. Release class of the PR: `feat`.

## Not in this change

- **Frontend adoption.** That is op-e2e5 and cli-e2e5: deleting `pkg/core`, `pkg/resourceorder`
  and `compiled_adapter.go`, switching importers, per-weight-group `ApplyAll`, and depguard
  rules against the old paths.
- **Removing `opm/helper/objectset`.** That is its own change, after both frontends have
  migrated (SD1).
- **The render digest and inventory entries** (lib-e3, `opm/k8s/inventory`), **ownership
  verdicts and the adopt annotation key** (lib-e4), **health** (lib-f5) and **the deletion
  protocol** (lib-f2). `Export` exists so that those packages and the frontends need one export,
  but this change computes no digest.
- **The g1 switch of `Compiled.Value` to JSON bytes.** It is deferred by the owner. `Resource`
  wraps the value. Because `Export` already hands out bytes, the later switch can stay internal
  to `opm/k8s/object`.
- **Changing any weight.** The table is ported as it is (owner e5). The Flux comparison
  documents the differences and decides nothing.
- **A dependabot or release rule that moves `k8s.io/apimachinery` only through the library.**
  SD24 records no owner decision on it. This change only states the MVS floor fact in
  `AGENTS.md`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `kubernetes-tier`: the first packages. A dependency-free label vocabulary that never stamps,
  one-export conversion of compiled objects, the weight table and apply stages, and the tier's
  import allow list.
- `duplicate-object-identities`: the check lives in `opm/k8s/object`. The helper copy is
  deprecated, frozen and kept until both frontends migrate.
- `helper-packages`: `objectset` is listed as deprecated, with its landing place moved to
  `opm/k8s/object`.

## Impact

- Packages: new `opm/k8s/labels` and `opm/k8s/object`. `opm/helper/objectset` gets doc comments
  only. `opm/helper/doc.go` and the `kernel.Compiled` doc comment get prose only. A
  `//nolint:staticcheck` with its reason goes on each line staticcheck reports in the two kernel
  tests that use the deprecated package (`opm/kernel/flow_integration_test.go` and
  `opm/kernel/render_test.go`, the import and the `Duplicates` call in each; the kernel tests
  cannot import the tier). The helper's own external test package needs none.
- Repo files: `go.mod`, `go.sum`, `.golangci.yml`, `.cascade-frozen` (an entry for the ported
  duplicates test), possibly `.github/workflows/lint.yml` (lint
  pin), `README.md`, `AGENTS.md`, `CONSTITUTION.md`, `adr/011-kubernetes-tier-beside-the-kernel.md`.
- Downstream: cli and opm-operator compile unchanged against this tree. The consumer-build job
  must stay green. Their adoption changes (cli-e2e5, op-e2e5) gate on the first library release
  that contains this change (SD2). The operator applies one library weight group per
  `ApplyAll` (SD11) using `Stages`.
- Ordering (wave-2 serialization): this change follows lib-consolidate and lib-c2 (both merged).
  lib-d1d3 rebases on its `go.mod`. lib-e3, lib-e4 and lib-f5 build on these packages, and
  lib-e4 adds the adopt annotation key to `opm/k8s/labels`.
- `enhancement.yaml` declares 0012 with no decision claimed. 0012:D3, 0012:D5 and 0012:D6 are
  delivered only when the frontends adopt (ADR-011 item 3), so the claim belongs to those
  changes. Under-claiming is the safe direction.
