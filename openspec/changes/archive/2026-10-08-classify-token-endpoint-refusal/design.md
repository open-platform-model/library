## Context

See proposal.md, "Why". The constraints that shape the fix were measured against the embedded CUE (v0.17.1) and its OCI client (`cuelabs.dev/go/oci/ociregistry` at the `go.mod` pin) with a local token-authenticated registry:

| Path | Token answer | Typed chain | Text ends in | Classified before |
| --- | --- | --- | --- | --- |
| `Kernel.AcquireModuleFromRegistry` (direct fetch) | 401, 429, 5xx | `ociregistry.HTTPError` with the status | `<code> <text>` | by status (correct) |
| direct fetch | 403, 404 | `modregistry.ErrNotFound`, no status | `module not found` | `FetchNotFound` |
| directory load with a dependency (`cue/load`) | 401, 429, 5xx | none | `cannot do HTTP request: Get "<url>": <code> <text>` | `FetchUnreachable` (wrong) |
| directory load | 403, 404 | none | `module not found` | `FetchNotFound` |
| push (`modregistry.Client.PutModule`) | every status | none | `cannot make scratch config: cannot do HTTP request: Post "<url>": <code> <text>` | `FetchUnreachable` (wrong) |

Where the typed chain is lost: `ociauth` returns the token endpoint's answer as an `ociregistry.HTTPError` from `RoundTrip` (`ociauth/auth.go`, `doTokenRequest`), `net/http` wraps it in a `*url.Error`, and `ociclient` adds `cannot do HTTP request: %w`. On a direct fetch that chain reaches `Classify` intact, and the `HTTPError` check already runs before the `net.Error` check. `modregistry` then flattens it with `%v` on the push path (`cannot make scratch config: %v`), and `cue/load` flattens every import failure. `modregistry.isNotExist` turns a 403 or 404 into `ErrNotFound` before any wrap.

## Goals / Non-Goals

**Goals:**

- A token endpoint's 401 or 403 is `FetchUnauthorized` with its status, on every chain where the status survives as a type or as text.
- No text that is not this form changes class.

**Non-Goals:**

- Recovering the 403 that `modregistry` turns into `ErrNotFound` on a version lookup. Nothing of the status survives, as a type or as text.
- A new exported type, kind or sentinel. `FetchUnauthorized` plus `Status` already says what a caller needs.
- A library-owned transport that observes token requests (see the rejected alternative below).

## Research & Decisions

### How to recognise the answered token request

**Context**: On the two broken paths the chain holds no typed cause at all: no `HTTPError`, no `net.Error`, no sentinel. The only signal is the text.

**Explored**:

1. Read the typed chain only. Not possible: the probe shows an empty typed chain on the push and on the directory load.
2. Drop the `cannot do HTTP request` text match. A flattened refused connection (a directory module whose dependency registry is down) would stop being `FetchUnreachable` and transient, which the spec requires.
3. Observe the token request in a library-owned `http.RoundTripper` under `modconfig.Config.Transport`, record the answer per operation, and type the error where the operation returns. It needs a recorder in the request context, a second hook above `ociauth`, and a way to carry the result past `cue/load`, which flattens what the registry returns. It would not cover a frontend's own client (the cli push builds its own `modregistry.Client`) without a new exported constructor. Too much surface for one misread form (Principle VII).
4. Narrow the existing text match: a line that carries `cannot do HTTP request: ` and ends in `: <code> <status text>` is an answer, not a missing response.

**Decision**: Option 4. One anchored pattern, `(?m)cannot do HTTP request: [^\n]*: ([1-5][0-9]{2}) ([A-Za-z][A-Za-z' -]*)$`, checked before the bare prefix. The status text MUST equal `http.StatusText(code)` and the code MUST be 400 or higher, as the existing status pattern requires. The kind comes from the existing `kindOfStatus`.

**Rationale**: In the pinned OCI client, `cannot do HTTP request: %w` wraps only what `http.Client.Do` returns, which is a `*url.Error` (`<Op> "<URL>": <cause>`). A transport-level cause (dial, DNS, TLS, timeout, EOF) never ends in `<code> <status text>`; Go's proxy CONNECT failure reports the status text without the code. The only producer of a trailing status there is an `ociregistry.HTTPError` with no underlying error, which `ociauth` builds for a token answer. The URL is printed with `%q` and cannot hold a raw space, so it cannot fake the tail.

