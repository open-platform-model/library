package kernel

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

// single-build-render spec, "An unexplained not-routable verdict refuses":
// the catch-all fires only when the decoded routable reads false with no
// collision or over-subscription row, and it is joined after every other
// cause present.
func TestGateErrors_NotRoutableCatchAll(t *testing.T) {
	collision := []oerrors.ContractCollision{{Key: "k", Catalogs: []string{"a@v0", "a@v1"}}}
	over := []oerrors.OverSubscribedContract{{Key: "p", Catalogs: []string{"a@v0", "b@v0"}}}
	unresolved := []oerrors.UnresolvedDemand{{Component: "web", FQN: "r", Kind: "resource"}}
	unmatched := []oerrors.UnmatchedComponent{{Component: "web"}}

	notRoutable := func(err error) bool {
		var nr *oerrors.NotRoutableError
		return errors.As(err, &nr)
	}

	assert.NoError(t, gateErrors(RenderDiagnostics{Routable: true}), "routable with no rows passes")
	assert.True(t, notRoutable(gateErrors(RenderDiagnostics{Routable: false})), "routable false with no row refuses")
	assert.False(t, notRoutable(gateErrors(RenderDiagnostics{Routable: false, Collisions: collision})),
		"a collision row explains routable false")
	assert.False(t, notRoutable(gateErrors(RenderDiagnostics{Routable: false, OverSubscribed: over})),
		"an over-subscription row explains routable false")
	assert.False(t, notRoutable(gateErrors(RenderDiagnostics{Routable: true, Unresolved: unresolved})),
		"an unresolved demand on a routable platform is not the catch-all")

	err := gateErrors(RenderDiagnostics{Routable: false, Unresolved: unresolved, Unmatched: unmatched})
	j, ok := err.(interface{ Unwrap() []error })
	require.True(t, ok)
	causes := j.Unwrap()
	require.Len(t, causes, 3)
	assert.IsType(t, &oerrors.UnresolvedDemandsError{}, causes[0])
	assert.IsType(t, &oerrors.UnmatchedComponentsError{}, causes[1])
	assert.IsType(t, &oerrors.NotRoutableError{}, causes[2], "the catch-all is joined last")
}

// The collision cause is joined first, ahead of every other row-carrying
// cause.
func TestGateErrors_CollisionFirst(t *testing.T) {
	err := gateErrors(RenderDiagnostics{
		Routable:       false,
		Collisions:     []oerrors.ContractCollision{{Key: "k", Catalogs: []string{"a@v0", "a@v1"}}},
		Unresolved:     []oerrors.UnresolvedDemand{{Component: "web", FQN: "r", Kind: "resource"}},
		OverSubscribed: []oerrors.OverSubscribedContract{{Key: "p", Catalogs: []string{"a@v0", "b@v0"}}},
		Unmatched:      []oerrors.UnmatchedComponent{{Component: "web"}},
	})
	j, ok := err.(interface{ Unwrap() []error })
	require.True(t, ok)
	causes := j.Unwrap()
	require.Len(t, causes, 4)
	assert.IsType(t, &oerrors.ContractCollisionsError{}, causes[0])
	assert.IsType(t, &oerrors.UnresolvedDemandsError{}, causes[1])
	assert.IsType(t, &oerrors.OverSubscribedContractsError{}, causes[2])
	assert.IsType(t, &oerrors.UnmatchedComponentsError{}, causes[3])
}
