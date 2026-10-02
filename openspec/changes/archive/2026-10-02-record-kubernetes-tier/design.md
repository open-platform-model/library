## Context

See proposal.md for why. The facts this change rests on, all read on library `099117b` (after `1.0.0-beta.1`):

- `.golangci.yml` has one depguard rule, `kernel-never-imports-helper`. It denies `github.com/open-platform-model/library/opm/helper` to non-test files under `opm/kernel`, `opm/module`, `opm/platform`, `opm/schema`, `opm/errors` and `opm/internal`. `opm/catalog`, which `acquire-catalog-artifact` added, is not in its file list.
- `opm/helper/doc.go`, `README.md` § Helper boundary, `AGENTS.md` § Repository Rules and the `helper-packages` main spec all say that anything outside `opm/helper/` is kernel contract. Once `opm/k8s` exists, that statement is false.
- No package under `opm/` imports `k8s.io/*`, and `go.mod` does not require it.
- ADR-008's Decision has four numbered rules. The sentence "The kernel derives no ordering of its own" ends rule 4 ("Lifecycle facts are data off a build"). Rule 3 is "The kernel names an action; it never performs one".
- ADR-007 was amended in place by `read-provider-count-from-core`: one dated "Amended ... by `<change>`" sentence on the Status line, with the rules left as written.

## Goals / Non-Goals

**Goals:**

- The decision is recorded once, as ADR-011, and everything else (constitution, package doc, README, AGENTS.md, specs) points at it rather than restating it.
- The fence is enforced by `task lint` before any `opm/k8s` package exists, and the rule is proven to fail on a violating import.
- No statement in the repo still says that everything outside `opm/helper/` is kernel contract.

**Non-Goals:**

- Any `opm/k8s` package, `go.mod` change or move of `opm/helper/objectset` (`add-kubernetes-object-packages`).
- Editing enhancement 0012. Its 0012:D3 to 0012:D6 are written there separately. This change only cites them.
- Rules in cli or opm-operator.

## Decisions

### 1. Four depguard rules

```yaml
nothing-imports-k8s-tier:
  files:
    - "**/opm/kernel/**"
    - "**/opm/module/**"
    - "**/opm/platform/**"
    - "**/opm/catalog/**"
    - "**/opm/schema/**"
    - "**/opm/errors/**"
    - "**/opm/internal/**"
    - "**/opm/helper/**"
  deny:
    - pkg: github.com/open-platform-model/library/opm/k8s
k8s-tier-imports-no-runtime:
  files:
    - "**/opm/k8s/**"
  deny:
    - pkg: k8s.io/client-go
    - pkg: sigs.k8s.io/controller-runtime
    - pkg: github.com/fluxcd
    - pkg: github.com/open-platform-model/library/opm/internal
    - pkg: github.com/open-platform-model/library/opm/helper
kernel-imports-no-kubernetes:
  files:  # the globs of nothing-imports-k8s-tier minus opm/helper
    - "**/opm/kernel/**"  # ... through "**/opm/internal/**"
  deny:
    - pkg: k8s.io
    - pkg: sigs.k8s.io
opm-imports-no-cluster-runtime:
  files:
    - "**/opm/**"
  deny:
    - pkg: k8s.io/client-go
    - pkg: sigs.k8s.io/controller-runtime
    - pkg: github.com/fluxcd
```

The inward rule MUST cover test files as well. `kernel-never-imports-helper` exempts tests because kernel fixtures build their platform module through `opm/helper/platformmodule`. Nothing yet needs a kernel test to import `opm/k8s`, and an exemption is easier to add later, with its reason, than to remove. The inward rule lists `opm/catalog` and `opm/helper`, because 0012:D3 names every other `opm/` package and `opm/catalog` is a kernel package.

The same edit adds `opm/catalog` to `kernel-never-imports-helper`'s file list. The rewritten `helper-packages` requirement names it as a kernel package that lint keeps off the helper tier, and today nothing enforces that.

The outward rule adds `opm/internal` and `opm/helper` to the three framework prefixes the brief names. 0012:D3 bounds the tier by a denylist: beyond the standard library and the CUE SDK that the kernel's output types carry, it imports only the kernel's exported packages and apimachinery. `opm/internal` is reachable from `opm/k8s` by Go's internal rule (both sit under `opm/`), so only lint can close it. `opm/helper` is the opt-in tier, and a mandatory tier depending on it would make the helper mandatory, the inversion `kernel-never-imports-helper` exists to prevent. `objectset` moves into `opm/k8s/object` (0012:D3), so no tier package needs the helper.

The review added the last two rules. The constitution and ADR-011 say the kernel imports no Kubernetes package and no `opm/` package imports client-go, controller-runtime or Flux. Before these rules only `go.mod` enforced that, and the first tier package adds apimachinery to `go.mod`. `kernel-imports-no-kubernetes` covers kernel test files too, since no kernel test needs Kubernetes. `opm-imports-no-cluster-runtime` overlaps the outward rule under `opm/k8s` on purpose: the outward rule keeps its tier-specific messages, and the wide rule covers the helper tier.

### 2. A new `kubernetes-tier` capability, plus a rewritten `helper-packages` requirement

