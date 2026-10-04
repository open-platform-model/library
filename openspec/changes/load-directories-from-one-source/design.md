## Context

See proposal.md, Why. Design-local decisions are numbered LS1 to LS7, so they do not collide
with any other numbering. Locations are at `origin/main` `58f8151`:

- `opm/internal/loader/load.go:39-93`: `LoadDir(ctx, root, pkg, overlay, env, spec)`. The
  on-disk branch stats `root` joined with `pkg` (`:44-58`); the overlay branch wraps the bytes
  as `load.Source` and sets `cfg.ModuleRoot = root` (`:64-73`).
- Eight callers: `opm/kernel/acquire.go:97` (module), `:204` (catalog), `:246` (platform),
  `:313` (instance, no values), `:403` (instance with values), `:453`
  (`attributeValuesError`); `opm/internal/synth/instance.go:190`;
  `opm/internal/loader/registry.go:128`; plus `load_test.go:26` and `:140`.
- `acquire.go:120-128`: `overlaySourceForDir`, the second tree read for module and catalog.
- `acquire.go:364-412`: `loadInstanceWithValues`, with its own `os.Stat` (`:365-371`) that
  repeats the loader's text for the `instance` label.
- `acquire.go:479-499`: `sourceForDir`, the ancestor walk for `cue.mod/module.cue`.
- `opm/internal/sourcetree/sourcetree.go:114-134` (`OverlayFromDir`) and `:143-170`
  (`OverlayFromFS`).
- `cuelang.org/go` v0.17.1 `cue/load/config.go:286-295`: `Config.Overlay` is "overlaid on top
  of the host operating system"; a file not in the overlay is read from disk.

## Goals / Non-Goals

**Goals:** one `Source`-taking build step; one kernel helper for every directory verb; an
overlay-mode verb reads the tree once, before the build, and builds from that read; the
`os.DirFS` form of `OverlayFromDir`; no change to any exported signature, stamped `Source`,
sentinel or `Kernel.<Verb>:` prefix.

**Non-Goals:** compiling values once and the `validateSources` rework (b1+g2, the next
change on this chain); a per-Kernel registry client in the load options (g5 part A); the
`IsLocal` and version-helper consolidation that is the rest of b5 (the consolidate change);
the general comment pass (c4); carrying non-CUE files in `Source.Overlay`.

## Research & Decisions

### LS1. `LoadDir(cueCtx, src, opts, spec)` with a `loader.Options` struct

**Context**: the owner decision asks for a Source-taking `LoadDir`; the plan asks for a small
options struct so g5 part A adds the registry client in one place.
**Decision**:

```go
// Options carries the load settings every build shares.
type Options struct {
	// Env is the environment slice load.Config consults; nil reads the
	// process environment unchanged.
	Env []string
}

func LoadDir(cueCtx *cue.Context, src *module.Source, opts Options, spec ArtifactSpec) (cue.Value, error)
```

**Rationale**: `module.Source` already holds `Root`, `Pkg` and `Overlay`, the three values every
caller assembles today. The two modes keep today's load configuration exactly, so no existing
caller sees a different build:

| Mode | `load.Config` | package argument |
| --- | --- | --- |
| on-disk (`Overlay == nil`) | `Dir` = `Root` joined with `Pkg` | `.` |
| overlay | `Dir` = `Root`, `ModuleRoot` = `Root`, `Overlay` = the bytes | `.` when `Pkg` is empty, else `./<Pkg>` |

On-disk mode builds from the package directory and not from `Root` with `./<Pkg>`, because
today's on-disk callers pass the package directory as `root`. Every load and build error in
that mode then names the same directory it names today. Overlay-mode errors name `Root`
and the package argument, as the overlay callers' errors do today. Module and catalog move
from on-disk to overlay mode (LS3), so for a module or catalog acquired from a subdirectory a
load or build error names the module root and `./<Pkg>` instead of the package directory; for
a root package the text is unchanged. A nil `src`, or one with an empty
`Root`, is refused with a plain `errors.New("source carries no module root")`, the wording
`sourcetree.PackageName` and `sourcetree.ReadFile` already use for the same caller
precondition. It does not wrap `oerrors.ErrInvalidPackage`: that sentinel classifies a
package defect, and a missing root is a caller bug.

`loader` already imports `opm/module` (`registry.go`), so there is no new import edge.

### LS2. `loader.CheckDir` owns the on-disk path check

