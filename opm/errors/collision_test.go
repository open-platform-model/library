package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContractCollisionsError_Message(t *testing.T) {
	e := &ContractCollisionsError{Contracts: []ContractCollision{
		{Key: "example.test/maj/resources/container@v1", Catalogs: []string{"example.test/maj@v0", "example.test/maj@v1"}},
		{Key: "example.test/maj/traits/expose@v1", Catalogs: []string{"example.test/maj@v0", "example.test/maj@v1", "example.test/maj@v2"}},
	}}
	assert.Equal(t, "2 colliding contract(s):\n"+
		`  contract "example.test/maj/resources/container@v1" is defined by 2 enabled registry entries ("example.test/maj@v0", "example.test/maj@v1"); a contract key must have exactly one enabled definer until side-by-side catalog majors are supported, so disable all but one of these entries`+"\n"+
		`  contract "example.test/maj/traits/expose@v1" is defined by 3 enabled registry entries ("example.test/maj@v0", "example.test/maj@v1", "example.test/maj@v2"); a contract key must have exactly one enabled definer until side-by-side catalog majors are supported, so disable all but one of these entries`,
		e.Error())
}

func TestContractCollisionsError_RoutesThroughJoin(t *testing.T) {
	rows := []ContractCollision{{Key: "k", Catalogs: []string{"a", "b"}}}
	err := fmt.Errorf("render refused: %w", errors.Join(&ContractCollisionsError{Contracts: rows}, errors.New("other")))
	var got *ContractCollisionsError
	require.ErrorAs(t, err, &got)
	assert.Equal(t, rows, got.Contracts, "every row is joined into the gate once, as one cause")
	var nr *NotRoutableError
	assert.False(t, errors.As(err, &nr), "a collision cause never yields a cause of another kind")
}

func TestNotRoutableError_Message(t *testing.T) {
	assert.Equal(t,
		"platform is not routable: its contract inventory reads routable false and reports no over-subscribed or colliding contract to explain it",
		(&NotRoutableError{}).Error())
}

func TestUnresolvedDemand_CollidingCase(t *testing.T) {
	d := UnresolvedDemand{Component: "web", Kind: "trait", FQN: "example.test/maj/traits/backup@v1",
		Colliding: []string{"example.test/maj@v0", "example.test/maj@v1"}}
	e := &UnresolvedDemandsError{Demands: []UnresolvedDemand{d}}
	assert.Equal(t, "1 unresolved demand(s):\n"+
		`  component "web": unresolved trait demand "example.test/maj/traits/backup@v1": defined by more than one enabled registry entry ([example.test/maj@v0 example.test/maj@v1])`,
		e.Error())
	assert.NotContains(t, e.Error(), "no enabled catalog defines")

	d.Unprovided = true
	e = &UnresolvedDemandsError{Demands: []UnresolvedDemand{d}}
	assert.Equal(t, "1 unresolved demand(s):\n"+
		`  component "web": unresolved trait demand "example.test/maj/traits/backup@v1": defined by more than one enabled registry entry ([example.test/maj@v0 example.test/maj@v1]); provider-fulfilled, no provider on this platform`,
		e.Error())
}