The tier's rules (placement, both fence directions, no executor, the obligation) describe a capability that is not a helper. Writing them as ADDED requirements in `helper-packages` would file the mandatory tier under the opt-in tier's spec, the confusion ADR-011 rejects. The repo's specs are one capability per boundary or verb (`helper-packages`, `duplicate-object-identities`, `platform-module-generation`), so the tier gets its own.

`helper-packages` still needs a delta, because its first requirement says "Anything outside `opm/helper/` SHALL be considered part of the kernel core contract", and its first scenario asserts that `doc.go` says so. That requirement is MODIFIED and keeps all five scenarios by name. Only the body of "Helper boundary documented" changes. The rewrite also lists `opm/catalog` in its kernel list and its "exactly these exported packages" sentence. That sentence has been false since `acquire-catalog-artifact` and has to be rewritten here anyway, to leave room for `opm/k8s`. The spec's Purpose paragraph carries the same sentence, and a delta cannot edit it. It is fixed by hand when the change is archived (see Archive notes).

### 3. ADR-008 gains a rule 4 clarification and keeps rule 3

The brief names "ADR-008 rule 3" and quotes the sentence "The kernel derives no ordering of its own". That sentence is in rule 4. The amendment follows the quoted text: one dated sentence on the Status line (the ADR-007 precedent), plus a one-line parenthetical at the end of rule 4 so a reader of the rule meets the clarification there. For the same reason, ADR-011 cites ADR-008 rules 1 to 3 for "no loop, caller-owned state, names actions and never performs them". The brief said "rules 1 and 2".

Rule 3 ends "If the library ever ships executor backends they live under `opm/helper/`". The owner decided on 2026-10-02 that this allowance stays. ADR-011 item 4 and the `kubernetes-tier` spec ban an executor in `opm/k8s` and the kernel only, so ADR-008 rule 3 needs no amendment.

### 4. Label stamping against enhancement 0010

0012:OQ11 says enhancement 0010 moves the `module.opmodel.dev/version` label "from the schema to the kernel". That plan was reversed. Archived 0010:D9 (revised 2026-08-04, absorbing 0010:D39) keeps the label declared by the schema and makes the kernel only its verifier, because the CLI's local-directory render has no resolved coordinate to stamp from. Labels therefore stay stamped in CUE in both cases, and ADR-011's item 8 does not conflict with 0010. The version label is part of the digest input and is identical on both frontends. Only the managed-by value (`#runtimeName`) differs between them, and only that value is left out. 0012:OQ11's wording about 0010 is stale. Correcting it belongs to the enhancement edit that writes 0012:D6.

### 5. `enhancement.yaml` declares 0012 and claims no decision

Delivery logging claims a decision only when the change delivers it whole (`#LogEntry.decisions`). An over-claim is the one error the coverage derivation cannot absorb. This change delivers the library half of 0012:D3 (the record, the fence, the constitution) but not the frontend obligation or any package, and it delivers none of 0012:D4, 0012:D5 or 0012:D6, which it only records. So the file names enhancement 0012 with an empty decision list. The later package and frontend changes claim the numbers.

## Archive notes

Not tasks, because tasks.md allows no delivery operation past the section commit:

- `enhancement.yaml` names 0012 and claims no decision (Decision 5), so the delivery log line records the landing without covering 0012:D3 to 0012:D6.
- The `helper-packages` main spec's Purpose paragraph still says that everything outside `opm/helper/` is kernel contract, and a delta cannot reach it. Edit it by hand when the change is archived.

## Research & Decisions

### Where the tier's requirements live
**Context**: The brief allowed a new capability or ADDED requirements in `helper-packages`.
**Explored**: The spec list under `openspec/specs/`. Each spec owns one boundary or one verb, and `helper-packages` owns the opt-in boundary.
**Decision**: New `kubernetes-tier` capability, plus a MODIFIED `helper-packages` boundary requirement.
**Rationale**: Decision 2 above.

### Does a depguard deny on a prefix match subpackages
**Context**: `github.com/fluxcd` must cover `github.com/fluxcd/pkg/ssa`, and `.../opm/k8s` must cover `.../opm/k8s/object`.
**Explored**: depguard v2 matches a `deny.pkg` entry as a prefix of the import path. The existing rule already relies on this (`.../opm/helper` covers `.../opm/helper/platformmodule`).
**Decision**: Prefixes without a trailing `/`. tasks.md 3.2 proves both rules bite on throwaway probe files that are never committed.
**Rationale**: One entry per framework, and future subpackages are covered without editing the rule.

## Risks / Trade-offs

- **A fence over an empty directory proves nothing until it is probed.** Mitigation: tasks.md 3.2 runs `task lint` against throwaway violating files for both directions and records the failure. The probes are deleted before the commit.
- **A prefix deny on `github.com/fluxcd` also covers any harmless Flux utility a future package might want.** Accepted: 0012:D2 excludes Flux as a whole, and relaxing it would be an ADR amendment, not a lint edit.
- **The outward rule is a denylist.** ADR-011, the `kubernetes-tier` spec and 0012:D3 all state the bound as one: the standard library, the CUE SDK, the kernel's exported packages and apimachinery, and never the named runtimes or library tiers. A strict allow list would enforce that literally. It is deferred to `add-kubernetes-object-packages`, which first learns which imports the tier actually needs. Writing it now would guess.
