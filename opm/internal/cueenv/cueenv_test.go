package cueenv

import (
	"os"
	"strings"
	"testing"

	"cuelang.org/go/mod/modconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// entries returns the values env carries for key, in order; a variable that
// appears twice would be a bug the count exposes.
func entries(env []string, key string) []string {
	var got []string
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			got = append(got, v)
		}
	}
	return got
}

func TestOverride(t *testing.T) {
	cases := []struct {
		name string

		// Process environment before the call ("" = unset).
		registry, cache string

		// Arguments to Override.
		overrideReg, overrideCache string

		wantNil bool

		// Expected CUE_REGISTRY / CUE_CACHE_DIR entries of the result.
		wantRegistry, wantCache []string
	}{
		{name: "nothing overridden is nil", registry: "proc=reg", cache: "/proc/cache", wantNil: true},
		{name: "registry replaces", registry: "proc=reg", overrideReg: "new=reg", wantRegistry: []string{"new=reg"}},
		{name: "registry appends", overrideReg: "new=reg", wantRegistry: []string{"new=reg"}},
		{name: "cache dir replaces", cache: "/proc/cache", overrideCache: "/new/cache", wantCache: []string{"/new/cache"}},
		{name: "cache dir appends", overrideCache: "/new/cache", wantCache: []string{"/new/cache"}},
		{name: "both replace", registry: "proc=reg", cache: "/proc/cache", overrideReg: "new=reg", overrideCache: "/new/cache",
			wantRegistry: []string{"new=reg"}, wantCache: []string{"/new/cache"}},
		{name: "both append", overrideReg: "new=reg", overrideCache: "/new/cache",
			wantRegistry: []string{"new=reg"}, wantCache: []string{"/new/cache"}},
		{name: "registry override keeps the process cache dir", cache: "/proc/cache", overrideReg: "new=reg",
			wantRegistry: []string{"new=reg"}, wantCache: []string{"/proc/cache"}},
		{name: "cache override keeps the process registry", registry: "proc=reg", overrideCache: "/new/cache",
			wantRegistry: []string{"proc=reg"}, wantCache: []string{"/new/cache"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setOrUnset(t, "CUE_REGISTRY", c.registry)
			setOrUnset(t, "CUE_CACHE_DIR", c.cache)

			env := Override(c.overrideReg, c.overrideCache)
			if c.wantNil {
				assert.Nil(t, env, "no override means nil, so cue/load reads the process environment itself")
				return
			}
			require.NotNil(t, env)
			assert.Equal(t, c.wantRegistry, entries(env, "CUE_REGISTRY"))
			assert.Equal(t, c.wantCache, entries(env, "CUE_CACHE_DIR"))

			// The process environment is never written.
			assert.Equal(t, c.registry, os.Getenv("CUE_REGISTRY"))
			assert.Equal(t, c.cache, os.Getenv("CUE_CACHE_DIR"))
		})
	}
}

// The remaining process variables ride along untouched, so an override
// changes exactly the variables it names.
func TestOverride_PreservesOtherVariables(t *testing.T) {
	t.Setenv("OPM_CUEENV_PROBE", "kept")
	env := Override("new=reg", "")
	assert.Equal(t, []string{"kept"}, entries(env, "OPM_CUEENV_PROBE"))
}

// A nil Env is read by modconfig as the process environment: an invalid
// CUE_REGISTRY in the process fails NewRegistry with Env nil, and an
// override slice replacing it succeeds. This pins the semantics Override
// relies on for its nil return against the pinned cuelang.org/go.
func TestOverride_NilMeansProcessEnvironmentForModconfig(t *testing.T) {
	t.Setenv("CUE_REGISTRY", "not a registry mapping !!!")
	_, err := modconfig.NewRegistry(&modconfig.Config{Env: Override("", "")})
	require.Error(t, err, "a nil Env must make modconfig read the (invalid) process CUE_REGISTRY")

	_, err = modconfig.NewRegistry(&modconfig.Config{Env: Override("example.com=localhost:5000", "")})
	require.NoError(t, err, "the override slice replaces the invalid process value")
}

// setOrUnset sets key for the test, or unsets it when value is empty, and
// restores the original value at cleanup either way.
func setOrUnset(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
	if value == "" {
		require.NoError(t, os.Unsetenv(key))
	}
}
