# Library repository guide

The OPM kernel, the reference Go runtime for Open Platform Model, consumed as a Go library by every
frontend (`cli`, `opm-controller`, a planned Crossplane composition function). The repo ships no binary
and has no `main` package. It must never grow command, controller or output concerns: those belong to
the frontends.

## Repository rules

- `CONSTITUTION.md` is the human-readable principle source and `openspec/config.yaml` is normative.
  Their principles are not restated here.
- Mergeable sections (Principle VIII) have a hard gate. A request that cannot be cut into about five
  task sections, each ending green and closing with its own commit, gets the gate phrase from
  `openspec/config.yaml` and a proposed split. This keeps `main` releasable.
- Kernel neutrality (Principle I). The kernel is consumed by several runtimes, so it holds no global
  mutable state or package-level singleton, no `os.Exit`, no direct stdout or stderr output, no shell
  invocation, no hidden environment lookup (config arrives as arguments) and no non-determinism.
- The public surface is `opm/` only. It imports no `k8s.io/client-go`, controller-runtime or Flux,
  and `k8s.io/apimachinery` only under `opm/k8s/`. Nothing in `opm/` acts on a cluster: each frontend
  does that with its own client (ADR-008, ADR-011). Moving a function across a tier changes SemVer.
  The tiers and the allow lists are written out in the tiers reference.
- I/O lives at the edges (`internal/loader`, registry calls) and takes caller-supplied config. Logging
  is passed by the caller as a parameter or in `context.Context`.
- A statement has one home (ADR-013, decision c4). A runtime contract goes in the godoc of the package,
  type or function that owns it. Rationale goes in an ADR under `adr/`. A SHALL requirement goes in an
  `openspec/specs` capability. `README.md`, this file and `docs/getting-started.md` link to those homes
  and never restate a contract. Cite a source a reader can open: an ADR, a pull request, an archived
  change, or an enhancement by id beside the reason (enhancement 0012, the Kubernetes tier). Never cite
  a decision recorded only in an agent session.
- Edit the godoc, do not restate it. The render contract (single verb, shares-nothing context, verdicts
  as data, fail-closed gate, catalog skew, `WithRegistry`) is the `opm/kernel` package doc.
  The render steps and gate cause order are the `Kernel.Render` doc and the
  `RenderError` doc. Staging, promotion, skew and local replacements are the `opm/internal/renderstage`
  package doc. The schema cache contract is the cache godoc in `opm/schema/`.
- The old verb set (compile, match, materialize, synthesize), the raw value-loading tier and the
  free-function entry points are gone, and a test in `opm/kernel/` pins their absence. Do not bring them
  back. Do not add an `opm/validate/` package: validation is a kernel verb. Do not reintroduce the
  retired debug module definition.
- Loading a published module by path and version lives in the library, not in consumers (Principle V).
  A frontend must not hand-roll OCI fetch, wrapper-package shims, dependency walks or a directory walk
  to stage a tree: the kernel acquire verbs return a byte overlay and `module.Source` writes one out.
  The shape gate is single-sourced in `opm/internal/loader`
  and its sentinels in `opm/errors`: extend them there.
- Before pre-GA, a kernel-surface break writes no migration fragment. On the beta line it lands only as
  a `feat!` commit whose `BREAKING CHANGE:` footer is the migration note the CHANGELOG shows (ADR-010).
  From GA a break needs a migration fragment under `migrations/`, per the policy there. When a signature
  changes, check the effect on `cli/` and `opm-controller/`.
- A `retract` in `go.mod` takes effect only from the go.mod of the version the go command resolves as
  latest, which is the highest release and falls back to a prerelease only when no release exists.
  While a release exists, a retract shipped in a prerelease is ignored: fix a bad beta with the next
  beta.
- A squash-merge body line that starts with a function call makes
  release-please reject the whole commit, and a release can then find nothing to release. Prune the
  auto-filled body when you squash-merge.
- Every new exported symbol needs a doc comment, because it gets a reference entry in the docs bundle.
  Keep `ADR-NNN` and other maintainer pointers out of those doc comments.

## Four artifact types, and nothing else

| Artifact         | Schema (`opmodel.dev/core@v2`) | Go type              |
| ---------------- | ------------------------------ | -------------------- |
| `Module`         | `#Module`                      | `*module.Module`     |
| `ModuleInstance` | `#ModuleInstance`              | `*module.Instance`   |
| `Platform`       | `#Platform`                    | `*platform.Platform` |
| `Catalog`        | `#Catalog`                     | `*catalog.Catalog`   |

`Catalog` is the fourth, admitted by ADR-009 on stated terms: the kernel acquires, reads and derives, and
every verdict about what it reads stays with the caller. A catalog is never rendered. ADR-009 also states
the four-part test a fifth kind must pass. `debugValues` is a field on `Module`, and whether a frontend
layers it into the values stack is helper-layer policy.

## OPM schema versioning

The schema lives in the `opmodel.dev/core` CUE module, resolved at runtime through `CUE_REGISTRY` and
cached per Kernel in `*schema.Cache`. Versioning is per OCI module version. The default loader pins an
exact release, `opmodel.dev/core@v2.X.Y[-pre]` (`schema.DefaultSchemaModule`). The bare major
`opmodel.dev/core@v2` is opt-in and resolves to the highest published release of that major.

