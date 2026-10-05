package errors

import (
	"context"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelang.org/go/mod/modregistry"
)

// Classify recognises a registry fetch failure and returns it wrapped in a
// [*FetchError], and an author-defect resolution failure and returns it
// wrapped in a [*ResolutionError]. It takes the raw errors the CUE module
// machinery returns: a registry fetch (mod/modconfig, mod/modregistry), a
// cue/load instance error, and the output of `cue mod tidy`.
//
// It returns nil for nil. It returns err unchanged when the chain already
// holds a *FetchError or a *ResolutionError (so classifying twice changes
// nothing), when the chain holds context.Canceled (the caller's cancellation
// is not a fetch failure), and when it recognises nothing, so any other
// author defect (a syntax error, a conflict) is never wrapped and never
// transient.
//
// It reads the typed chain first: context.DeadlineExceeded, an
// ociregistry.HTTPError status, modregistry.ErrNotFound, the ociregistry
// not-found, unauthorized and denied codes, and net.Error. Only when no typed
// cause is found does it match text, because cue/load flattens the cause of a
// failed import into a string. The fetch forms are matched first, so a
// registry failure anywhere in the text never reads as an author defect; the
// author-defect forms only after. That text fallback is the only place in the
// library that matches the text of a registry or cue/load error. Source:
// 0021:D8:R12.
func Classify(err error) error {
	if err == nil {
		return nil
	}
	var fe *FetchError
	var re *ResolutionError
	if errors.As(err, &fe) || errors.As(err, &re) {
		return err
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	if kind, status, ok := classifyTyped(err); ok {
		return &FetchError{Kind: kind, Status: status, Err: err}
	}
	msg := err.Error()
	if kind, status, ok := classifyText(msg); ok {
		return &FetchError{Kind: kind, Status: status, Err: err}
	}
	if kind, ok := classifyResolutionText(msg); ok {
		return &ResolutionError{Kind: kind, Err: err}
	}
	return err
}

// classifyTyped reads the typed chain, most specific cause first.
func classifyTyped(err error) (FetchKind, int, bool) {
	if errors.Is(err, context.DeadlineExceeded) {
		return FetchUnreachable, 0, true
	}
	var he ociregistry.HTTPError
	if errors.As(err, &he) {
		return kindOfStatus(he.StatusCode()), he.StatusCode(), true
	}
	switch {
	case errors.Is(err, modregistry.ErrNotFound),
		errors.Is(err, ociregistry.ErrNameUnknown),
		errors.Is(err, ociregistry.ErrManifestUnknown),
		errors.Is(err, ociregistry.ErrBlobUnknown):
		return FetchNotFound, 0, true
	case errors.Is(err, ociregistry.ErrUnauthorized),
		errors.Is(err, ociregistry.ErrDenied):
		return FetchUnauthorized, 0, true
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return FetchUnreachable, 0, true
	}
	return 0, 0, false
}

// kindOfStatus maps a registry answer to its kind.
func kindOfStatus(status int) FetchKind {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return FetchUnauthorized
	case http.StatusNotFound:
		return FetchNotFound
	default:
		return FetchOther
	}
}

// The text forms the embedded CUE (v0.17.1) produces when it flattens a fetch
// or resolution failure into a string. Each is pinned by TestCUEFailureForms and
// TestClassify_CUEFailureForms in classify_cue_test.go, which drive the
// embedded CUE; a CUE bump that changes one fails there, and this list moves
// with it.
const (
	// textUnreachable is the OCI client's prefix for a request that got no
	// HTTP response.
	textUnreachable = "cannot do HTTP request"
	// textModuleNotFound is modregistry.ErrNotFound's text, which a 403 or
	// 404 tag lookup also reads as.
	textModuleNotFound = "module not found"
	// textCannotFetch is cue/load's prefix around a fetch whose cause is
	// none of the above.
	textCannotFetch = "cannot fetch "
	// textImportUnprovided is modpkgload's ImportMissingError: no module of
	// the build provides an imported package. textVersionNotProvided takes
	// the exact-version form first, as a fetch failure.
	textImportUnprovided = "cannot find module providing package "
	// textImportAmbiguous is modpkgload's AmbiguousImportError: more than
	// one module of the build provides an imported package.
	textImportAmbiguous = "ambiguous import: "
	// textModuleFileUnparsed is modcache's prefix for a fetched module file
	// that does not parse, which graph expansion wraps.
	textModuleFileUnparsed = "cannot parse module file"
)

