package api

import (
	"errors"
	"io/fs"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/pkg/apiversion"
)

// stubBinding is a minimal Binding for registry tests. Methods other than
// Version are unused by these tests and panic if hit.
type stubBinding struct {
	v apiversion.Version
}

func (s *stubBinding) Version() apiversion.Version { return s.v }
func (s *stubBinding) Paths() Paths                { panic("unused in registry tests") }
func (s *stubBinding) DecodeModuleMetadata(cue.Value) (*ModuleMetadata, error) {
	panic("unused in registry tests")
}
func (s *stubBinding) DecodeReleaseMetadata(cue.Value) (*ReleaseMetadata, error) {
	panic("unused in registry tests")
}
func (s *stubBinding) DecodeProviderMetadata(cue.Value, string) (*ProviderMetadata, error) {
	panic("unused in registry tests")
}
func (s *stubBinding) DecodePlatformMetadata(cue.Value) (*PlatformMetadata, error) {
	panic("unused in registry tests")
}
func (s *stubBinding) BuildTransformerContext(
	*cue.Context, ReleaseView, string, cue.Value, string,
) (cue.Value, []string, error) {
	panic("unused in registry tests")
}
func (s *stubBinding) EmbeddedSchema() fs.FS { return nil }
func (s *stubBinding) SchemaValue(*cue.Context) (cue.Value, error) {
	panic("unused in registry tests")
}

// Compile-time assertion: stubBinding satisfies the Binding interface,
// including the SchemaValue method added by add-release-synth-helper.
var _ Binding = (*stubBinding)(nil)

// withCleanRegistry runs fn against a freshly cleared registry and restores
// the prior state on return. Lets tests register/unregister without leaking
// state across test functions or affecting bindings registered via init() in
// production builds.
func withCleanRegistry(t *testing.T, fn func()) {
	t.Helper()
	registryMu.Lock()
	saved := registry
	registry = map[apiversion.Version]Binding{}
	registryMu.Unlock()
	t.Cleanup(func() {
		registryMu.Lock()
		registry = saved
		registryMu.Unlock()
	})
	fn()
}

func TestRegisterAndLookup(t *testing.T) {
	withCleanRegistry(t, func() {
		stub := &stubBinding{v: apiversion.V1alpha2}
		Register(stub)

		got, err := Lookup(apiversion.V1alpha2)
		require.NoError(t, err)
		assert.Same(t, stub, got)
	})
}

func TestLookupMiss(t *testing.T) {
	withCleanRegistry(t, func() {
		got, err := Lookup(apiversion.V1alpha2)
		assert.Nil(t, got)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion), "want ErrUnknownAPIVersion, got %v", err)
	})
}

func TestRegisterDuplicatePanics(t *testing.T) {
	withCleanRegistry(t, func() {
		first := &stubBinding{v: apiversion.V1alpha2}
		second := &stubBinding{v: apiversion.V1alpha2}
		Register(first)

		var recovered any
		func() {
			defer func() { recovered = recover() }()
			Register(second)
		}()
		require.NotNil(t, recovered, "Register should panic on duplicate")
		msg, ok := recovered.(string)
		require.True(t, ok, "panic value should be string, got %T", recovered)
		assert.Contains(t, msg, "duplicate registration")
		assert.Contains(t, msg, string(apiversion.V1alpha2))
	})
}

func TestRegisterNilPanics(t *testing.T) {
	withCleanRegistry(t, func() {
		defer func() {
			r := recover()
			require.NotNil(t, r, "Register(nil) should panic")
		}()
		Register(nil)
	})
}

func TestRegisterEmptyVersionPanics(t *testing.T) {
	withCleanRegistry(t, func() {
		defer func() {
			r := recover()
			require.NotNil(t, r, "Register(empty version) should panic")
		}()
		Register(&stubBinding{v: ""})
	})
}

func TestForEndToEnd(t *testing.T) {
	withCleanRegistry(t, func() {
		stub := &stubBinding{v: apiversion.V1alpha2}
		Register(stub)

		ctx := cuecontext.New()
		v := ctx.CompileString(`apiVersion: "opmodel.dev/v1alpha2"`)
		require.NoError(t, v.Err())

		got, err := For(v)
		require.NoError(t, err)
		assert.Same(t, stub, got)
	})
}

func TestForUnknown(t *testing.T) {
	withCleanRegistry(t, func() {
		ctx := cuecontext.New()
		v := ctx.CompileString(`apiVersion: "opmodel.dev/v9beta42"`)
		require.NoError(t, v.Err())

		got, err := For(v)
		assert.Nil(t, got)
		require.Error(t, err)
		assert.True(t, errors.Is(err, apiversion.ErrUnknownAPIVersion))
	})
}