A consumer that wants its own pin, or the bare major, passes its own loader, for example:

```go
k := kernel.New(kernel.WithSchemaLoader(schema.OCILoader{Module: "opmodel.dev/core@v2.0.0-beta.4"}))
```

After the first schema-touching call (on a pinned kernel no verb touches the schema), the kernel's
schema cache reports what resolved.

A shape-breaking schema change is a coordinated event: `core` publishes the new shape, the Go code in
`opm/schema`, `opm/kernel` and `opm/internal/renderstage` (and the glue template) adapts, and consumers
re-pin. Within a major, an additive change needs no Go API change: artifacts re-pin core in their own
`cue.mod`, and `DefaultSchemaModule` moves separately in the cascade's `fix(deps)` PR, which re-verifies
the glue. Two compat tracks never mix: Go-module SemVer for Go types (a break is a MAJOR library bump),
and OPM schema versioning for CUE module versions (the kernel must adapt to additive changes).

## Layout

```text
opm/kernel/        PUBLIC ENTRY POINT: Kernel, acquire / synthesize / validate verbs, Render
opm/module/        *module.Module, *module.Instance and module.Source
opm/platform/      *platform.Platform, Render's sole platform input
opm/catalog/       *catalog.Catalog: the acquired #Catalog (ADR-009)
opm/schema/        core schema loader (OCILoader, Cache) and the CUE paths a consumer reads
opm/errors/        verdict rows, gate causes and fetch / resolution classification
opm/helper/        opt-in convenience for frontends; nothing outside it may import it
opm/k8s/           the Kubernetes tier beside the kernel (ADR-011); fenced by depguard
opm/internal/      loader, synth, renderstage, cueenv, modversion, registrytest and other internals
adr/               architecture decision records (the template is inside)
openspec/          OpenSpec specs, active changes and archive
modules/           test-only CUE modules (fixtures, not shipped)
testdata/          CUE fixtures for package tests, including the parity oracle and render fixtures
migrations/        migration policy and fragments (dormant through beta until GA, ADR-004)
docs/              getting-started walkthrough, CUE evaluator notes (design/), site pages (site/)
docs-kit.cue       the docs bundle config
.cue-cache/        gitignored shared CUE module cache; tests never delete from it
```

## Environment

Set `CUE_REGISTRY` and `OPM_REGISTRY` as the workspace registry reference says. Nothing needs a local
registry: `opmodel.dev/*` reads resolve from GHCR, and the Go suite runs against it. Only
`task cue:publish` and `task cue:publish:smart` use `localhost:5000`, and they run only when the user
asks. New tests use the in-process registry in `opm/internal/registrytest`; tests in
`opm/kernel/` show the pattern.

## Commands

| Command | What it does | When |
| --- | --- | --- |
| `task fmt` / `task vet` / `task lint` / `task test` | gofmt + goimports, go vet, golangci-lint, go test | while working |
| `task check:fast` | the four above without lint | iterative work |
| `task check` | the four, then `docs:bundle:check`, `cascade:wiring:check`, `cite-lint`, `api:diff:test` | before merge |
| `task test:run TEST=TestName` | one Go test | debugging |
| `task cite-lint` | the enhancement citation rule for Go and CUE comments | after editing comments |
| `task api:diff` | incompatible changes to the exported `opm/` API since the last release tag (`BASE=<tag>` to pick one) | before a kernel-surface change |
| `task cue:check` | cue fmt and vet across `modules/` and `testdata/` (auto-discovered through `CUE_MODULE_GLOBS`) | after a CUE fixture edit |
| `task cue:test:flow` | acquire-to-render test against the published catalog (`OPM_FLOW_TEST_FORCE=1` to require it) | before a render change |
| `task -x deps:cascade` | moves the upstream pins; always pass `-x`, since plain `task` turns exit 3 into 201 | when the cascade runs |
| `task docs:bundle` | builds the library docs bundle into `out/library/` | before a docs change |
| `task deps:release-check` | release-pin gate; CI runs it on `release-please--` branches | on a release PR |

The cascade task also rewrites the core release in the `OCILoader` `Module` example of this file and of
`docs/getting-started.md`; keep that example above.

## Style

- Standard Go import grouping (stdlib, external including `cuelang.org/go`, then
  `github.com/open-platform-model/library/...`); `gofmt` and `goimports` handle it.
- Accept interfaces, return concrete structs. Pass `context.Context` through I/O and CUE evaluation.
  Wrap errors (`fmt.Errorf("loading module: %w", err)`) and reuse the `opm/errors` types.
- Commits are Conventional Commits v1 with a package scope (`core`, `loader`, `module`, `kernel`,
  `errors`, `schema`, `platform`, `helper`, `render`, `renderstage`, `k8s`). `chore`, `test`, `ci`,
  `build` and `docs` never release. Go and CUE deps the kernel embeds bump as `deps` or `fix(deps)`;
  test fixture pins bump as `test(fixtures)`.
- A comment cites an enhancement only once, at the symbol, as the id beside a reason in words, such as
  `(enhancement 0011)`. Never cite a decision number or a section, slice or task number.

@CONTEXT.md
