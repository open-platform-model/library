# ADR-014: One text match for a token endpoint's answer

## Status

Accepted (2026-10-08). Records the owner's decision of 2026-10-08 on the change `classify-token-endpoint-refusal`. It is an exception to the strict reading of 0021:D8:R12, recorded in ADR-013, decision d1.

## Context

ADR-013, decision d1 says fetch failures are typed, so the frontends stop matching message text. `opm/errors.Classify` reads the typed error chain first and keeps one text fallback for the forms `cue/load` flattens into a string.

A registry that uses token authentication has a token endpoint. When that endpoint answers the token request with a status, the embedded OCI client returns the answer as the error of the HTTP round trip. Three paths then hold no typed status at all. `modregistry.Client.PutModule` flattens the push error with `%v`. `cue/load` flattens every failed import. A client that refreshes a token it holds gets the answer flattened on purpose inside a `*url.Error`, which is a `net.Error`. On each of them `Classify` read a refused credential (401) or a refused permission (403) as `FetchUnreachable`, which is transient: the cli said "registry unreachable", and a controller would retry a refusal as a network failure.

The only signal left is the text: `cannot do HTTP request: <request>: <code> <status text>` at the end of a line. The alternative that reads no text is a library-owned HTTP transport that observes the token request, with a way to carry its result past `cue/load` and a new exported client constructor for a frontend's own push.

## Decision

`Classify` matches one anchored text pattern for an answered token request, `textTokenAnswer` in `opm/errors/classify.go`, and classifies the failure by its status. The text fallback checks it before the no-response prefix it shares. The typed path checks the same pattern before a `net.Error` becomes `FetchUnreachable`, which is the one place the typed path reads text. No other text match is added by this decision, and a further one needs its own record.

A library-owned registry transport is not built now. It was weighed and set aside: it adds exported surface and a second code path for one misread form.

Three limits are accepted as built. A fetch whose token endpoint answers 403 stays `FetchNotFound`, because CUE's registry client reports a 403 on a version lookup as not found and drops the status. A token endpoint's 404 is `FetchNotFound` with status 404. A token endpoint's 429 is `FetchOther` and is no longer transient.

## Consequences

**Positive:** A refused token is `FetchUnauthorized` with its status on the push, the directory load and the token refresh, with no new exported symbol. The frontends keep one call, `Classify`, and read no text themselves.

**Negative:** The classification of these paths depends on text the embedded CUE and its OCI client produce. A bump that changes the form brings back the old answer. `TestCUEFailureForms` and `TestClassify_CUEFailureForms` drive the real client against local token registries and fail when the form changes.

**Trade-off:** A user whose credential lacks permission on a fetch is still told the module is not found. Changing that needs the transport that this decision sets aside.
