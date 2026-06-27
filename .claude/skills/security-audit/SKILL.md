---
name: security-audit
description: Security audit skill for the OPM library — the Go kernel/reference runtime that loads, validates, materializes, and compiles untrusted CUE artifacts (Module / ModuleInstance / Platform) and executes catalog transformer expressions. Audits CUE-evaluation safety, artifact input validation, loader path handling, registry/OCI trust, kernel concurrency, and supply chain. Targets a path, feature, or the full project. Produces a severity-ranked report (CRITICAL / WARNING / SUGGESTION).
user-invocable: true
argument-hint: "[path-or-feature]"
---

Perform a security audit of the OPM library (kernel) codebase. Reports findings ranked by severity — never modifies code.

The library is the OPM **reference runtime**: it accepts untrusted CUE artifacts and runs them through a load → validate → materialize → compile → execute pipeline. Unlike most Go libraries, **the core risk is CUE evaluation of untrusted, user-authored content** (artifact CUE and, indirectly, the data filled into catalog transformer `#transform` expressions). There is deliberately **no shell, no `text/template`, no process model, and no logging** (CONSTITUTION). So the audit centers on: does untrusted input ever bypass shape-gating or finalization; is pulled registry content trusted without verification; and is the single-CUE-context concurrency contract upheld.

**Input**: Optionally specify a target after the command:

- A directory path (e.g., `opm/compile/`, `opm/helper/loader/`) — scope to that subtree
- A feature name (e.g., `transformer-execution`, `validation`, `registry`, `concurrency`) — scope to code related to that feature
- Omit entirely — audit the full project (loader → kernel → materialize → compile → execute pipeline)

## Scope Detection

- **Path provided** → Targeted audit of that directory / file
- **Feature keyword provided** → Discover relevant code via Explore subagent, then audit
- **Nothing provided** → Full-project audit of the pipeline

> **Read first**: `CONSTITUTION.md` (kernel neutrality, no logging/shell/process), `openspec/config.yaml` (normative rules). These define the contracts a finding can deviate from.

---

## Audit Dimensions

Eight dimensions tailored to a CUE-evaluating Go kernel. Each is checked against the in-scope code. Skip dimensions structurally irrelevant to the target (e.g., skip Registry Trust when auditing only `opm/compile/finalize.go`).

### Dimension 1: Artifact Input Validation & Shape-Gating

The trust gate: all three artifact types must be validated before the kernel acts on them.

- Every artifact (`Module`, `ModuleInstance`, `Platform`) passes the load-time shape gate before kernel processing: `kind` match, required concrete fields present, embedded `#Module` ref validation (ModuleInstance), empty-string rejection
- No bypass path: a code route that reaches `Compile`/`Materialize`/`Execute` without first passing `helper/loader/file/validate.go` shape-gating + full schema validation
- Full schema validation (`Kernel.Validate*`) runs user values against the module `#config` schema **before** any execution
- Sentinel errors (`ErrInvalidPackage`, `ErrWrongKind`, `ErrMissingRequiredField`) are returned so callers can branch — and validation fails closed (an unparseable/ambiguous artifact is rejected, not partially processed)
- Decode steps (`opm/schema/decode.go`, `opm/module`, `opm/platform`) tolerate missing/extra fields safely (no panic on attacker-shaped CUE)
- Key files: `opm/helper/loader/file/validate.go`, `opm/kernel/validate*.go`, `opm/schema/decode.go`

### Dimension 2: CUE Evaluation & Transformer-Execution Safety

The heart of the threat model — user-influenced data flows into CUE evaluation.

- Transformers are sourced from the **trusted catalog** (`Platform.#composedTransformers`), never from user artifact input — confirm no path lets a Module/Instance supply or override a `#transform`
- `FillPath` injection of `#component` and `#context.*` is non-mutating / purely functional (no in-place mutation of shared CUE values)
- `FinalizeValue()` strips definition fields and constraints before filling, so user-supplied concrete data **cannot bypass or redefine** schema constraints (constraint-injection); confirm every user→transformer data path goes through finalization
- Transformer output evaluation (`unified.LookupPath(schema.Output)`) handles both StructKind and ListKind without trusting attacker-controlled shape to drive unbounded expansion
- No path where user CUE can reference/import arbitrary external packages or registries during evaluation (import surface confined to declared, trusted deps)
- CUE evaluation produces **data, not code** — verify no generated output is later `exec`'d, templated, or eval'd elsewhere
- Key files: `opm/compile/execute.go`, `opm/compile/finalize.go`, `opm/compile/match.go`, `opm/compile/module.go`

### Dimension 3: Loader Path Handling

