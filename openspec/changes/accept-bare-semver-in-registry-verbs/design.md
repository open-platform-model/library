## Context

See proposal.md, Why. Design-local decisions are numbered BS1 to BS4, so they do not collide
with any other numbering. Locations are at `origin/main` `cf79a5c`:

- `opm/internal/loader/registry.go:82`: `module.NewVersion(modPath, version)` with the caller's
  string. `:101`: `sourcetree.SyntheticRoot(modPath, version)` with the same string. `:33`:
  `FetchModule` passes the caller's string to `verifyModuleIdentity`, which builds the error
  coordinate from it and strips a `v` before comparing (`:168`).
- `opm/helper/platformmodule/generate.go:197-204`: `canonicalVersion`, used by `Roots`
  (`:74,76`) and by the package-internal `closure_test.go:62,64`.
- `opm/internal/sourcetree/sourcetree.go:171-179`: `SyntheticRoot` sanitises `path@version` into
  one path segment.
- `opm/kernel/acquire.go:39-40` and `:130-131`: the godoc examples `"v0.0.2"` and `"v4.3.0"`.
- `cuelang.org/go` v0.17.1 `mod/module/module.go:218-224`: `NewVersion` rejects any version
  `semver.IsValid` rejects, and an empty version is accepted as "no version".

## Goals / Non-Goals

**Goals:** both spellings acquire the same artifact through both registry verbs, with one helper
holding the rule.

**Non-Goals:** any change to what `platformmodule.Generate` stamps, any wider version parsing
(ranges, `latest`), the operator side, and the b3 consolidation of the other version helpers.

## Decisions

### BS1. The helper lives in a new `opm/internal/modversion` package

`Canonical(v string) string` keeps `canonicalVersion`'s exact behaviour: `""` and any string
starting with `v` come back unchanged; anything else gains a `v`. It validates nothing;
`module.NewVersion` stays the validator, so a malformed version is refused where it is today.

```go
// Canonical returns v with the "v" prefix CUE module versions require. A bare
// SemVer ("1.0.0") gains it; an already-prefixed or empty version is returned
// unchanged. It validates nothing.
func Canonical(v string) string {
	if v == "" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}
```

The package is internal, so the helper adds nothing to the SemVer surface. It is not a
`sourcetree` or `loader` function, because `platformmodule` (helper tier) and the loader both need
it, and b3 will add the other version helpers here. It is not named `version`, which every
caller already uses as a parameter name, nor `semver`, which would collide with
`golang.org/x/mod/semver` when b3 adopts it.

### BS2. Canonicalise at the loader entry, not in the kernel verbs

`FetchArtifact` canonicalises before `NewVersion`, and `FetchModule` canonicalises before it
calls `FetchArtifact` and `verifyModuleIdentity`. The two kernel verbs stay one-line delegations.
Every caller of the registry path goes through these two functions, so the rule cannot be
skipped. `FetchArtifact` canonicalising again is idempotent and covers a direct internal caller.

The parse-failure error keeps the caller's spelling (`parsing artifact version
<path>@<as given>`), so the message names what the caller wrote. Errors after that point
(fetch, staging, identity) name the canonical coordinate, which is the tag that was fetched.

### BS3. `SyntheticRoot` canonicalises its version argument

The owner decision says `SyntheticRoot` uses the canonical form. Putting the call inside
`SyntheticRoot`, rather than relying on the loader passing the canonical string, makes the
"same root for both spellings" property hold for every caller, including render staging
(`stage_test.go:228`), at no cost. Existing callers already pass `v`-prefixed versions, so their
roots do not change.

### BS4. The identity check is unchanged

`verifyModuleIdentity` already compares the declared bare `metadata.version` with the fetched
tag minus its `v`. Fed the canonical version, it behaves identically for both spellings; no spec
text about it changes.

## Risks / Trade-offs

- A caller that relied on a bare version being refused loses that refusal. No such caller is
  known: the cli passes `v`-prefixed versions, and the operator wants the bare form accepted.
- `Canonical("latest")` returns `"vlatest"`, which `NewVersion` refuses. The error then names the
  caller's `latest`, per BS2.
