package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	oerrors "github.com/open-platform-model/library/pkg/errors"
)

func TestSentinelErrors(t *testing.T) {
	assert.NotEqual(t, oerrors.ErrValidation, oerrors.ErrConnectivity)
	assert.NotEqual(t, oerrors.ErrValidation, oerrors.ErrPermission)
	assert.NotEqual(t, oerrors.ErrValidation, oerrors.ErrNotFound)
}

func TestWrap(t *testing.T) {
	wrapped := oerrors.Wrap(oerrors.ErrValidation, "schema check failed")

	assert.True(t, errors.Is(wrapped, oerrors.ErrValidation))
	assert.Contains(t, wrapped.Error(), "schema check failed")
}