- Filesystem paths (`filepath.Abs`, `os.Stat`, `IsDir`) originate from **caller arguments** (CLI/embedder), not from artifact content — confirm artifact fields never steer a filesystem read/write
- `filepath.Abs` normalizes but does **not** strip `..` or resolve symlinks — assess whether a hostile caller path or a symlink in a loaded module dir can escape an intended root (lower severity since the caller is semi-trusted, but note it)
- Registry env construction (`registryEnv()` in `release.go`) builds a fresh env slice and does **not** call `os.Setenv` — confirm loader stays concurrency-safe and process-neutral (CONSTITUTION)
- CUE `load.Instances` is relied on to reject malformed/escaping import paths — verify no custom path joining sidesteps it
- Key files: `opm/helper/loader/file/{module,release,platform}.go`

### Dimension 4: Registry / OCI Trust & Integrity

- Schema (`opmodel.dev/core@v1`) and catalogs are pulled via CUE's OCI loader (`OCILoader.Load`, `pullCatalog`); assess that there is **no checksum/signature/digest verification** of pulled modules beyond OCI registry + TLS trust — flag MITM / registry-substitution risk and whether version pinning (vs mutable major `@v0`) is enforced
- Per-call registry override is plumbed via `load.Config.Env` only and never mutates `os.Environ()` (kernel neutrality) — a fresh env slice per call, no global state
- Catalog enumeration/filtering (`enumerate.go`, `filter.go`) cannot be steered by untrusted platform `#registry` content into pulling an attacker-chosen version/module
- Cache key derivation (`opm/materialize/cache/key.go`, `sha256` over the registry subtree) is integrity-only, not a security control — confirm it isn't relied on as one
- Key files: `opm/schema/loader.go`, `opm/materialize/pull.go`, `opm/materialize/env.go`, `opm/materialize/{enumerate,filter}.go`

### Dimension 5: Concurrency & Resource Safety

- `*cue.Context` is **not goroutine-safe**: the one-kernel-per-goroutine contract is upheld — no code shares a single Kernel/`cue.Context` across concurrent calls
- A shared `MaterializedPlatform` read across multiple per-goroutine kernels is genuinely read-only (no mutation of shared CUE values during `Compile`)
- `MaterializeCache` mutex (`opm/materialize/cache/cache.go`) correctly guards the first-call race; no double-pull or torn read
- No package-level mutable state / singletons (kernel neutrality) that could race or leak across embedders
- Resource bounds: CUE evaluation depth/recursion and transformer fan-out are unbounded in-kernel (delegated to the CUE SDK) — flag the **DoS-via-malicious-artifact** gap (deep recursion, comprehension explosion, huge list output) as defense-in-depth; confirm callers can cancel via `context.Context` and that the kernel honors it
- Key files: `opm/kernel/kernel.go`, `opm/materialize/cache/cache.go`, `opm/materialize/materialize.go`

### Dimension 6: Secrets & Error Hygiene

- CONSTITUTION forbids kernel logging — verify there is genuinely no log output that could leak artifact contents
- Structured sentinel errors and grouped CUE diagnostics (`opm/errors/`) do not embed full artifact values / config that may carry secrets (no `%v`/`%+v` dump of an entire `cue.Value` in an error returned to callers)
- No secrets, tokens, or keys hardcoded in source or `testdata` fixtures
- Decoded metadata caches (`Metadata` projections) don't durably retain sensitive config beyond need
- Key files: `opm/errors/*.go`, `opm/module/*.go`, `opm/platform/*.go`

### Dimension 7: Supply Chain & Build

- `cuelang.org/go v0.17.0-alpha.1` is an **alpha** SDK pin on the security-critical evaluation path — flag the alpha dependency and track for a stable upgrade
- `go.sum` present and integrity-checked; module deps pinned; transitive OCI plumbing (`cuelabs.dev/go/oci/ociregistry`, `opencontainers/*`, `golang.org/x/net`, `golang.org/x/oauth2`) reviewed for known CVEs
- `crypto/sha256` used for the cache key (fine); confirm no `math/rand` used for any security-relevant value
- CI workflows (`.github/workflows/`) use least-privilege `permissions:` and pin third-party actions to commit SHAs; `Taskfile.yml` build inputs are not unpinned
- Key files: `go.mod`, `go.sum`, `Taskfile.yml`, `.github/workflows/`

### Dimension 8: Architecture & Trust Boundaries

Apply when scope is project-wide or covers a significant subsystem.

**Trust boundary identification**:
- Trusted: the OPM core schema and catalog modules (pulled from the registry, pinned), the kernel itself
- Untrusted: user-authored Module / ModuleInstance / Platform CUE content
- The boundary is crossed at load (shape-gate), at validation (schema), and at compile (finalize → fill → evaluate)

