## Context

This is a prose-only change to four files. The facts it writes were checked at library `origin/main` 58f8151 and core `origin/main` (core PR 103, 606498b, moved the pins into `src/pins/`). Line numbers below are at 58f8151.

## Goals / Non-Goals

**Goals:**

- Every statement the proposal lists says what the code does today.
- Each fix states the fact once in the reader's words, with no new claim the code does not back.

**Non-Goals:**

- Any change under `opm/`, any test or task.
- The lines other wave-2 changes own (proposal, "Not in this change").
- Rewriting sections that are correct. Only the lines named here change, plus the clause around them when the sentence would otherwise not read.

## Research & Decisions

### D1. The schema label is the core module, not a version

**Context**: `README.md:27` and `AGENTS.md:163` label the artifact schemas `v1alpha2`. The plan asked for a fact-checked label before writing one.

**Explored**: core `src/module.cue:10-11`, `src/module_instance.cue:13-14`, `src/platform.cue:158-159` and `src/catalog.cue:82-83` define each root with `kind` and no `apiVersion`. The library main spec `schema-dispatch` requires that artifact roots carry no `apiVersion` field. In core, `apiVersion: "v1beta1"` and `"v1alpha1"` appear only on catalog members (resources, traits, blueprints; `src/pins/catalog_pins.cue:36`), where they are a contract's level. The only `v1alpha2` left in core is a stale note in `docs/adapters.md:117`.

**Decision**: Name the schema by the CUE module that carries it: `opmodel.dev/core@v2`. `README.md:27` drops "(v1alpha2)" and the README column header becomes "Schema definition (`opmodel.dev/core@v2`)". The `AGENTS.md:163` header becomes "Schema (`opmodel.dev/core@v2`)". Table padding is adjusted so the columns stay aligned.

**Rationale**: Any version label would be invented. The major of the core module is the only version the artifact schemas have, and it is the identifier a reader passes to `OCILoader`.

### D2. Pinned by default, bare major opt-in, and how an additive change arrives

**Context**: Three spots say additive schema changes are "absorbed by floating-major resolution". The owner's wording is "pinned by default, bare major opt-in".

**Explored**: `schema.DefaultSchemaModule` names an exact release (`opm/schema/loader.go:43`). Acquisition and `Render` never read the schema cache; the module's own `cue.mod` resolves core inside the build (`AGENTS.md:221-228`). A bare-major loader makes `SynthesizeInstance` resolve the release through the cache. The release cascade moves `DefaultSchemaModule` with `task -x deps:cascade` and labels the PR `need-human-review` (`AGENTS.md:303`).

**Decision**: The replacement text says, in each spot:

- The default loader pins an exact core release (`schema.DefaultSchemaModule`); a bare major (`opmodel.dev/core@v2`) is opt-in and resolves to the highest published release of that major.
- An additive shape change within a major needs no Go API change. Artifacts take it up by re-pinning core in their own `cue.mod`, and `DefaultSchemaModule`, a Go const (`opm/schema/loader.go:43`), moves separately, in the cascade's `fix(deps)` PR, which re-verifies the render glue.

Concretely:

