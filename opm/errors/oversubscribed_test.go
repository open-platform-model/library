package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverSubscribedContractsError_Message(t *testing.T) {
	e := &OverSubscribedContractsError{Contracts: []OverSubscribedContract{{
		Key:      "example.test/cat/resources/gateway@v1",
		Catalogs: []string{"example.test/cat@v0", "example.test/cat2@v0"},
	}}}
	msg := e.Error()
	assert.Contains(t, msg, "1 over-subscribed provider contract(s)")
	assert.Contains(t, msg, `contract "example.test/cat/resources/gateway@v1"`)
	assert.Contains(t, msg, `fulfilment "provider"`)
	assert.Contains(t, msg, `2 catalogs ("example.test/cat@v0", "example.test/cat2@v0")`)
	assert.Contains(t, msg, "exactly one provider")
}

func TestOverSubscribedContractsError_RoutesThroughJoin(t *testing.T) {
	rows := []OverSubscribedContract{{Key: "k", Catalogs: []string{"a", "b"}}, {Key: "k2", Catalogs: []string{"a", "c"}}}
	err := fmt.Errorf("render refused: %w", errors.Join(errors.New("other"), &OverSubscribedContractsError{Contracts: rows}))
	var got *OverSubscribedContractsError
	require.ErrorAs(t, err, &got)
	assert.Equal(t, rows, got.Contracts, "every row is joined into the gate once, as one cause")
}
