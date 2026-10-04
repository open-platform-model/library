Depends on: nothing unmerged. Lands before wave-2 `lib-e2e5` and `lib-c4` (proposal.md, Impact). Line numbers are at `origin/main` 58f8151; re-check them before editing. Every section's gate: `TMPDIR=$(mktemp -d) task check` (which includes `task docs:bundle:check`) and `openspec validate fix-front-door-docs --strict` pass.

## 1. Four artifact types and their schema module

- [ ] 1.1 `README.md:7`: the owns bullet lists "modules, module instances, platforms, catalogs". `README.md:19`: the kernel accepts only `Module`, `ModuleInstance`, `Platform` and `Catalog`.
- [ ] 1.2 `README.md:25-27`: the column header reads "Schema definition (`opmodel.dev/core@v2`)", the `Module` row drops "(v1alpha2)", and the table stays aligned (design.md D1).
- [ ] 1.3 `AGENTS.md:163`: the header reads "Schema (`opmodel.dev/core@v2`)", table aligned (design.md D1).
- [ ] 1.4 Verify: `grep -n 'v1alpha2' README.md AGENTS.md` prints nothing, and no line of `README.md` names three of the kinds as the whole set the kernel accepts or acquires.
- [ ] 1.5 `TMPDIR=$(mktemp -d) task check` and `openspec validate fix-front-door-docs --strict` green, then commit `docs: name four artifact types and the core module in the front door`.

## 2. Pinned by default, bare major opt-in

- [ ] 2.1 `README.md:97`: the D2 text in design.md.
- [ ] 2.2 `AGENTS.md:351`, `:353` and `:361`: the D2 texts in design.md. The code example at `:356` is unchanged.
- [ ] 2.3 Verify: `grep -n -i 'floating' README.md AGENTS.md` prints nothing; `grep -nE 'opmodel\.dev/core@v[0-9]+\.[0-9]+\.[0-9]+' README.md AGENTS.md` lists only the `AGENTS.md` loader example, so the cascade's prose warning set is unchanged.
- [ ] 2.4 `TMPDIR=$(mktemp -d) task check` and `openspec validate fix-front-door-docs --strict` green, then commit `docs: say the core schema is pinned by default`.

## 3. Package map, constitution and site page

- [ ] 3.1 `README.md:40`: delete the `core/` layout row; the `kernel/` row ends "and `Compiled`, its terminal output" (design.md D3).
- [ ] 3.2 `CONSTITUTION.md:64`, `:65` and the pipeline block at `:78-80`: the D4 texts in design.md (acquire, synthesize, then `Render` over an instance and a platform; no `Catalog` into render). Leave the `opm/k8s/` bullet at `:70` alone.
- [ ] 3.3 `README.md:144`: "small batches" becomes "mergeable sections".
- [ ] 3.4 `docs/site/diagnostics/colliding-contracts.md:35`: `core/src/platform_contracts_pins.cue` becomes `core/src/pins/platform_contracts_pins.cue`.
- [ ] 3.5 Verify: every directory the `README.md` layout block lists under `opm/` exists (`ls -d opm/<dir>`); `grep -n 'load, process\|release model\|schema-validate' CONSTITUTION.md` prints nothing; `grep -rn 'core/src/[a-z_]*_pins\.cue' docs README.md AGENTS.md` lists only `core/src/pins/` paths.
- [ ] 3.6 `TMPDIR=$(mktemp -d) task check` and `openspec validate fix-front-door-docs --strict` green, then commit `docs: fix the package map, constitution pipeline and pins path`.

## 4. Verify and archive

- [ ] 4.1 The repo's verify skill (`openspec verify`) reports no CRITICAL finding.
- [ ] 4.2 `openspec archive fix-front-door-docs -y`, then `openspec validate --all --strict` passes. The archive rides the PR.
- [ ] 4.3 Commit `chore(openspec): archive fix-front-door-docs`.
