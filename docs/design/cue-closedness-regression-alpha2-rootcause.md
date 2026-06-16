# Root cause report: CUE `v0.17.0-alpha.2` comprehension-pushdown closedness regression

Status: **Root-caused, bisected, reproduced; unfixed upstream; not yet filed.**
Date: 2026-06-16. Author: investigation triggered from
[`cue-closedness-regression-alpha2.md`](./cue-closedness-regression-alpha2.md)
(the narrative/impact note). This document is the standalone technical root-cause
report and the basis for an upstream bug report against `github.com/cue-lang/cue`.

---

## 1. One-paragraph summary

`cuelang.org/go v0.17.0-alpha.2` introduced a spurious closedness error
(`field not allowed`) that rejects fields which are explicitly declared in a
closed struct. The first bad commit, found by `git bisect`, is
[`339485d`](https://github.com/cue-lang/cue/commit/339485ddf008a5b536714a5ed0fc625769a0f1a1)
— *"internal/core/adt: dependency-tracking comprehension pushdown"*. The new
`ArcPending` comprehension scheduler runs a struct's closedness check before an
*embedded conjunct's* fields have been inserted into that struct's allowed-field
set, when a *self-referential conditional comprehension* on a sibling conjunct
forces early evaluation. The regression is present on every release since
`alpha.2` and still reproduces on upstream `master` HEAD (`89492a12`,
2026-06-16). It is an upstream evaluator bug, **not** an OPM schema or catalog
defect — the same sources validate cleanly on `v0.16.0` and `v0.17.0-alpha.1`.

## 2. Environment / version matrix

Measured with isolated binaries
(`GOBIN=… go install cuelang.org/go/cmd/cue@<ver>`) and source builds of the
upstream repo:

| `cue` version                      | minimal repro (`cue vet -c=false ./...`) |
| ---------------------------------- | ---------------------------------------- |
| `v0.16.0`                          | clean                                    |
| `v0.17.0-alpha.1` (`d11a8f8b`)     | **clean** (last good)                    |
| `v0.17.0-alpha.2` (`7935fe8a`)     | **FAIL** `field not allowed`             |
| `v0.17.0-alpha.3`                  | **FAIL** `field not allowed`             |
| `master` `89492a12` (2026-06-16)   | **FAIL** `field not allowed`             |

## 3. Minimal reproduction

24 lines, single file, single package, **no registry, no cross-package imports**.
Checked in at [`repro-comprehension-closedness/`](./repro-comprehension-closedness/).

```cue
package repro

#Inner: {count: int}
#Schema: {
    a:        string
    scaling?: #Inner          // optional field, typed by a closed definition
}
#Component: {
    _allFields: {
        scaling:           #Inner
        statelessWorkload: #Schema
    }
    spec: {_allFields}        // (1) spec populated by EMBEDDING a reference
}
#SW: #Component & {
    spec: {
        statelessWorkload: #Schema
        if spec.statelessWorkload.scaling != _|_ {   // (3) self-ref CONDITIONAL
            scaling: spec.statelessWorkload.scaling
        }
    }
}
web: #SW & {
    spec: statelessWorkload: {a: "x", scaling: count: 3}   // set the optional
}
```

```bash
# last good
GOBIN=/tmp/a1 go install cuelang.org/go/cmd/cue@v0.17.0-alpha.1
( cd repro-comprehension-closedness && /tmp/a1/cue vet -c=false ./... )   # exit 0

# first bad
GOBIN=/tmp/a2 go install cuelang.org/go/cmd/cue@v0.17.0-alpha.2
( cd repro-comprehension-closedness && /tmp/a2/cue vet -c=false ./... )
#   web.spec.statelessWorkload: field not allowed:
#       ./repro.cue:11:3
#       ./repro.cue:17:3
#       ./repro.cue:24:8
```

`spec.statelessWorkload` is a declared field of the (recursively closed)
definition `#SW`, so setting it is valid — and `alpha.1` agrees. `alpha.2`+
rejects it.

## 4. The three essential ingredients (ablation)

Reduced systematically from the full catalog stack (variants `varA`–`varM`,
`min`–`min4`). Each row toggles one feature off the faithful repro and records
whether the bug still fires on alpha.2:

| Variant | Change from faithful repro                                   | alpha.1 | alpha.2 |
| ------- | ------------------------------------------------------------ | ------- | ------- |
| faithful (traits+blueprints maps, comprehension, `close`)    | —       | clean   | **FAIL** |
| varC    | drop `close()` → plain `spec: {_allFields}`                  | clean   | **FAIL** |
| varD    | drop comprehension-over-map → inline `_allFields` fields     | clean   | **FAIL** |
| varK    | conditional `if` → **unconditional** copy                    | clean   | clean   |
| varL    | `spec.`-prefixed self-path → **local** path in `if`          | clean   | **FAIL** |
| varM    | `close({_allFields})` → open `{_allFields}` (def recursion only) | clean | **FAIL** |
| min2/3  | `spec: {_allFields}` embed → **fields written directly**     | clean   | clean   |
| min4    | min2 + restore **only** the `{_allFields}` embedding         | clean   | **FAIL** |

**Necessary (bug disappears if removed):**

1. **`spec` populated by *embedding a referenced struct*** (`spec: {_allFields}`),
   not by writing the merged fields directly (min2/min3 vs min4). This is the
   ingredient earlier reductions missed; the catalog hits it through core
   `#Component`'s `spec: close({_allFields})` where `_allFields` is a hidden
   field built from a comprehension.
2. **An embedded field typed by a closed definition carrying a nested optional**
   (`statelessWorkload: #Schema`, `#Schema.scaling?: #Inner`).
3. **A self-referential *conditional* comprehension** at the same struct level
   reading a field of the struct under construction
   (`if spec.statelessWorkload.scaling != _|_ { … }`). An unconditional copy does
   not trigger it (varK) — the `if` guard is required.

**Not required (verified):** `close()` itself — recursive closedness of the
enclosing `#definition` suffices (varM); the `spec.` self-prefix — a local
reference also triggers it (varL); the comprehension over a `#traits`/`#blueprints`
map — inline fields trigger it too (varD); any cross-package or cross-module
boundary — one file in one package suffices.

## 5. Bisection

- Range: `v0.17.0-alpha.1`..`v0.17.0-alpha.2` = 117 commits.
- Test: build `./cmd/cue` at each commit, run `cue vet -c=false ./...` on the
  §3 repro. Exit 0 ⇒ good, nonzero ⇒ bad, build failure ⇒ skip (125).
- Endpoints verified: alpha.1 GOOD, alpha.2 BAD.
- Result: **`339485ddf008a5b536714a5ed0fc625769a0f1a1` is the first bad commit**
  (~7 steps, all unambiguous, no skips).

```
339485ddf008a5b536714a5ed0fc625769a0f1a1 is the first bad commit
Author: Marcel van Lohuizen <mpvl@gmail.com>
    internal/core/adt: dependency-tracking comprehension pushdown
```

## 6. Root-cause mechanism

The commit rewrites comprehension scheduling in the v3 evaluator
(`internal/core/adt`, ~400 lines changed in `comprehension.go` plus `sched.go`,
`states.go`, `unify.go`, `fields.go`, `composite.go`). From its own description:

> Switch the comprehension pushdown algorithm from field pushdown — eagerly
> materializing the body's fields on child arcs — to dependency pushdown:
> pre-create the arcs as `ArcPending` placeholders, run the comp eagerly when its
> guard fields are concrete, and let `ArcPending` propagate into the rest of the
> evaluator instead of materializing fields ahead of time.

The false positive arises from an ordering hazard between two conjuncts of the
same struct:

1. One conjunct contributes the struct's fields by **embedding a reference**
   (`spec: {_allFields}`). Under the new scheme these arrive as `ArcPending`
   arcs that are materialized lazily via `insertArc` rather than up front.
2. A sibling conjunct holds a **dependency-tracked conditional comprehension**
   (`if spec.statelessWorkload.scaling != _|_`) whose guard reads a field of the
   same struct. Evaluating that guard forces the struct toward closedness
   resolution.
3. The closedness check computes the struct's **allowed-field set from its
   currently-materialized arcs** — but the embedded conjunct's arcs from (1) are
   still pending / not yet inserted, so they are absent from the allowed-field
   set. A field that *is* declared by the embedded conjunct is therefore reported
   `field not allowed`.

This is consistent with the commit's own caveats: it lands two tests marked
`todo` "to be addressed in subsequent CLs", adds a "`processAncestors` race fix"
to "prevent a sibling comprehension from starting on a node while another comp is
still adding fields to it", and updates the golden `field not allowed` error
positions in `cue/testdata/comprehensions/closed.txtar`. The OPM catalog pattern
is an unhandled instance of exactly that race.

### Symptom mapping: catalog vs. minimal repro

| | catalog (`modules/web_app`) | minimal repro |
| --- | --- | --- |
| rejected field | `spec.statelessWorkload.scaling` (nested optional) | `spec.statelessWorkload` (outer embedded field) |
| why that level | outer `statelessWorkload` is settled first by the blueprint's own `spec.statelessWorkload` conjunct, so only the nested optional loses the race | nothing settles `statelessWorkload` before the comprehension fires, so the outer field loses |

Both are the same defect — an allowed-field set computed before an embedded
conjunct's arcs settle — surfacing at whichever level the racing comprehension
forces closedness first.

## 7. Prior art / relationship to existing issues

Same *class* as these evalv3 closedness-vs-comprehension issues, but a **new,
independently-introduced regression** from the May 2026 pushdown rewrite (not a
recurrence):

- [#3486](https://github.com/cue-lang/cue/issues/3486) — "field not allowed when
  embedding a definition inside an if comprehension" (open; labels `closedness`,
  `evaluator`, `evalv3-win`).
- [#3533](https://github.com/cue-lang/cue/issues/3533) — "field not allowed
  regression with fields under two comprehensions".

No open issue covers the `339485d` pushdown regression specifically (searched
`cue-lang/cue`, 2026-06-16).

## 8. Impact on OPM

- **Today:** none at runtime. The kernel (`library`, `opm-operator`) pins
  `cuelang.org/go v0.17.0-alpha.1`, which is clean. Only a developer running a
  newer `cue` CLI locally sees the false errors (and may misdiagnose a working
  module as broken).
- **On a toolchain bump:** bumping `cuelang.org/go` to `alpha.2`+ before this is
  fixed upstream would make the Go evaluator reject every catalog-backed module
  that uses the workload blueprints with `scaling`/`updateStrategy` set — real
  rendering breaks, not just the CLI.

## 9. Recommendations

1. **Stay on `cuelang.org/go v0.17.0-alpha.1`** in `library`/`opm-operator`
   `go.mod` until fixed upstream.
2. **Before any bump past alpha.1**, re-run §2's matrix and the operator's
   `task dev:test:local` render integration tests against the candidate version.
3. **File upstream.** Bisect is done — cite `339485d`, attach
   [`repro-comprehension-closedness/`](./repro-comprehension-closedness/)
   verbatim (no registry needed), note it still reproduces on `master`
   (`89492a12`, 2026-06-16), and cross-reference #3486.
4. For local CLI work, use `cue` at `v0.17.0-alpha.1` (matches the kernel) or
   `v0.16.0`; do not trust closedness errors from `alpha.2`/`alpha.3`.

## 10. Reproduction provenance

- Repro module: [`repro-comprehension-closedness/`](./repro-comprehension-closedness/)
  (`repro.cue` + `cue.mod/module.cue`, `language.version: v0.11.0`).
- Surfacing module: `modules/web_app/` (clean on alpha.1).
- Mis-rejected schemas: `opmodel.dev/catalogs/opm`
  `blueprints/workload/stateless_workload.cue`, `traits/scaling.cue`.
- Closing construct: `opmodel.dev/core` `component.cue` —
  `spec: close({_allFields})`.
- First bad commit:
  [`339485d`](https://github.com/cue-lang/cue/commit/339485ddf008a5b536714a5ed0fc625769a0f1a1).