- `README.md:97`: "The two tracks are independent. The kernel's schema loader pins an exact core release by default (`schema.DefaultSchemaModule`), and a bare major (`opmodel.dev/core@v2`) is opt-in; acquisition and `Render` resolve core through each artifact's own `cue.mod`. An additive shape change within an OPM schema major therefore needs no Go API change: artifacts take it up by re-pinning core, and `DefaultSchemaModule` moves separately, in the release cascade's `fix(deps)` PR, which re-verifies the render glue. A shape break in the schema is itself a coordinated library-breaking event."
- `AGENTS.md:351`: the second sentence becomes "The default loader pins an exact release, `opmodel.dev/core@v2.X.Y[-pre]` (`schema.DefaultSchemaModule`); the bare major `opmodel.dev/core@v2` is opt-in and resolves to the highest published release of that major."
- `AGENTS.md:353`: "A consumer that wants its own pin, or the bare major, passes its own loader, for example:" introduces the unchanged example at `:356` (the example's release equals the default, which the cascade keeps in step).
- `AGENTS.md:361`: the last sentence becomes "Within a major, an additive schema change needs no Go API change: artifacts take it up by re-pinning core in their own `cue.mod`, and `DefaultSchemaModule` (the default for synthesized instances) moves separately, in the cascade's `fix(deps)` PR, which re-verifies the glue."

The word "floating" stays only where it describes the opt-in bare major (the godoc and specs already use it that way); after this change neither `README.md` nor `AGENTS.md` uses it.

**Rationale**: "Absorbed transparently" was true only of a bare-major default. The new text keeps the true half (no Go API change for an additive change) and names the real path, including that the Go const itself moves in a release-class `fix(deps)` commit (`.tasks/cascade/test.sh`, `AGENTS.md` deps releases). The new wording names no release literal, so the cascade's prose warning (main spec `deps-cascade`) stays quiet.

### D3. The layout drops `core/`; `Compiled` is named on the `kernel/` row

**Context**: `README.md:40` lists `opm/core/` holding `Compiled`.

**Explored**: `git ls-tree origin/main opm/` lists catalog, errors, helper, internal, kernel, module, platform and schema. `type Compiled struct` is at `opm/kernel/render.go:90`.

**Decision**: Delete the `core/` row. The `kernel/` row ends "(acquire, synthesize, validate, Render) and `Compiled`, its terminal output".

**Rationale**: Repointing the row to a package that holds more than `Compiled` would mislead; one clause on the existing row says where it lives.

### D4. CONSTITUTION Principle III verbs and pipeline

**Context**: `CONSTITUTION.md:64-65` and `:78-80` describe a loader and processing pipeline that was replaced by acquire verbs and one-build render.

**Explored**: the exported `Kernel` methods are the four `Acquire*` families (module, catalog, platform, instance), `SynthesizeInstance`, `ValidateConfigDetailed`, `Render`, plus `LoadSourceFromFile` / `LoadSourceFromBytes` (which parse nothing) and `SchemaCache`. Values are validated where they are applied, inside the instance build (`README.md:80`).

**Decision**:

- `:64`: "`opm/kernel/` — public `Kernel` struct: the single runtime entry point (acquire, synthesize, validate, render)".
- `:65`: "`opm/module/` — module and instance model, value-validation accessors".
- `:78-80`:

  ```text
  acquire -> Module | ModuleInstance | Platform | Catalog
  SynthesizeInstance(Module, values) -> ModuleInstance
  Render(ModuleInstance, Platform) -> one CUE build -> []*kernel.Compiled
  ```

  A `Catalog` is read and derived from, never rendered (ADR-009); `Render` takes only an instance and a platform (`RenderInput{Instance, Platform}`, `opm/kernel/render.go`).

**Rationale**: The pipeline block illustrates the package boundaries of Principle III, so it names the stages and artifacts at the same grain as the old one, no deeper, and sends into `Render` only what `Render` takes. The `opm/k8s/` bullet at `:70` is untouched (proposal, "Not in this change").

### D5. Section cut

Three content sections and one closing section (verify, then the archive that rides the PR). Section 1 holds the README and AGENTS artifact-kind fixes (D1, README:7 and :19), section 2 the pinning wording (D2), section 3 the package map, the constitution, the further-reading line and the site page (D3, D4, w1-05). Each ends in one `docs:` commit and leaves `main` releasable, since no code changes.

The gate for each section is `TMPDIR=$(mktemp -d) task check` on the whole tree (it includes `task docs:bundle:check`, which builds and lints the `docs/site` pages) and `openspec validate fix-front-door-docs --strict`, per the repo rule that every section's commit task carries the validation gates.

## Risks / Trade-offs

- **[AGENTS.md is edited by several wave-2 changes]** → the hunks here stay at `:163` and `:351-361`. A later change rebases onto this one.
- **[README wording on how core reaches a render could overclaim]** → D2 states only what `AGENTS.md:221-228` and the cascade paragraph already state, in shorter form; the review checks each clause against those lines.
