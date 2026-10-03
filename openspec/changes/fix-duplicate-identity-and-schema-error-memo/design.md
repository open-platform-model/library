## Context

See proposal.md, Why. Three independent fixes share one change because each is small and the
owner bundled them. Design-local decisions are numbered DM1 to DM5 so they collide with no other
numbering. Line numbers are at `origin/main` `cf79a5c`.

## Goals / Non-Goals

**Goals:** `Duplicates` finds objects Kubernetes treats as one; an errored core build never
reaches a caller as a value; the shared-platform race test covers the first render.

**Non-Goals:** changing the exported `objectset.Identity`; making `schema.Cache` retry; the
opm-operator `ModFiles` memo; moving `objectset` into `opm/k8s/`.

## Decisions

### DM1: Key on group, keep `Identity`

`Duplicates` builds its map on an unexported key:

```go
type key struct{ group, kind, namespace, name string }

func groupOf(apiVersion string) string {
	group, _, found := strings.Cut(apiVersion, "/")
	if !found {
		return "" // "v1": the core group
	}
	return group
}
```

An object with no `apiVersion` maps to the core group, as the empty string did before. Kind,
namespace and name compare exactly, as today. The exported `Identity` keeps its four fields and
its `String()`: a row's `Identity` is the first-rendered object's, `APIVersion` verbatim, so a
same-version row is byte-identical to today's. Changing `Identity` to carry `Group` was the
breaking alternative the owner declined; both runtimes build `Identity` literals in tests.

### DM2: `Producer.APIVersion`, named in the message only when versions differ

`Producer` gains `APIVersion string`, set from each object's own `apiVersion`. `Producer.String()`
is unchanged. `DuplicateIdentitiesError.Error()` appends ` as <apiVersion>` to each producer of a
row whose producers do not all carry the same `APIVersion`:

```text
2 rendered objects share one identity, so the last apply would silently overwrite the first:
  apps/v1 Deployment web rendered by component "a" (…/deployment@1.0.0) as apps/v1 and component "b" (…/legacy@1.0.0) as apps/v1beta2
```

Comparing producers to each other, not to `Identity.APIVersion`, keeps today's wording for a
same-version row and for hand-built rows whose producers leave `APIVersion` empty (the cli and
opm-operator test literals), so neither consumer's wording test moves.

### DM3: An errored build is a load failure

In `loadVersioned`, after `ctx.BuildInstance(instances[0])`:

```go
val := ctx.BuildInstance(instances[0])
if err := val.Err(); err != nil {
	return cue.Value{}, "", fmt.Errorf("schema OCILoader: building %q: %w", moduleID, err)
}
```

The resolved version is dropped with the value, so `Cache.ResolvedVersion()` stays `""` after the
failure, as its spec already requires for a failed load. `OCILoader.Load` goes through
`loadVersioned`, so both entry points change together.

### DM4: The Cache stays never-retry

Owner decision: the `sync.Once` memo, error included, stays. The fix is upstream of it: `Get`
already memoises whatever `Load` returns, and after DM3 an errored build arrives as an error. The
`Cache` doc comment gains one sentence saying so; its code does not change.

### DM5: The cold concurrent render compares against a separate acquisition

The new `TestRender_SharedPlatformConcurrentRendersCold` takes its expected objects from a render
of a separately acquired platform and instance, then acquires a second platform and instance on
the same kernel and starts the eight goroutines on that pair with no render before them. Each
`AcquirePlatformFromDir` builds into its own `cuecontext.New()` (`opm/kernel/acquire.go:243`), so
the baseline render touches nothing the shared pair holds, and the shared `Package`'s first
`LookupPath(schema.ContractsProvidedBy)` (`opm/kernel/render.go:341`) is raced. One kernel is
used because `newRenderKernel` sets the test's registry environment, and a second call would
replace it under the first kernel. It runs under `-race` through `task test` like its
warm sibling, which stays as it is.

## Research & Decisions

### What `val.Err()` catches

**Context**: DM3 relies on `BuildInstance` reporting build failures through the value.
**Explored**: a throwaway probe (not committed) built four one-file packages with
`load.Instances` plus `cuecontext.New().BuildInstance` on CUE v0.17.1: a field conflict
(`x: 1 & 2`), a root conflict, an unresolved reference (`x: y`) and a struct/int conflict. All four
loaded without `Instance.Err` and returned a non-nil `val.Err()`. A second probe loaded the real
default core (`schema.OCILoader{}` against the seeded workspace cache): `val.Err()` is nil.
**Decision**: check `val.Err()` only, not `val.Validate()`. A schema is not concrete, and
`Validate()` with default options reports no more for these cases.
**Rationale**: the failure classes the review named are all caught, the real core is unaffected,
and the test fixture can be a stand-in core served by `registrytest.NewRegistryWithCore` with an
unresolved reference. design.md carries no unverified assumption, so section 1 is not a spike.

### Where the group comes from

**Context**: DM1 needs the API group without a discovery client.
**Explored**: Kubernetes `apiVersion` is `group/version`, or a bare `version` for the core group
(`k8s.io/apimachinery` `schema.ParseGroupVersion` splits on the first `/`). The helper may not
import apimachinery (only `opm/k8s/` may).
**Decision**: split on the first `/` in the standard library.
**Rationale**: same result as `ParseGroupVersion` for every well-formed value, no new import.

## Risks / Trade-offs

- A module that today renders the same kind and name under two versions of one group now fails
  at the runtime's duplicate check. That module was already broken at apply (last write wins);
  the refusal is the fix.
- An operator whose core build is broken now fails at start-up instead of later. Intended.
