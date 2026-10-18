# library: where do I go?

Router only. Facts and rules are in `AGENTS.md`.

| Task | Read first | Then |
| --- | --- | --- |
| Change the kernel API, a verb or the render pipeline | `opm/kernel/` | `CONSTITUTION.md` |
| Find or add a package, or decide where code lives | `references/package-map.md` | `references/kernel-tiers.md` |
| Check what a package may import | `references/kernel-tiers.md` | `.golangci.yml`, the depguard rules |
| Record a design decision | `adr/` | `openspec/config.yaml` |
| Start a feature change | `openspec/config.yaml` | `.claude/skills/` (`openspec-*`) |
| Write or fix a test, or a registry fixture | `references/testing.md` | `opm/internal/registrytest/` |
| Bump CUE, `cuelang.org/go` or apimachinery | `references/cue-toolchain.md` | `docs/design/` |
| Adapt the Go code to a new core schema shape | `opm/schema/` | `opm/internal/renderstage/` |
| Move the default core release, a `deps/cascade` PR or a workflow pin | `references/release-cascade.md` | `RELEASING.md` at the workspace root |
| A red `Consumer build` or `API diff` job | `references/ci-gates.md` | `.tasks/` |
| Docs bundle, a site page or a doc comment on an exported symbol | `references/docs-bundle.md` | `docs/site/`, `docs-kit.cue` |
| Choose a commit type, or fix a stalled release | `references/commit-style.md` | `migrations/` |
| Cite an enhancement in a comment | `references/comment-citations.md` | `references/enhancements.md` at the workspace root |
| Embed the kernel | `docs/getting-started.md` | `README.md` |
| Migration notes for a break | `migrations/` | `adr/` |
| Commit, open a PR or write GitHub text | `references/git-text.md` at the workspace root | |
