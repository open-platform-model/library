## Why

A registry that uses token authentication (GHCR, Docker Hub) answers a request with a Bearer challenge, and the client then asks the registry's token endpoint for a token. When that endpoint answers 401 or 403, `opm/errors.Classify` reports the failure as `FetchUnreachable` wherever the embedded CUE has flattened the cause into text: a module push, and a directory load that must fetch a dependency. `FetchUnreachable` is transient, so the cli prints "registry unreachable" with no login hint, and a controller retries a refused credential as if the network were down. The registry did answer; the answer was a refusal.

The cause is in the text fallback. The OCI client writes `cannot do HTTP request: <request>: <code> <status text>` for a token request that was answered, because the token client returns the answer as an error from the HTTP round trip. `Classify` matches the prefix `cannot do HTTP request` first and reads it as "no HTTP response".

## What Changes

- `Classify` reads an answered token request by its status: 401 and 403 become `FetchUnauthorized`, 404 `FetchNotFound`, everything else `FetchOther`, with `Status` set. None is `FetchUnreachable`; only a 5xx stays transient.
- The text fallback's existing no-response match (`cannot do HTTP request`) is narrowed: it no longer claims a line that ends in an HTTP status. No new failure family is matched by text. The status pattern gains one anchored variant for the end-of-line form.
- A test-only token-authenticated registry (`registrytest.NewTokenRegistry`) drives the form through the embedded CUE, on a push, a registry fetch and a directory load, for token answers 401, 403, 404, 429, 500 and 503.
- Godoc of `Classify`, `FetchUnauthorized` and `FetchUnreachable` and the `fetch-error-classification` spec state the rule.

No exported symbol is added, removed or changed. This is a PATCH-class bug fix (`fix(errors)`), not breaking: a caller that branched on `FetchUnreachable` or `ErrTransient` for these failures now gets `FetchUnauthorized` (or `FetchOther` with the status), which is what the type documents.

Not covered, and stated in the spec: on a module version lookup the embedded CUE turns a 403 (and a 404) from the token endpoint into `modregistry.ErrNotFound` and drops the status, so that case stays `FetchNotFound`, as a forbidden tag lookup already does.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `fetch-error-classification`: a token endpoint's answer classifies by its status and never as unreachable; the text fallback's order gains the answered-token-request form ahead of the no-response form.

## Impact

- Code: `opm/errors/classify.go`, `opm/errors/fetch.go` (godoc), `opm/internal/registrytest/status.go` (test fixture), tests in `opm/errors` and `opm/kernel`.
- Public API: none added or removed. Behaviour of `Classify` changes for one family of texts.
- Consumers: the cli (`internal/cuemod/connectivity.go` readers over `Classify`) gets the right answer after a pin bump; its pinned gap test `TestPush_TokenEndpointRefusal_Pinned` must then expect a refused credential, not connectivity. The operator's retry decision (`ErrTransient`) stops retrying a refused token as a network failure.
- Dependencies: none.
