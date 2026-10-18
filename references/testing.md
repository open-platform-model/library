# Testing: caches and contracts

Covers how the test tree caches CUE modules, how tests serve registries in process, and the two
contract-test rules for the schema cache and the render path. The contracts themselves are godoc.

## Test module cache: two tiers

`go test ./...` runs every package as its own process, and CUE's module cache assumes an extracted directory is immutable while any process can read it (readers hold no lock). The test tree therefore keeps two cache tiers and never deletes from the shared one:

- **Shared:** `.cue-cache/` (gitignored) holds the `opmodel.dev` namespace — `opmodel.dev/core@v2` and the GHCR catalogs — for every test and test process. A cold checkout fetches core once through CUE's own lock-protected fetch path; nothing removes entries from it. Tests that need only `opmodel.dev` (schema cache, file loader, synth unit, flow test) use the helpers in `opm/internal/schematest/` and build here directly.
- **Private:** every `registrytest` constructor points `CUE_CACHE_DIR` at the private-cache helper in `opm/internal/schematest/`, a temp cache whose `mod/extract/opmodel.dev` and `mod/download/opmodel.dev` are symlinks into the shared tier. Served fixture prefixes (`test.example`, `testing.opmodel.dev/...`) extract into it fresh, so a committed fixture edited under a fixed version is always built from its current bytes, two packages serving the same coordinate never touch the same directory, and the cache is removed at test end (read-only extracted directories included).
- **Isolated:** The `registrytest` stand-in-core constructor serves a stand-in `opmodel.dev/core` (a core lacking a definition, for the synth-build failure test) and so uses the isolated-cache helper in `opm/internal/schematest/`, a temp cache with no link into the shared tier: the stand-in never reaches `.cue-cache`.

If `.cue-cache` ever holds `test.example` or `testing.opmodel.dev` entries, they predate this layout and can be deleted by hand once; the suite no longer writes them there.

## Schema cache lifetime contract

The schema cache contract is godoc: the cache godoc in `opm/schema/` (one memoized load per cache into a
private context, the on-disk cache two caches share, no package-level singleton),
the kernel constructor godoc (no load at construction, and which calls load) and the kernel's
schema-cache accessor (one cache per Kernel for its lifetime). Edit the godoc; do not restate it here. Facts the godoc does not carry:

- On a pinned kernel (the default), the only calls that load the schema are the
  consumers' own: the cli publish gate and the operator's startup smoke check call
  the schema cache.
- Frontends set `CUE_REGISTRY` (the library applies no default, `opm/schema` package
  doc); tests use the workspace-local cache through `opm/internal/schematest` (see
  "Test module cache: two tiers").

## Render contract tests

The render contract is the `opm/kernel` package doc (`go doc ./opm/kernel`): the single render verb, every operation building in a context of
its own, verdicts as data, the fail-closed gate, catalog skew, the registry
mapping and the local-replacements opt-in are specified there. Edit the godoc; do not
restate it here. Two rules are for this repo's tests only:

- Tests serve catalogs and modules from the in-process registry
  (`opm/internal/registrytest`) while resolving `opmodel.dev/core@v2` from the warm
  workspace cache; the production stage → build → decode path runs unchanged.
- The parity harness (the parity tests in `opm/kernel/` against `testdata/parity/`, the
  pure-CUE oracle) is the oracle for rendered values, and `task test` runs `opm/kernel`
  and `opm/internal/renderstage` under `-race` to keep the shares-nothing claim checked.

## Check before you finish

- [ ] No test deletes from `.cue-cache`.
- [ ] A new registry-backed test uses `opm/internal/registrytest`, not `localhost:5000`.