**Context**: today two copies of the stat check exist (`load.go:44-58` and
`acquire.go:365-371`). The supervisor note asks for a stat check first in the kernel helper,
with the missing-directory text pinned in a test.
**Decision**: one exported internal function, `loader.CheckDir(dir string, spec ArtifactSpec)
error`, returning `accessing <label> directory %q: %w` (wrapping the `os.Stat` error, so
`errors.Is(err, fs.ErrNotExist)` holds) or `<label> path %q is not a directory`. `LoadDir`
calls it in on-disk mode; the kernel helper calls it before `sourceForDir` and before any
overlay read; `loadInstanceWithValues` drops its copy.
**Rationale**: today a missing directory fails on the first `LoadDir` line, before anything
else reads the tree. If the overlay read came first, the same directory would fail as
`reading module tree ...` from the walker. Calling `CheckDir` first keeps the text. For the
instance-with-values path the text is the same as today's own copy, because its label is
`instance`. The errors carry no `Kernel.<Verb>:` prefix, as today.

### LS3. One kernel helper, two stamping modes

**Context**: the owner decision keeps the difference between platform (on-disk `Source`) and
module/catalog (overlay `Source`).
**Decision**:

```go
// dirSource resolves dirPath, checks it, and describes the package in it as
// a Source: on-disk mode, or overlay mode with every .cue file under the
// module root read once.
func dirSource(verb, dirPath string, spec loader.ArtifactSpec, withOverlay bool) (*module.Source, error)

// acquireDir is dirSource followed by the build from that same Source.
func (k *Kernel) acquireDir(cueCtx *cue.Context, verb, dirPath string, spec loader.ArtifactSpec, withOverlay bool) (cue.Value, *module.Source, error)
```

