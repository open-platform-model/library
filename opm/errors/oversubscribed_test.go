package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverSubscribedContractError_Message(t *testing.T) {
	e := OverSubscribedContractError{
		Key:      "example.test/cat/resources/gateway@v1",
		Catalogs: []string{"example.test/cat@v0", "example.test/cat2@v0"},
	}
	msg := e.Error()
	assert.Contains(t, msg, `contract "example.test/cat/resources/gateway@v1"`)
	assert.Contains(t, msg, `fulfilment "provider"`)
	assert.Contains(t, msg, `2 catalogs ("example.test/cat@v0", "example.test/cat2@v0")`)
	assert.Contains(t, msg, "exactly one provider")
}

func TestOverSubscribedContractError_RoutesThroughJoin(t *testing.T) {
	row := OverSubscribedContractError{Key: "k", Catalogs: []string{"a", "b"}}
	err := fmt.Errorf("render refused: %w", errors.Join(errors.New("other"), row))
	var got OverSubscribedContractError
	require.ErrorAs(t, err, &got)
	assert.Equal(t, row, got)
}
