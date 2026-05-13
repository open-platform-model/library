## Context

The kernel-vs-helper distinction is the most important boundary in the redesigned library. Frontends embedding the kernel must know what they MUST use (kernel core) versus what they MAY use (helpers). Without that clarity, every frontend builds its own view of "what counts as the kernel" and they drift.

`opm/helper/values/` (slice 05) established the convention. This slice extends it to the loader and sets the directory layout that subsequent slices follow:

```
opm/
   core/           kernel — shared primitives (Rendered, Resource, Identity)
   errors/         kernel — structured errors
   apiversion/     kernel — version detection (from add-multi-apiversion-support)
   api/v1alpha2/   kernel — version binding
   module/         kernel — Module / Release types
   provider/       kernel — Provider type (retired in slice 09)
   render/         kernel — render pipeline internals
   validate/       kernel — Tier-2 schema validation
   kernel/         kernel — public Kernel struct + phase methods
   
   helper/         opt-in convenience boundary
     loader/
       file/       filesystem-coupled loading
       bytes/      in-memory loading (skeleton; deferred implementation)
     values/       Tier-1 layering (slice 05)
     platform/     Platform composition (slice 10)
     embed/        one-call embedding wrappers (deferred until a consumer asks)
```

The `loader/file` and `loader/bytes` split is intentional: each is a sibling because they share no code and have different I/O assumptions. A frontend that has a filesystem imports `loader/file`; a Crossplane fn that does not imports `loader/bytes` (when implemented).

This slice depends on slice 01 (Kernel struct exists) and slice 06 (phase methods reference loaders through the Kernel). It does NOT depend on slice 09 — the loader move is independent of the match rewrite.

## Goals / Non-Goals

**Goals:**
- Move `opm/loader/*` to `opm/helper/loader/file/*` cleanly, preserving the public API of every function.
- Provide a deprecation shim at the old `opm/loader/` path so downstream code compiles for one SemVer cycle.
- Skeleton the `opm/helper/loader/bytes/` package — empty file with package doc — so future slices have a clear destination.
- Document the helper boundary in `opm/helper/doc.go` and the umbrella enhancement.

**Non-Goals:**
- Implementing `opm/helper/loader/bytes/`. Defer until a real consumer (Crossplane fn, fuzzing harness, in-memory tests) asks for it. YAGNI.
- Implementing `opm/helper/embed/`. Same rationale — defer until justified.
- Touching `opm/helper/values/`. It is already in the right place.
- Restructuring `opm/api/<v>/` or moving `opm/apiversion/`. Those are kernel-core; not helpers.

## Decisions

**Shim, not hard cut.** A `opm/loader/` package remains, with each former exported function re-exported as a deprecated thin wrapper. Reason: gives downstream consumers one SemVer cycle to migrate. The shim is a single file with `// Deprecated:` annotations; trivial to remove later.

**`loader/file` package name, not `loader/fs` or `loader/disk`.** Reason: matches the eventual `loader/bytes` sibling — both are I/O sources named by their substrate. `fs` would imply `io/fs.FS`-based abstraction, which we may add later; `file` is concrete to the current behavior.

**`loader/bytes` ships skeleton-only.** A doc comment, no functions. Reason: avoids YAGNI violation while marking the intent and giving Crossplane fn authors (when that work begins) a known landing place.

**`opm/helper/doc.go`.** A single file at `opm/helper/` (the directory has no Go file otherwise) documenting the boundary: what's in here is opt-in, opinionated, frontend convenience.

**Move, not rename.** `git mv` to preserve history. Each file's content is unchanged except for package declaration (`package loader` → `package file`).

## Risks / Trade-offs

**Risk — downstream import breakage.** Every consumer that imports `opm/loader` must update. Mitigation: shim package re-exports every prior symbol; `// Deprecated:` doc steers migration; CHANGELOG migration recipe.

**Risk — package name collision when both old and new are imported.** A consumer that imports both old `opm/loader` and new `opm/helper/loader/file` must alias one. Mitigation: most consumers will only see one in real code; the shim is a one-cycle bridge.

**Risk — encourages helper sprawl.** Easy to throw any new helper under `opm/helper/` without scrutiny. Mitigation: each helper subpackage requires its own slice; the umbrella enhancement records what's planned. Drift requires a deliberate enhancement.

**Trade-off — slightly deeper import paths.** `opm/helper/loader/file` is more keystrokes than `opm/loader`. Acceptable; the boundary clarity is worth it. Consumers can alias on import (`import lfile "github.com/open-platform-model/library/opm/helper/loader/file"`).
