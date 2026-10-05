# Tasks: add-kubernetes-health-package

Worktree `library/.claude/worktrees/add-kubernetes-health-package`, branch
`feat/add-kubernetes-health-package` (from `origin/main` `ca7c56b`). Seed `.cue-cache` by
copying the main checkout's (`cp -a`), never by symlinking. Every command runs inside the
worktree with an absolute private `TMPDIR` from `mktemp -d` under the session scratchpad. The
registry env goes on two lines:

```bash
export CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
export OPM_REGISTRY="$CUE_REGISTRY"
```

`TestGenerate_BuildsThroughTheKernel` can fail in a full-suite run because of a known
cross-process cache race. Rerun `go test ./opm/helper/platformmodule -count=1` on its own before
treating it as a finding. Commit bodies never start a line with `word(` and carry no bare
at-sign. The only trailer is `Co-Authored-By: Claude <noreply@anthropic.com>`. Source of the
ported code: cli `origin/main` `bd4d1a7c`, `internal/kubernetes/health.go` and
`health_test.go`.

## 1. opm/k8s/health (design HP1, HP2, HP3, HP5)

- [ ] 1.1 `opm/k8s/health/health.go`: `Status`, the seven constants, `Evaluate`, `IsHealthy`
      and the unexported helpers ported verbatim from the cli's `health.go`, renamed per HP1
      and with the five kind constants declared locally. Logic and status strings unchanged,
      including the PersistentVolumeClaim raw-phase passthrough and the `nolint:errcheck`
      reasons. `Aggregate` per HP2. Doc comments keep the cli's wording where it still holds;
      `Aggregate`'s says what `unhealthy` counts. Verify: `go build ./opm/k8s/...` clean.
- [ ] 1.2 `opm/k8s/health/doc.go`: the package's place in the tier (ADR-011, 0012:D3 cited
      once at the package); that it is pure and the caller fetches every object with its own
      client; that a caller must read the object after its apply, uncached, or a pre-apply read
      can report the old rollout as `Ready` (design Risks); that the status strings are
      byte-equal to the cli's output and must not change without a `feat!`; and the cli source
      commit the code was ported from.
- [ ] 1.3 `opm/k8s/health/health_test.go` (package `health`): every test of the cli's
      `health_test.go` moved with its cases and expectations unchanged and the identifiers
      renamed. The two `QuickInstanceHealth` tests become `Aggregate` tests over `Evaluate`
      results. Add `TestStatusStrings` (scenario "The status strings match the cli's output"),
      `TestAggregate` (scenarios "Missing and unreadable objects count against the aggregate"
      and "Nothing to aggregate is unknown", plus all healthy and `Applied` counted healthy) and
      the HP2 equivalence test: over the cli's fixtures, a local copy of the cli's
      `QuickInstanceHealth` loop and `Aggregate` over `Evaluate` agree for unhealthy counts 0,
      1 and 3. Check that the scenarios "A Deployment mid-rollout is not ready", "A Deployment
      past its progress deadline is not ready", "A partitioned StatefulSet is ready at its
      partition", "A pending claim reports its phase" and "A custom resource without a Ready
      condition counts as applied" each have a case; add any that is missing. Verify:
      `go test ./opm/k8s/health -count=1` green;
      `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./opm/k8s/health` lists
      only the package and `k8s.io/apimachinery` packages (and their own dependencies), no
      `opm/` package and no client.
- [ ] 1.4 Byte check, not committed: `diff` the cli's `health.go` at `bd4d1a7c` against the new
      `health.go` with the HP1 renames applied to the cli copy by `sed`. The only differences
      are the package clause, the local kind constants, `Aggregate` in place of
      `QuickInstanceHealth`, and doc-comment wording. Record nothing; fix any other difference.
- [ ] 1.5 `task check` green, then commit
      `feat(k8s): add the opm/k8s/health readiness evaluator`.

## 2. Docs, consumer builds and the API diff

- [ ] 2.1 `README.md`: add `health/` to the layout tree under `k8s/` and name
      `opm/k8s/health` (readiness evaluation over objects the frontend fetches) in the
      tier paragraph. `AGENTS.md`: add `health` to the tier list in the "Public surface"
      bullet and a `health/` line under `k8s/` in the Repository Layout (Status strings,
      Evaluate, IsHealthy, Aggregate; pure, ported from the cli). `CONSTITUTION.md`: add
      `opm/k8s/health` to the list of the tier's packages today. Keep any package a concurrent
      tier change has already added to those lines.
- [ ] 2.2 `adr/011-kubernetes-tier-beside-the-kernel.md` Status: append one sentence,
      "Amended 2026-10-05 by `add-kubernetes-health-package`: readiness evaluation arrives as
      `opm/k8s/health`, ported unchanged from the CLI, pure over objects the frontend fetches."
      Item 1's indicative list already names `health`; no item changes.
- [ ] 2.3 Consumer builds, not committed: clone cli and opm-operator `main` fresh into the
      scratch dir and run `GOTOOLCHAIN=local bash .tasks/consumer-build.sh <clone> .` for each.
      Both pass. Run `task api:diff`: it reports only additions (the new package), no
      incompatible change charged to this branch.
- [ ] 2.4 `openspec validate add-kubernetes-health-package --strict` green, `task check` green,
      then commit `docs(k8s): list opm/k8s/health in the tier docs and ADR-011`.
