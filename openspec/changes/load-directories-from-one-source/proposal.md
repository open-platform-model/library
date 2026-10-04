## Why

The directory acquire verbs read their tree more than once, and the build and the stamped
`Source` do not come from the same read. At `origin/main` `58f8151`:

- `AcquireModuleFromDir` and `AcquireCatalogFromDir` build the package from disk
  (`opm/kernel/acquire.go:97`, `:204`) and then walk the same tree a second time through
  `overlaySourceForDir` (`:120-128`, over `sourcetree.OverlayFromDir`) to stamp the overlay. A
  file edited between the two reads leaves `Package` and `Source.Overlay` disagreeing about
  the same artifact.
- `AcquireInstanceFromDir` with values runs its own `os.Stat`, reads the tree into an overlay
  and builds from it (`:364-412`). When that build fails, `attributeValuesError` builds the
  authored package from disk again (`:453`), a third read.
- `loader.LoadDir` takes six positional arguments (`ctx, root, pkg, overlay, env, spec`), and
  its eight callers each assemble `root`/`pkg`/`overlay` by hand, although a
  `*module.Source` already carries exactly `Root`, `Pkg` and `Overlay`. Two callers
  (`synth.Instance`, `loader.FetchArtifact`) build the same `Source` a second time for their
  return value.
- Four near-identical directory acquirers (`acquire.go:92-111`, `199-218`, `241-256`,
  `301-335`) repeat the same absolute-path, load, construct and stamp steps.
- `sourcetree.OverlayFromDir` (`opm/internal/sourcetree/sourcetree.go:114-134`) hand-writes a
  `filepath.WalkDir` that `OverlayFromFS` (`:143-170`) already does over any `fs.FS`.

Owner decisions from the beta-1 kernel-plan walkthrough (2026-10-02/03):

- b2: "Full version — Source-taking LoadDir, one acquireDir helper (keep platform on-disk vs
  overlay stamping difference), read overlay first and build from it, include b5 os.DirFS
  swap. After i1, before b1."
- b5: "OverlayFromDir rides b2." (the rest of b5 goes to the consolidate change)

i1 is merged (library #170). b1 (compile values once) follows this change and owns
`attributeValuesError`'s compile reuse; this change touches that function only where its
read source changes.

## What Changes

- **`loader.LoadDir` takes a Source.** The signature becomes
  `LoadDir(cueCtx *cue.Context, src *module.Source, opts Options, spec ArtifactSpec)`, with a
  new `loader.Options` struct holding `Env []string`, so a later load setting is added in one
  place instead of at eight call sites. `src.Overlay == nil` is on-disk mode and builds the
  package directory (`Root` joined with `Pkg`) exactly as today; a non-nil overlay is overlay
  mode with `Root` as the module root and `./<Pkg>` as the package, exactly as today's
  overlay callers pass. The on-disk path check moves into an exported `loader.CheckDir(dir,
  spec)`, which keeps today's error text: `accessing <label> directory "<dir>": <err>` and
  `<label> path "<dir>" is not a directory`.
- **Callers rewritten once.** `synth.Instance` and `loader.FetchArtifact` build their `Source`
  once and return the same pointer they loaded. The loader tests build an on-disk `Source`
  through a small helper.
- **One directory helper in the kernel.** `opm/kernel/acquire.go` gets one helper that
  resolves the absolute path, runs `loader.CheckDir` first (so a missing directory still
  fails with the same text, before any tree read), describes the directory with
  `sourceForDir`, reads the overlay once when the verb stamps overlay mode, and builds from
  that `Source`. Module and catalog build from the overlay they stamp. Platform and the
  no-values instance stay on-disk mode (`Overlay` nil), as today. The instance-with-values
  path reads the overlay through the same helper, adds the rendered `opm-values.cue` to a
  copy, and builds. `overlaySourceForDir` and the second `os.Stat` in
  `loadInstanceWithValues` go away.
- **Attribution reuses the authored read.** `attributeValuesError` builds the authored package
  from the overlay already read for the layered build (without the rendered values file)
  instead of reading the directory again. Its compile and validation steps are unchanged (b1
  changes them later).
- **`OverlayFromDir` over `os.DirFS`.** `sourcetree.OverlayFromDir(root)` becomes
  `OverlayFromFS(os.DirFS(root), ".", root)` wrapped in today's `reading module tree <root>:`
  error. Keys, the `.cue`-only filter and symlink handling below the root stay the same;
  tests pin them. A root that is itself a symlink is now walked through the link (design LS6).
- **Tests first.** Before the refactor, tests pin the missing-directory and not-a-directory
  error text of all four directory verbs, a module that embeds a non-CUE file through the
  embed attribute, and `OverlayFromDir`'s keys, error text and symlink behaviour.
- **Doc comments** on `LoadDir` and the lines in `acquire.go` that this change makes false are
  rewritten. All other comment work is left to the c4 pass.

Not **BREAKING**. Every changed signature is under `opm/internal`. The exported `Acquire*`
signatures, the `Source` stamping of each verb, the error sentinels and the `Kernel.<Verb>:`
error prefixes are unchanged.

SemVer class: PATCH. Release class of the PR title: `refactor`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `artifact-types`: one new requirement stating that a directory verb checks the path before
  reading anything, reads an overlay-mode tree once and builds the package from that read,
  defers platform and authored-instance acquisition to the on-disk mode their own
  requirements state, and still builds a
  package that embeds a non-CUE file beside it.

## Impact

- Packages: `opm/internal/loader` (`load.go`, `load_test.go`, `registry.go`),
  `opm/internal/synth` (`instance.go`), `opm/internal/sourcetree` (`sourcetree.go`,
  `sourcetree_test.go`), `opm/kernel` (`acquire.go`, `acquire_test.go`,
  `acquire_catalog_test.go`). `opm/module/source.go` is read only.
- Public surface under `opm/`: none.
- Behaviour: a module or catalog acquired from a directory is now built from the overlay
  stamped on it instead of from a separate disk read. For a root package the error text is
  unchanged; for a module or catalog acquired from a subdirectory, a load error now names
  the module root and `./<pkg>` (as the overlay-mode instance path already does) instead of
  the package directory. Build and shape-gate errors name the package directory in both
  modes, so an overlay build with a non-empty package (a synthesized instance, a layered
  instance in a subdirectory) now names that package directory rather than the root.
- Downstream: none. cli and opm-operator call only the exported verbs.
- Sequencing: this change is the head of the library acquire chain (b2, then b1+g2, then
  d1+d3, then g5 part A). Each later change rebases on it before merge.
- No `enhancement.yaml`: the decisions come from the beta-1 kernel-plan walkthrough, not
  from an enhancement entry.