This narrows what the existing no-response match claims and adds one anchored variant of the existing status pattern. It adds no new failure family to the text fallback. The form is pinned by `TestCUEFailureForms`, so a CUE or OCI client bump that changes it fails there.

### Owner decision of 2026-10-08

**Context**: The task asked that no message-text match be added. The review held the change on that line and on the fetch-403 case.

**Decision**: The owner accepted the one anchored match as a recorded exception to the strict reading of 0021:D8:R12, and decided that a library-owned registry transport is not planned now. Accepted as built: a fetch whose token endpoint answers 403 stays `FetchNotFound`; a token endpoint's 404 reads `FetchNotFound`; a token endpoint's 429 is no longer transient. `adr/014-one-text-match-for-a-token-endpoint-answer.md` is the record, and the comment at `textTokenAnswer` names it.

### The refresh-token path

**Context**: A client that holds a refresh token (a Docker `identitytoken`, a credential helper, a token server that returns one) asks for a new access token before it sends the request. `ociauth` flattens that token answer on purpose (`cannot acquire access token: %v`, `ociauth/auth.go:247-252`). `net/http` wraps it in a `*url.Error`, which is a `net.Error`, so even a direct call has no typed status and `classifyTyped` answered `FetchUnreachable`. Driven through the real client with `registrytest.NewRefreshTokenRegistry` (`token/refresh/*` in `classify_cue_test.go`).

**Decision**: `classifyTyped` checks the same anchored pattern on the chain's text before it turns a `net.Error` into `FetchUnreachable`. `context.DeadlineExceeded` is still read first.

**Rationale**: It is the same failure as the proposal names (a revoked credential retried as a network failure), and the same single pattern covers it. The cost is one text read inside the typed path, stated in the spec.

### Order against the other fetch forms

**Decision**: The answered-token form runs first in `classifyText`, before the bare `cannot do HTTP request`. When several lines of one text match, the first decides. Everything after it keeps its order.

**Rationale**: It is the more specific form of the same prefix. Keeping the rest of the order means no other text changes class.

### No new API

**Decision**: No new kind for "token endpoint". A caller reads `Kind == FetchUnauthorized` and `Status` (401 for a refused credential, 403 for a refused permission).

**Rationale**: The frontends act the same on a refusal from the registry and from its token endpoint (name the refusal, point to login, do not retry).

## Risks / Trade-offs

- [The form is text and can change with the embedded client] → pinned by the CUE form tests, which drive the real client against a local token registry.
- [A CUE error list that appends text after the status on the same line would not match] → it then classifies as before (`FetchUnreachable`); no regression, and none was observed.
- [A token endpoint's 404 becomes `FetchNotFound`, which the kind's doc words as an absent module] → it keeps `kindOfStatus` uniform and `Status` 404 tells the cases apart; on the lookup paths CUE already reports it as not found.
- [A 403 from the token endpoint on a version lookup stays `FetchNotFound`] → stated in the spec; changing it needs option 3.

## Verification

All commands ran in the worktree.

- `go test ./opm/errors ./opm/kernel -run 'Classify|CUEFailure' -count=1`: both packages pass.
- The same command with `opm/errors/classify.go` put back to the section 1 commit: `TestClassify_CUEFailureForms` fails for `token/load/401`, `token/load/503` and all six `token/push/*`; `TestClassify_Text` fails for the five `token answer` rows; `TestFetchClassify_TokenEndpointRefusal` fails for the directory module. The rows that pin unchanged answers pass in both states.
- `task check`: exit 0 (`0 issues.` from lint; every package `ok`).
- `task api:diff`: exit 0. It lists no new incompatible change for this branch; the entries it prints are inherited from the base since the last release tag.

Follow-up for the cli, after a library release: bump the library pin, and change `TestPush_TokenEndpointRefusal_Pinned` to expect a refused credential (`FetchUnauthorized`, status 401 or 403) and not connectivity. No cli source change is needed for the classification itself: `internal/cuemod/connectivity.go` already reads `Classify`.
