package api

import (
	"fmt"
	"sync"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/apiversion"
)

// registry holds the active version-to-binding mapping. It is guarded by a
// RWMutex; reads happen on every render, writes only at init() time.
var (
	registryMu sync.RWMutex
	registry   = map[apiversion.Version]Binding{}
)

// Register installs b under b.Version() in the process-wide registry. It
// panics if a binding for the same version is already registered — this
// surfaces misconfiguration (two packages claiming the same version) at the
// earliest possible point, in the init() chain rather than at first use.
//
// Register is intended to be called from a per-version package's init().
// Callers SHOULD NOT call it dynamically at runtime.
func Register(b Binding) {
	if b == nil {
		panic("api.Register: nil Binding")
	}
	v := b.Version()
	if v == "" {
		panic("api.Register: Binding.Version() returned empty string")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if existing, ok := registry[v]; ok {
		panic(fmt.Sprintf(
			"api.Register: duplicate registration for %q (existing=%T new=%T)",
			v, existing, b,
		))
	}
	registry[v] = b
}

// Lookup returns the binding registered for v, or a non-nil error if none is
// registered. The error wraps apiversion.ErrUnknownAPIVersion so callers can
// match on it with errors.Is.
func Lookup(v apiversion.Version) (Binding, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	b, ok := registry[v]
	if !ok {
		return nil, fmt.Errorf("no binding registered for %q: %w", v, apiversion.ErrUnknownAPIVersion)
	}
	return b, nil
}

// For is a convenience that detects v's apiVersion and looks up the
// corresponding binding in one step. Returns an error wrapping
// apiversion.ErrUnknownAPIVersion if detection fails or no binding is
// registered for the detected version.
func For(v cue.Value) (Binding, error) {
	ver, err := apiversion.Detect(v)
	if err != nil {
		return nil, err
	}
	return Lookup(ver)
}
