package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

func TestUnmatchedComponentsError_MessageAndUnwrap(t *testing.T) {
	err := &oerrors.UnmatchedComponentsError{
		Components: []string{"web", "orphan"},
		Matches: map[string]map[string]oerrors.MatchResult{
			"web": {
				"cat/transformers/deployment@0.1.0": {Matched: false, MissingLabels: []string{"tier"}},
			},
			"orphan": {},
		},
	}
	msg := err.Error()
	assert.Contains(t, msg, "2 component(s) have no matching transformer")
	assert.Contains(t, msg, `component "web"`)
	assert.Contains(t, msg, `transformer "cat/transformers/deployment@0.1.0" did not match`)
	assert.Contains(t, msg, "missing labels:    [tier]")

	joined := errors.Join(errors.New("other"), err)
	var uce *oerrors.UnmatchedComponentsError
	require.ErrorAs(t, joined, &uce)
	assert.Equal(t, []string{"web", "orphan"}, uce.Components)

	var terr *oerrors.TransformError
	require.ErrorAs(t, joined, &terr, "each unmatched component unwraps to a TransformError")
	assert.Equal(t, "web", terr.ComponentName)
	assert.Equal(t, "cat/transformers/deployment@0.1.0", terr.TransformerFQN)
}