// textImportedModuleFileUnparsed matches the direct import path's form of a
// dependency module file that does not parse: cue/load prefixes the parse
// error with the module's coordinate (cue/load modfilecache.go), and
// cue/build wraps it as "import failed". At that site nothing else starts
// with a coordinate, and the file-position form of an import failure
// ("import failed: <file>:3:8: ...") has the position after the version, so
// it never matches. The text after the coordinate is the author's module
// file content and is never read.
var textImportedModuleFileUnparsed = regexp.MustCompile(`import failed: [^\s:@]+@v[0-9]+\.[0-9]+\.[0-9]+[^\s:]*: `)

// textStatus matches an HTTP status as the OCI client writes a registry's
// error answer: ": 503 Service Unavailable: ". The status text must be the
// one net/http gives the code, so a number in a path never matches.
var textStatus = regexp.MustCompile(`(?:^|: )([1-5][0-9]{2}) ([A-Za-z][A-Za-z' -]*): `)

// textVersionNotProvided matches cue/load's "cannot find module providing
// package P" only where P names an exact version, which is a standalone
// path@version load (the schema loader's): there the registry was asked for
// that version and it does not provide the package, a fetch failure.
// Elsewhere the same words report an import no module of the build provides
// (a package missing from the main module's own path, an undeclared module,
// a package missing from a declared dependency). An import path carries at
// most a major version, so those never match here; classifyResolutionText
// types them as ResolutionImportUnprovided.
var textVersionNotProvided = regexp.MustCompile(`cannot find module providing package \S+@v[0-9]+\.[0-9]+\.[0-9]+`)

// classifyText is the fetch half of the text fallback, most specific form
// first. It recognises only a failed registry interaction, and it runs
// before classifyResolutionText, so a fetch form anywhere in the text wins.
// It does not match cue/load's "cannot expand module graph" on its own: that
// prefix also wraps a published dependency whose module file does not parse,
// which is an author defect (ResolutionModuleFileInvalid) and not a fetch, so
// it classifies only through the form it carries.
func classifyText(msg string) (FetchKind, int, bool) {
	if strings.Contains(msg, textUnreachable) {
		return FetchUnreachable, 0, true
	}
	for _, m := range textStatus.FindAllStringSubmatch(msg, -1) {
		status, err := strconv.Atoi(m[1])
		if err != nil || status < 400 || http.StatusText(status) != m[2] {
			continue
		}
		return kindOfStatus(status), status, true
	}
	if strings.Contains(msg, textModuleNotFound) || textVersionNotProvided.MatchString(msg) {
		return FetchNotFound, 0, true
	}
	if strings.Contains(msg, textCannotFetch) {
		return FetchOther, 0, true
	}
	return 0, 0, false
}

// classifyResolutionText is the author-defect half of the text fallback. It
// runs only when classifyText found no fetch form. When a text carries
// several author-defect forms, the first in this order decides.
func classifyResolutionText(msg string) (ResolutionKind, bool) {
	switch {
	case strings.Contains(msg, textImportUnprovided):
		return ResolutionImportUnprovided, true
	case strings.Contains(msg, textImportAmbiguous):
		return ResolutionImportAmbiguous, true
	case strings.Contains(msg, textModuleFileUnparsed), textImportedModuleFileUnparsed.MatchString(msg):
		return ResolutionModuleFileInvalid, true
	}
	return 0, false
}
