package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlatformCoreTooOldError_Message(t *testing.T) {
	e := &PlatformCoreTooOldError{Platform: "prod", Field: "providedBy", Since: "2.0.0-alpha.12", Require: "2.0.0-alpha.12"}
	assert.Equal(t,
		`platform "prod" carries no "providedBy" (core derives it from release 2.0.0-alpha.12 on): re-pin opmodel.dev/core in the platform module to v2.0.0-alpha.12 or later`,
		e.Error())
}

// An older missing field keeps its own Since as data, but the re-pin target
// is the floor the kernel enforces, so one re-pin is enough.
func TestPlatformCoreTooOldError_RepinTargetIsTheFloor(t *testing.T) {
	e := &PlatformCoreTooOldError{Platform: "alpha9", Field: "comparable", Since: "2.0.0-alpha.10", Require: "2.0.0-alpha.12"}
	assert.Equal(t,
		`platform "alpha9" carries no "comparable" (core derives it from release 2.0.0-alpha.10 on): re-pin opmodel.dev/core in the platform module to v2.0.0-alpha.12 or later`,
		e.Error())
}

func TestPlatformCoreTooOldError_UnnamedPlatform(t *testing.T) {
	e := &PlatformCoreTooOldError{Field: "providedBy", Since: "2.0.0-alpha.12", Require: "2.0.0-alpha.12"}
	assert.Equal(t,
		`platform <unnamed> carries no "providedBy" (core derives it from release 2.0.0-alpha.12 on): re-pin opmodel.dev/core in the platform module to v2.0.0-alpha.12 or later`,
		e.Error())
}

func TestPlatformCoreTooOldError_EmptyRequireFallsBackToSince(t *testing.T) {
	e := &PlatformCoreTooOldError{Platform: "p", Field: "comparable", Since: "2.0.0-alpha.10"}
	assert.Contains(t, e.Error(), "to v2.0.0-alpha.10 or later")
}

func TestPlatformCoreTooOldError_RoutesThroughWrap(t *testing.T) {
	err := fmt.Errorf("render refused before staging: %w", &PlatformCoreTooOldError{Platform: "p", Field: "providedBy", Since: "2.0.0-alpha.12", Require: "2.0.0-alpha.12"})
	var got *PlatformCoreTooOldError
	require.True(t, errors.As(err, &got))
	assert.Equal(t, "providedBy", got.Field)
	assert.Equal(t, "2.0.0-alpha.12", got.Since)
	assert.Equal(t, "2.0.0-alpha.12", got.Require)
}
