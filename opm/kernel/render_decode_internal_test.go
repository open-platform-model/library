package kernel

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// single-build-render spec, "Matching runs inside the build with verdicts
// as data": a pair is listed as failed when its output's Value.Err() is
// non-nil, an output whose root is incomplete included; a struct or list
// root whose only defect is non-concrete fields is not listed, and neither
// is an output that does not exist. The outputs are shaped by the same
// disjunction core's #transform declares, so a nested conflict reaches the
// root the way it does in a real render.
func TestOutputFailed(t *testing.T) {
	const src = `
#T: output: {...} | [...{...}]
x: bool
cases: {
	healthyStruct:   #T & {output: {a: 1}}
	healthyList:     #T & {output: [{a: 1}, {a: 2}]}
	nestedOpen:      #T & {output: {a: string}}
	nestedOpenList:  #T & {output: [{a: string}]}
	nestedConflict:  #T & {output: {a: {b: 1} & {b: 2}}}
	listConflict:    #T & {output: [{a: 1}, {a: 1 & 2}]}
	rootIncomplete:  #T & {output: [if x {{a: 1}}][0]}
}
`
	// The conflicting cases make the compiled value's own Err() non-nil,
	// so each case is checked through its lookup, as the kernel does.
	v := cuecontext.New().CompileString(src)
	require.True(t, v.LookupPath(cue.ParsePath("cases")).Exists())

	for _, tc := range []struct {
		name   string
		failed bool
	}{
		{"healthyStruct", false},
		{"healthyList", false},
		{"nestedOpen", false},
		{"nestedOpenList", false},
		{"nestedConflict", true},
		{"listConflict", true},
		{"rootIncomplete", true},
		{"missing", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := v.LookupPath(cue.ParsePath("cases." + tc.name + ".output"))
			if tc.name == "missing" {
				require.False(t, out.Exists(), "the missing case looks up a path no case defines")
				require.Error(t, out.Err(), "a non-existent value carries an error, so outputFailed must check Exists")
			}
			assert.Equal(t, tc.failed, outputFailed(out))
		})
	}
}

// The missing-output refusal names the same key the lookup used.
func TestPairKey(t *testing.T) {
	assert.Equal(t, "web :: example.com/tx/deployment@1.0.0",
		pairKey(RenderPair{Component: "web", Transformer: "example.com/tx/deployment@1.0.0"}))
}
