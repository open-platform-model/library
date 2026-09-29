package errors

import (
	"fmt"
	"strconv"
)

// PlatformCoreTooOldError reports a platform module pinning a core release
// older than the first one deriving a #Platform.#contracts field the kernel
// reads. Returned (wrapped) by Kernel.Render before staging and by
// Platform.Contracts. The fix is re-pinning opmodel.dev/core in the platform
// module; the kernel never falls back to a count or a verdict of its own.
type PlatformCoreTooOldError struct {
	// Platform is the platform's metadata.name (empty when the value
	// carries none; the message then reads <unnamed>).
	Platform string

	// Field is the missing field: a field under #contracts such as
	// "providedBy", or "#contracts" itself when the value carries no
	// inventory at all.
	Field string

	// Since is the first core release deriving Field, without the "v"
	// prefix, e.g. "2.0.0-alpha.12".
	Since string

	// Require is the oldest core release the kernel accepts today, without
	// the "v" prefix: the release the message tells the caller to re-pin
	// to, so one re-pin clears every missing field at once. The kernel's
	// callers fill it with the floor they enforce (schema.ProvidedBySince);
	// empty, the message falls back to Since.
	Require string
}

func (e *PlatformCoreTooOldError) Error() string {
	name := "<unnamed>"
	if e.Platform != "" {
		name = strconv.Quote(e.Platform)
	}
	require := e.Require
	if require == "" {
		require = e.Since
	}
	return fmt.Sprintf(
		"platform %s carries no %q (core derives it from release %s on): re-pin opmodel.dev/core in the platform module to v%s or later",
		name, e.Field, e.Since, require)
}