`dirSource` runs `filepath.Abs` (wrapped `<verb>: resolving <label> directory: %w`, today's
text), `loader.CheckDir`, `sourceForDir`, and with `withOverlay` true,
`sourcetree.OverlayFromDir(src.Root)` (wrapped `<verb>: %w`, today's text). `acquireDir` then
calls `loader.LoadDir(cueCtx, src, loader.Options{Env: k.loadEnv()}, spec)` and returns its
error unwrapped, as every directory verb does today.

| Verb | `withOverlay` | built from |
| --- | --- | --- |
| `AcquireModuleFromDir` | true | the stamped overlay |
| `AcquireCatalogFromDir` | true | the stamped overlay |
| `AcquirePlatformFromDir` | false | disk |
| `AcquireInstanceFromDir`, no values | false | disk |
| `AcquireInstanceFromDir`, with values | true (through `dirSource`) | a copy of the overlay plus `opm-values.cue` |

The three plain verbs become `acquireDir`, then `New<X>FromValue` (wrapped `<verb>: %w`), then
stamp `.Source` with the `Source` the build used. `overlaySourceForDir` is deleted, so the
directory is not walked a second time to stamp the `Source`. No runtime probe counts the
reads (a package-level seam would be global mutable state, which CONSTITUTION Principle I
bans), so that part is checked by reading the diff. What a test can check, it does: a kernel
test calls `dirSource` through `export_test.go`, rewrites a `.cue` file on disk, builds from
that `Source` with `loader.LoadDir`, and asserts the bytes read first win. A bool and
not a named mode type: there are exactly two modes, and the table above is the only place
that picks one.

### LS4. Instance with values builds from a copy of the authored overlay

**Decision**: `loadInstanceWithValues` takes the authored `Source` from
`dirSource(..., true)`, merges the values sources, reads the package name from that `Source`
(`sourcetree.PackageName` reads overlay-mode entries without touching disk), renders the
values file, and builds a second `Source` with the same `Root` and `Pkg` and a
`maps.Clone` of the authored overlay plus `opm-values.cue`. That second `Source` is the one
built and stamped. On a build failure, `attributeValuesError` receives the authored `Source`
and builds from it.
**Rationale**: the plan says attribution rebuilds "from the same overlay minus the values
file". Keeping the unmodified authored overlay is that set without a delete. A delete would
also be wrong in one case: if the author has an `opm-values.cue` on disk, today's overlay
replaces it with the rendered file, and deleting the key would show the disk file through the
host layer, while the kept authored overlay holds its original bytes, which is what today's
disk rebuild reads. The supervisor note leaves the rest of `attributeValuesError` to
lib-b1g2, so only its parameter (`absDir` becomes the authored `*module.Source`) and its
`LoadDir` call change. Attribution then makes no further read of the directory; like LS3,
that is checked by reading the diff, not by a runtime probe.

The order of failures shifts slightly: the overlay is now read before the values sources are
merged, so a directory that both fails to read and has bad sources reports the read error.
No test or caller depends on that order.

### LS5. Overlay-first build and embedded non-CUE files (spike done during planning)

**Context**: `OverlayFromDir` reads `.cue` files only. The research flagged that a module using
the embed attribute on a non-CUE file might stop building once it is built from the overlay,
and the supervisor asked for a test and, if needed, a disk fallback.
**Explored**: a throwaway test (not committed) against `origin/main` built a module root whose
`module.cue` declares the embed extern and embeds `data.json`, and a `sub` package that embeds
`sub/d.json`, four ways: on disk, and through today's `LoadDir` overlay mode over
`OverlayFromDir(root)` with the real root as `Root`, for the root package and for `./sub`. All
four built and read the embedded values. After the `.cue` file was rewritten on disk, the
overlay build still evaluated the bytes read earlier.
**Decision**: no disk fallback. Section 1 commits a kernel test that acquires such a module
from a directory and reads the embedded value, so a later CUE bump that changes the overlay
layering fails a test instead of a user.
**Rationale**: `load.Config.Overlay` sits on top of the host filesystem, so a file missing from
the overlay is read from disk under the real root, and an overlay `.cue` entry replaces the
file of the same path. Building a directory-acquired artifact from its overlay therefore sees
the `.cue` bytes that were read once and every other file from disk, which is what the
on-disk build sees. Not covered, and not changed by this change: when a module acquired
this way is staged elsewhere (`Render` re-keys overlay inputs under the staging directory),
its non-CUE files are not carried. That is today's behaviour, and the Non-Goals above list it.

One residual difference: a `.cue` file created on disk between the read and the build is seen
by the build (through the host layer) but not stamped. The window is the same function call,
and today's two reads have the same window with the roles reversed. It is noted here, not
fixed.

### LS6. `OverlayFromDir` over `os.DirFS`

**Decision**:

```go
func OverlayFromDir(root string) (map[string][]byte, error) {
	overlay, err := OverlayFromFS(os.DirFS(root), ".", root)
	if err != nil {
		return nil, fmt.Errorf("reading module tree %s: %w", root, err)
	}
	return overlay, nil
}
```

**Rationale**: `OverlayFromFS` already walks any `fs.FS` with the same `.cue` filter and keys
each file as `filepath.Join(keyRoot, rel)`, which equals the `filepath.WalkDir` path for a
clean root. Neither walker descends into a symlinked directory, and both read a symlinked
`.cue` file through the link (`os.DirFS` and `os.ReadFile` both follow it). Tests pin the keys,
the `reading module tree` text for a missing root, and both symlink cases, so a difference
shows up in `TestOverlayFromDir_*` and not in an acquisition. `opm/internal/renderstage`'s test
use of `OverlayFromDir` is unaffected.

The inner error text changes in two places; only the `reading module tree <root>:` prefix is
pinned. A missing root now reports `stat` where `filepath.WalkDir` reported `lstat`, and a
read failure in the middle of the walk now carries `OverlayFromFS`'s own `reading <rel>:` wrap
(`reading module tree R: reading sub/a.cue: open ...`). In the kernel, `loader.CheckDir` runs
before the overlay read, so only a mid-walk read failure can reach a directory verb's caller.

### LS7. Module root in overlay mode

**Context**: the research notes that overlay mode sets `cfg.ModuleRoot = Root` explicitly,
where today's on-disk module and catalog builds let cue/load find the module root.
**Decision**: accept it, and run the flow and parity suites in the last real section.
**Rationale**: `sourceForDir` and cue/load search for the same ancestor except for a
`cue.mod` directory with no `module.cue` in it, which cue/load treats as a root and
`sourceForDir` skips. Such a tree is not a valid CUE module and fails either way. A
directory with no enclosing module is its own `Root` in both, and the layered-instance path
already builds module-less overlays (`TestKernel_AcquireInstanceFromDir_WithSources_ModuleLess`).

## Impact across packages

- `opm/internal/loader`: new `Options`, new `CheckDir`, `LoadDir` signature. `FetchArtifact`
  keeps its exported signature (`env []string`) and passes `Options{Env: env}`; g5 part A
  changes it.
- `opm/internal/synth`: `Instance` builds one `Source` and returns it.
- `opm/internal/sourcetree`: `OverlayFromDir` body only.
- `opm/kernel`: `dirSource`, `acquireDir`, the four directory verbs, `loadInstanceWithValues`,
  `attributeValuesError`'s parameter. `overlaySourceForDir` removed.
- Public surface under `opm/`: none. Error text of every directory verb is unchanged for a
  root package; LS1 and the proposal Impact list the one subpackage message change.

## Risks

- Error text drift in the directory verbs. Section 1 pins the missing-directory and
  not-a-directory text of all four verbs before any code moves.
- The acquire chain after this change (b1+g2, d1+d3, g5 part A) edits the same functions.
  Those changes rebase on this one; this change keeps `attributeValuesError`'s body as it is
  apart from its read source, so the b1 diff stays local.