**Confused deputy assessment**:
- Can untrusted artifact content induce the kernel to pull an attacker-chosen catalog/module, or supply its own transformer?
- Can artifact content escape schema constraints via incomplete finalization?
- Can artifact content steer a filesystem path or registry target?

**STRIDE assessment** — for each pipeline stage crossing the boundary:

| Threat | Question |
|--------|----------|
| **Spoofing** | Can a pulled module/catalog be substituted (no digest verification)? Can an artifact masquerade as a different kind past the shape gate? |
| **Tampering** | Can user data redefine schema constraints (finalization bypass)? Can a pulled artifact be tampered between fetch and use? |
| **Repudiation** | (Largely N/A — kernel is library-level, no audit log by design; note this.) |
| **Information Disclosure** | Can artifact contents/secrets leak via returned errors or diagnostics? |
| **Denial of Service** | Can a malicious artifact trigger unbounded CUE recursion, comprehension blow-up, or huge list output? Goroutine/cache races? |
| **Elevation of Privilege** | Can untrusted CUE reach trusted transformer execution or registry authority it shouldn't? |

**Defense in depth**:
- Shape-gate + schema validation + finalization + trusted-catalog sourcing layer up — no single one is the sole barrier
- Kernel neutrality (no global state, no process mutation) limits blast radius across embedders (`cli`, `opm-operator`)

---

## Technology-Specific Checks

Apply the relevant subset based on in-scope code.

### Go Kernel Code

- No `math/rand` for security-relevant values; `crypto/*` used correctly where present
- All errors checked on security-sensitive calls (load, decode, validate, evaluate) — no silent `_`
- No `os.Setenv`/global state mutation (kernel neutrality); env plumbed via config slices
- No `exec`, no `text/template`, no `os/exec`, no reflection-based deserialization of untrusted payloads (confirm these absences hold)
- CUE value handling: deep operations don't mutate shared/cached values; finalization applied before fills
- `context.Context` cancellation honored on long evaluations where the API exposes it

### CUE Evaluation Specifics

- Finalization (`FinalizeValue`) strips `#`-definitions and constraints before user data is filled — the central constraint-bypass guard
- `FillPath` targets are fixed, schema-defined paths (`#component`, `#context.*`) — not attacker-controlled paths
- Output lookup uses the transformer-declared `schema.Output` path, not a user-supplied selector
- Import resolution confined to declared module deps; no dynamic/user-driven import

### Diagnostic CLI (`cmd/flow-inspect/`)

- Loads fixtures from `testdata/` with a `-library` path and `-stages` flag; hardcoded local registry — confirm it is a dev-only diagnostic that doesn't ship in a sensitive context; low risk but note arg/path handling

---

## Execution Steps

### Full-Project Audit

1. **Map the pipeline & trust boundary**

   Launch an Explore subagent to trace: where untrusted artifacts enter (loaders), where they are shape-gated and schema-validated, where catalogs/schema are pulled, where transformers execute, and where the one-context concurrency contract is asserted.

2. **Audit each dimension**

   Launch Explore subagents (parallelize where independent). Each returns findings with **file path**, **line number(s)**, **what the issue is**, **why it matters**, and **severity**.

3. **Apply technology-specific checks** (Go kernel, CUE evaluation specifics, flow-inspect).

4. **Deduplicate, rank, and generate report.**

### Targeted Audit (Path or Feature)

1. **Identify scope** — a path is used directly; a feature keyword (e.g. `transformer-execution`, `registry`, `concurrency`) is resolved to related code via an Explore subagent.

2. **Apply relevant dimensions** — skip inapplicable ones. Apply Dimension 8 (Architecture) only if the target spans the trust boundary.

3. **Generate report.**

---

## Severity Classification

| Severity | Definition | Examples |
|----------|-----------|----------|
| **CRITICAL** | Exploitable vulnerability, constraint/validation bypass, or trusted-execution escape. Must be addressed before release. | A path letting user artifact CUE supply/override a `#transform`, user data reaching evaluation without finalization (constraint bypass), a route reaching compile/execute without shape-gating, untrusted artifact steering an attacker-chosen registry/module pull, a panic-triggering artifact that crashes embedders |
| **WARNING** | Security weakness with material impact, or best-practice violation that increases attack surface. Should be addressed in the current cycle. | No digest/signature verification of pulled modules (registry substitution), unbounded CUE recursion/fan-out DoS path, a Kernel shared across goroutines, errors that dump full artifact values, alpha CUE SDK on the eval path, loader path that follows symlinks out of root |
| **SUGGESTION** | Defense-in-depth improvement, hardening recommendation, or theoretical risk with low current exploitability. Address when convenient. | Add an evaluation depth/timeout guard, pin schema to a specific version not just `@v0`, add fuzz tests for the shape gate, tighten an already-validated decode path, document the no-audit-log boundary |

