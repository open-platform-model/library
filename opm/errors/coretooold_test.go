package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlatformCoreTooOldError_Message(t *testing.T) {
	e := &PlatformCoreTooOldError{Platform: "prod", Field: "providedBy", Since: "2.0.0-alpha.12"}
	assert.Equal(t,
		`platform "prod" carries no "providedBy" (core derives it from release 2.0.0-alpha.12 on): re-pin opmodel.dev/core in the platform module to v2.0.0-alpha.12 or later`,
		e.Error())
}

func TestPlatformCoreTooOldError_RoutesThroughWrap(t *testing.T) {
	err := fmt.Errorf("render refused before staging: %w", &PlatformCoreTooOldError{Platform: "p", Field: "providedBy", Since: "2.0.0-alpha.12"})
	var got *PlatformCoreTooOldError
	require.True(t, errors.As(err, &got))
	assert.Equal(t, "providedBy", got.Field)
	assert.Equal(t, "2.0.0-alpha.12", got.Since)
}
