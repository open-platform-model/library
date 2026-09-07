package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

func TestUnmatchedComponentsError_MessageShape(t *testing.T) {
	err := &oerrors.UnmatchedComponentsError{Components: []oerrors.UnmatchedComponent{
		{Component: "web", Candidates: []oerrors.CandidateVerdict{
			{Transformer: "cat/transformers/deployment@0.1.0", Matched: false, MissingLabels: []string{"tier"}},
		}},
		{Component: "orphan"},
	}}
	msg := err.Error()
	assert.Contains(t, msg, "2 component(s) have no matching transformer: [web orphan]")
	assert.Contains(t, msg, `component "web"`)
	assert.Contains(t, msg, `transformer "cat/transformers/deployment@0.1.0" did not match`)
	assert.Contains(t, msg, "missing labels:    [tier]")
	assert.Contains(t, msg, `component "orphan"`)
}

func TestUnmatchedComponentsError_CarriesRowsAndWrapsNothing(t *testing.T) {
	components := []oerrors.UnmatchedComponent{
		{Component: "web", Candidates: []oerrors.CandidateVerdict{
			{Transformer: "cat/transformers/deployment@0.1.0", MissingLabels: []string{"tier"}},
		}},
		{Component: "orphan"},
	}
	err := &oerrors.UnmatchedComponentsError{Components: components}

	joined := errors.Join(errors.New("other"), err)
	var uce *oerrors.UnmatchedComponentsError
	require.ErrorAs(t, joined, &uce)
	assert.Equal(t, components, uce.Components, "the cause carries the rows unchanged and in order")

	var terr *oerrors.TransformError
	assert.False(t, errors.As(joined, &terr),
		"an unmatched-components refusal fabricates no TransformError")
}
