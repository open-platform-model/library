package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

func TestMaterializeError_Format(t *testing.T) {
	cause := errors.New("boom")

	tests := []struct {
		name string
		err  *oerrors.MaterializeError
		want string
	}{
		{
			name: "subscription and version",
			err:  &oerrors.MaterializeError{Kind: oerrors.MaterializeKindCatalog, Subscription: "test.example/cat", Version: "v0.1.0", Cause: cause},
			want: `materialize catalog: subscription "test.example/cat" at version "v0.1.0": boom`,
		},
		{
			name: "subscription only",
			err:  &oerrors.MaterializeError{Kind: oerrors.MaterializeKindCatalog, Subscription: "test.example/cat", Cause: cause},
			want: `materialize catalog: subscription "test.example/cat": boom`,
		},
		{
			name: "core-schema, no subscription",
			err:  &oerrors.MaterializeError{Kind: oerrors.MaterializeKindCoreSchema, Cause: cause},
			want: `materialize core-schema: boom`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestMaterializeError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := &oerrors.MaterializeError{Kind: oerrors.MaterializeKindCatalog, Subscription: "p", Cause: cause}

	require.Equal(t, cause, errors.Unwrap(err), "Unwrap reaches the cause")
	assert.True(t, errors.Is(err, cause), "errors.Is traverses the wrapped cause")

	var me *oerrors.MaterializeError
	require.True(t, errors.As(err, &me), "errors.As extracts *MaterializeError")
	assert.Equal(t, oerrors.MaterializeKindCatalog, me.Kind)
}
