package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
)

// A row is data: it carries the build's verdict and is never itself an error.
func TestRowsAreNotErrors(t *testing.T) {
	var (
		_ any = oerrors.UnresolvedDemand{}
		_ any = oerrors.UnifyRefusal{}
		_ any = oerrors.UnmatchedComponent{}
		_ any = oerrors.CandidateVerdict{}
		_ any = oerrors.OverSubscribedContract{}
	)
	assert.NotImplements(t, (*error)(nil), oerrors.UnresolvedDemand{})
	assert.NotImplements(t, (*error)(nil), oerrors.UnifyRefusal{})
	assert.NotImplements(t, (*error)(nil), oerrors.UnmatchedComponent{})
	assert.NotImplements(t, (*error)(nil), oerrors.CandidateVerdict{})
	assert.NotImplements(t, (*error)(nil), oerrors.OverSubscribedContract{})
}

func TestUnifyRefusal_CarriesItsConflicts(t *testing.T) {
	r := oerrors.UnifyRefusal{
		Component:   "web",
		Transformer: "cat/transformers/deployment@0.1.0",
		Conflicts:   []string{"example.com/r/container@v0", "example.com/r/volume@v1"},
	}
	assert.Equal(t, []string{"example.com/r/container@v0", "example.com/r/volume@v1"}, r.Conflicts,
		"one row per candidate, listing every FQN it conflicted at")
}

func TestUnresolvedDemandsError_MessageShape(t *testing.T) {
	// No alternatives: the contract is unimplemented on this platform.
	bare := &oerrors.UnresolvedDemandsError{Demands: []oerrors.UnresolvedDemand{
		{Component: "web", FQN: "example.com/r/volume@v1", Kind: "resource"},
	}}
	assert.Contains(t, bare.Error(), "1 unresolved demand(s)")
	assert.Contains(t, bare.Error(), `component "web"`)
	assert.Contains(t, bare.Error(), "unresolved resource demand")
	assert.Contains(t, bare.Error(), "nothing on this platform implements this contract")

	// Alternatives present: the D4 different-apiVersion diagnostic.
	alt := &oerrors.UnresolvedDemandsError{Demands: []oerrors.UnresolvedDemand{{
		Component:    "web",
		FQN:          "example.com/r/volume@v1",
		Kind:         "resource",
		Alternatives: []string{"example.com/r/volume@v2"},
	}}}
	assert.Contains(t, alt.Error(), "implemented at a different apiVersion")
	assert.Contains(t, alt.Error(), "example.com/r/volume@v2")

	// Disqualified candidates are counted.
	disq := &oerrors.UnresolvedDemandsError{Demands: []oerrors.UnresolvedDemand{{
		Component:    "web",
		FQN:          "example.com/t/backup@v1",
		Kind:         "trait",
		Disqualified: []oerrors.UnifyRefusal{{Component: "web", Transformer: "cat/t/backup@0.1.0"}},
	}}}
	assert.Contains(t, disq.Error(), "1 candidate(s) disqualified")
}

func TestUnresolvedDemandsError_CarriesRowsAndWrapsNothing(t *testing.T) {
	demands := []oerrors.UnresolvedDemand{
		{Component: "web", FQN: "example.com/r/a@v1", Kind: "resource"},
		{Component: "db", FQN: "example.com/t/b@v1", Kind: "trait"},
	}
	agg := &oerrors.UnresolvedDemandsError{Demands: demands}
	assert.Contains(t, agg.Error(), "2 unresolved demand(s)")

	joined := errors.Join(errors.New("other"), agg)
	var ude *oerrors.UnresolvedDemandsError
	require.ErrorAs(t, joined, &ude)
	assert.Equal(t, demands, ude.Demands, "the cause carries the rows unchanged and in order")

	var terr *oerrors.TransformError
	assert.False(t, errors.As(joined, &terr), "an unresolved-demands refusal synthesizes no other cause")
}
