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
	// No alternatives, no defining catalog: nothing enabled lists the key.
	bare := &oerrors.UnresolvedDemandsError{Demands: []oerrors.UnresolvedDemand{
		{Component: "web", FQN: "example.com/r/volume@v1", Kind: "resource"},
	}}
	assert.Contains(t, bare.Error(), "1 unresolved demand(s)")
	assert.Contains(t, bare.Error(), `component "web"`)
	assert.Contains(t, bare.Error(), "unresolved resource demand")
	assert.Contains(t, bare.Error(), "no enabled catalog defines this contract")

	// Alternatives present: the 0010:D4 different-apiVersion diagnostic.
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

// single-build-render spec, "Unresolved demands are diagnosed with
// alternatives": the row words three cases, and the defining catalog
// (0015:D18) is named wherever an enabled catalog lists the key.
func TestUnresolvedDemand_ThreeCases(t *testing.T) {
	const cat = "example.com/catalogs/main@v1"
	cases := []struct {
		name   string
		row    oerrors.UnresolvedDemand
		want   string
		absent string
	}{
		{
			name:   "defined by a catalog, implemented by nothing",
			row:    oerrors.UnresolvedDemand{Component: "web", FQN: "example.com/t/backup@v1", Kind: "trait", DefinedBy: cat},
			want:   `unresolved trait demand "example.com/t/backup@v1": defined by "example.com/catalogs/main@v1" and nothing on this platform implements it`,
			absent: "no enabled catalog defines",
		},
		{
			name:   "implemented at a different apiVersion, defining catalog named",
			row:    oerrors.UnresolvedDemand{Component: "web", FQN: "example.com/r/volume@v1", Kind: "resource", DefinedBy: cat, Alternatives: []string{"example.com/r/volume@v2"}},
			want:   `defined by "example.com/catalogs/main@v1", implemented at a different apiVersion (alternatives: [example.com/r/volume@v2])`,
			absent: "nothing on this platform implements",
		},
		{
			name:   "implemented at a different apiVersion, no defining catalog",
			row:    oerrors.UnresolvedDemand{Component: "web", FQN: "example.com/r/volume@v1", Kind: "resource", Alternatives: []string{"example.com/r/volume@v2"}},
			want:   `unresolved resource demand "example.com/r/volume@v1": implemented at a different apiVersion (alternatives: [example.com/r/volume@v2])`,
			absent: "defined by",
		},
		{
			name:   "no enabled catalog defines it",
			row:    oerrors.UnresolvedDemand{Component: "web", FQN: "example.com/r/volume@v1", Kind: "resource"},
			want:   `unresolved resource demand "example.com/r/volume@v1": no enabled catalog defines this contract`,
			absent: "defined by",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := (&oerrors.UnresolvedDemandsError{Demands: []oerrors.UnresolvedDemand{tc.row}}).Error()
			assert.Contains(t, msg, tc.want)
			assert.NotContains(t, msg, tc.absent)
		})
	}
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
