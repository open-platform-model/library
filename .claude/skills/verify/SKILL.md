---
name: verify
description: Runtime-verify a library change through its real surface — a standalone downstream Go consumer, plus the cli/opm-operator embedders — instead of re-running the test suite.
---

# Verifying library changes at the package boundary

The library is a Go module with no binary; its surface is the public package
boundary. Verify by driving it the way an embedder does, not by re-running
`task test`.

## 1. Standalone consumer (primary handle)

Create a scratch module with a `replace` at the working tree:

```
module verifyconsumer
go 1.25.0
require github.com/open-platform-model/library v0.0.0
replace github.com/open-platform-model/library => /var/home/emil/dev/open-platform-model/library
```

Drive the full pipeline through public exports only (recipe mirrors
`opm/kernel/flow_integration_test.go`):

- `kernel.New(kernel.WithRegistry(os.Getenv("CUE_REGISTRY")))` →
  `k.AcquirePlatformFromDir(ctx, platformDir, opts)` and
  `k.AcquireInstanceFromDir(ctx, instanceDir, opts)` (import
  `opm/helper/loader/file` as `loader`, pass
  `loader.LoadOptions{Registry: os.Getenv("CUE_REGISTRY")}`); both return
  source-carrying artifacts, the only inputs `Render` accepts. The synth
  path is `k.AcquireModuleFromRegistry` → `k.SynthesizeInstance`.
- `k.Render(ctx, kernel.RenderInput{Instance, Platform, RuntimeName})` →
  `*kernel.RenderResult`: `Compiled`, `Diagnostics` (`Pairs`, `Unmatched`,
  `Unresolved`, `Unify`, `OverSubscribed`, `ResolvedVersions`), `Warnings`.
  A refusal is a `*kernel.RenderError` carrying the same diagnostics.
- Inspect `*core.Compiled` fields: `Value`, `Instance`, `Component`, `Transformer`.

Run with the canonical GHCR mapping (reads only, no local registry):

```
CUE_REGISTRY='opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works'
```

Fixtures: `testdata/modules/web_app` (module + `instance/` package, instance
name `web-app-demo`, uuid `bf5b9c54-...`), platform `modules/opm_platform` (a
CUE module importing the catalog; `#CatalogEntry` form, 0019 D5). Rendered
deployment carries `spec.replicas=2`.

Gotchas:

- `AcquireInstanceFromDir` with no option processes the package as
  authored; the web_app instance authors concrete `values:` so it is already
  complete. To probe value rejection, acquire with
  `kernel.WithValues(badSource)` where the source is
  `k.LoadSourceFromBytes("probe.cue", []byte(`replicas: "three"`))`: the
  build reports the conflict at the source's own position, framed
  `instance "…":`. Passing values that merely mirror the authored ones is a
  no-op (the overlay unifies with the package's own `values`).
- A platform whose `#registry` entry carries a `version` scalar and no
  `#catalog` (the pre-D5 shape) is refused at acquisition with
  `loader.ErrMissingRequiredField`; there is no materialize step to probe.
- API-removal changes: put references to the removed identifiers in a separate
  `removed/` package; `go build -o /dev/null ./removed/` must fail on the
  branch and compile against a `git worktree add … main` baseline (point a
  second scratch module's `replace` at the worktree).

## 2. Downstream embedders (migration-claim probe)

Build `cli` and `opm-operator` against the branch non-invasively with a
GOWORK overlay — no go.mod edits:

```
printf 'go 1.26.2\n\nuse (\n\t<repo>\n\t<library>\n)\n' > /tmp/x.work
GOWORK=/tmp/x.work go build -C <repo> ./...
```

(Match the `go` line to the highest requirement among the used modules; the
error message tells you the version it wants.)

## 3. Real CLI surface (end-to-end)

```
GOWORK=… go build -C …/cli -o opm-branch ./cmd/opm
opm-branch module vet …/cli/tests/fixtures/modules/podinfo            # green ✔ lines, exit 0
opm-branch module vet <same> -f bad-values.cue                        # exit 2, positions cited
```

The library's own `testdata/modules/web_app` is also vet-able (it carries an
`identity/` package purely for the cli's identity gate; library tests never
load it).