### Classification Heuristics

- **Exploitability**: Can a malicious **artifact author** (untrusted) trigger it, or does it require a hostile embedder/caller (semi-trusted)?
- **Impact**: Worst-case outcome — constraint bypass, arbitrary transformer execution, embedder crash/DoS, registry substitution?
- **Scope**: One artifact, all artifacts processed by an embedder, or the whole runtime?
- **False positives**: When uncertain, prefer SUGGESTION over WARNING, WARNING over CRITICAL.
- **Confidence**: Only report findings with >= 80% confidence. If uncertain, state the uncertainty and suggest investigation rather than assert a vulnerability.

---

## Report Format

```markdown
## Security Audit Report

### Scope
- **Target**: Full project | `<path>` | Feature: `<name>`
- **Date**: YYYY-MM-DD

### Summary
| Dimension                                  | Status              |
|--------------------------------------------|---------------------|
| D1 Artifact Input Validation & Shape-Gating| N issues / Clean    |
| D2 CUE Evaluation & Transformer Execution  | N issues / Clean    |
| D3 Loader Path Handling                    | N issues / Clean    |
| D4 Registry / OCI Trust & Integrity        | N issues / Clean    |
| D5 Concurrency & Resource Safety           | N issues / Clean    |
| D6 Secrets & Error Hygiene                 | N issues / Clean    |
| D7 Supply Chain & Build                    | N issues / Clean    |
| D8 Architecture & Trust Boundaries         | N issues / Skipped  |

**Totals**: X CRITICAL · Y WARNING · Z SUGGESTION

### CRITICAL (Must fix)

1. **[Title]** — `file/path:line`
   **Dimension**: (e.g., D2 CUE Evaluation & Transformer Execution)
   **Description**: What the issue is and how it could be exploited
   **Evidence**: Code snippet or pattern observed
   **Recommendation**: Specific fix with file/line target

### WARNING (Should fix)

1. **[Title]** — `file/path:line`
   **Dimension**: ...
   **Description**: ...
   **Evidence**: ...
   **Recommendation**: ...

### SUGGESTION (Nice to fix)

1. **[Title]** — `file/path:line`
   **Dimension**: ...
   **Description**: ...
   **Recommendation**: ...

### Positive Observations
- (Security practices done well — always include at least one)

### Skipped / Out of Scope
- (Dimensions or checks skipped and why)

### Final Assessment
- If CRITICAL issues: "X critical issue(s) found. Address before release."
- If only warnings: "No critical issues. Y warning(s) to consider."
- If all clear: "All checks passed. No security issues identified in scope."
```

---

## Guardrails

- **NEVER make code changes** — this skill is analysis and reporting only
- **The CUE-evaluation boundary is the point** — prioritize whether untrusted artifact CUE can bypass shape-gating, schema validation, or finalization, or reach trusted transformer/registry authority
- **Delegate deep analysis to Explore subagents** — protect the main context window from the volume of file reads and grep operations
- **>= 80% confidence threshold** — if uncertain, state it explicitly and suggest investigation rather than assert a vulnerability
- **Always include Positive Observations** — kernel neutrality, no-shell/no-template/no-log discipline, and finalization-before-fill are real strengths; confirm them
- **Always include Skipped / Out of Scope** — many runtime dimensions (web auth, container) genuinely don't apply to a pure library; say so
- **Include code evidence** — every CRITICAL and WARNING cites a `file:line` and shows the relevant pattern
- **Be specific in recommendations** — name the file/line and the concrete change (e.g., "verify pulled catalog digest in `opm/materialize/pull.go:22` before evaluation")
- **Do not overstate severity** — a hostile-*caller* path (semi-trusted embedder) is generally lower severity than a hostile-*artifact* path (fully untrusted). Don't inflate a theoretical CUE-SDK DoS into a CRITICAL without an exploit path
- **Respect kernel neutrality** — recommendations must not introduce logging, shell, process model, or global state that the CONSTITUTION forbids
- **Respect the target scope** — a targeted audit stays in scope; note adjacent concerns under "Skipped / Out of Scope"

## Graceful Degradation

- **No HTTP/network-server surface** → skip web/auth/transport dimensions entirely (pure library); note in Skipped
- **No containers/manifests in scope** → skip container checks; note in Skipped
- **Only `opm/compile/` in scope** → focus D2 (CUE evaluation) + D5 (concurrency/resource); skip loader/registry
- **Only `opm/helper/loader/` in scope** → focus D1 (validation) + D3 (path handling); skip transformer execution
- **Only dependency/build files in scope** → focus D7; skip runtime dimensions
- **Single small file** → skip D8 (Architecture); note in Skipped
- Always note which checks were skipped and why
