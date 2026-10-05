package object_test

import (
	"encoding/json"
	"errors"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/k8s/object"
)

func exportFixtures(t *testing.T) []*object.Resource {
	t.Helper()
	ctx := cuecontext.New()
	mk := func(src, comp, tr string) *object.Resource {
		v := ctx.CompileString(src)
		require.NoError(t, v.Err())
		return &object.Resource{Value: v, Instance: "inst", Component: comp, Transformer: tr}
	}
	return []*object.Resource{
		mk(deploymentCUE, "web", "opmodel.dev/x/deployment@1"),
		mk(`{apiVersion: "v1", kind: "Service", metadata: {name: "web", namespace: "default"}}`, "web", "opmodel.dev/x/service@1"),
		mk(`{apiVersion: "v1", kind: "Namespace", metadata: name: "team"}`, "ns", "opmodel.dev/x/namespace@1"),
	}
}

// TestExport_OneExportFeedsEveryConsumer covers the kubernetes-tier scenario
// "One export feeds every consumer of the object".
// That each resource is exported from CUE exactly once is not observable from
// the returned values (a second export yields equal bytes); it rests on the
// code of Export, which calls MarshalJSON once and decodes Object from those
// bytes.
func TestExport_OneExportFeedsEveryConsumer(t *testing.T) {
	in := exportFixtures(t)

	got, err := object.Export(in)
	require.NoError(t, err)
	require.Len(t, got, len(in))

	for i, r := range in {
		want, err := r.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, want, got[i].JSON, "JSON of %d", i)

		var obj map[string]any
		require.NoError(t, json.Unmarshal(got[i].JSON, &obj))
		assert.Equal(t, obj, got[i].Object.Object, "object of %d", i)

		assert.Equal(t, r.Instance, got[i].Instance)
		assert.Equal(t, r.Component, got[i].Component)
		assert.Equal(t, r.Transformer, got[i].Transformer)
	}
	assert.Equal(t, "Deployment", got[0].Object.GetKind())
	assert.Equal(t, "Service", got[1].Object.GetKind())
	assert.Equal(t, "Namespace", got[2].Object.GetKind())
}

// TestExport_InputSurvives covers the kubernetes-tier scenario "The input
// survives the export".
func TestExport_InputSurvives(t *testing.T) {
	in := exportFixtures(t)
	before := make([]object.Resource, len(in))
	beforeJSON := make([][]byte, len(in))
	ptrs := make([]*object.Resource, len(in))
	for i, r := range in {
		before[i] = *r
		ptrs[i] = r
		b, err := r.MarshalJSON()
		require.NoError(t, err)
		beforeJSON[i] = b
	}

	_, err := object.Export(in)
	require.NoError(t, err)

	require.Len(t, in, len(before))
	for i, r := range in {
		assert.Same(t, ptrs[i], r, "slot %d replaced", i)
		assert.Equal(t, before[i].Instance, r.Instance)
		assert.Equal(t, before[i].Component, r.Component)
		assert.Equal(t, before[i].Transformer, r.Transformer)
		after, err := r.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, beforeJSON[i], after, "value of %d changed", i)
	}
}

func TestExport_Empty(t *testing.T) {
	got, err := object.Export(nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestExport_MarshalFailure covers the kubernetes-tier scenario "A failing
// export names the resource and the step".
func TestExport_MarshalFailure(t *testing.T) {
	in := exportFixtures(t)
	bad := cuecontext.New().CompileString(`{apiVersion: "v1", kind: "ConfigMap", metadata: name: "half", data: x: string}`)
	in = append(in[:1], append([]*object.Resource{{Value: bad, Component: "cfg"}}, in[1:]...)...)

	got, err := object.Export(in)
	require.Error(t, err)
	assert.Nil(t, got)

	var ee *object.ExportError
	require.True(t, errors.As(err, &ee))
	assert.Equal(t, 1, ee.Index)
	assert.Equal(t, object.ExportMarshal, ee.Step)
	assert.Equal(t, "ConfigMap/half", ee.Resource)
	assert.Contains(t, err.Error(), "ConfigMap/half")
	assert.Contains(t, err.Error(), "resource 1")
	assert.Contains(t, err.Error(), "cue export")
	assert.NotNil(t, errors.Unwrap(err))
	assert.Equal(t, ee.Err, errors.Unwrap(err))
}

// TestExport_DecodeFailure covers the kubernetes-tier scenario "A value that
// is not an object fails at the decode": a list and a string fail in the
// JSON decode, and null, which decodes to a nil map without error, is refused
// so that no Exported carries a nil object map.
func TestExport_DecodeFailure(t *testing.T) {
	for _, src := range []string{`[1]`, `"text"`, `null`} {
		t.Run(src, func(t *testing.T) {
			ok := exportFixtures(t)[:2]
			v := cuecontext.New().CompileString(src)
			require.NoError(t, v.Err())
			in := append(ok, &object.Resource{Value: v, Component: "odd"})

			got, err := object.Export(in)
			require.Error(t, err)
			assert.Nil(t, got)

			var ee *object.ExportError
			require.True(t, errors.As(err, &ee))
			assert.Equal(t, 2, ee.Index)
			assert.Equal(t, object.ExportDecode, ee.Step)
			assert.Contains(t, err.Error(), "json decode")
			assert.Equal(t, ee.Err, errors.Unwrap(err))
		})
	}
}

func TestExport_NilResource(t *testing.T) {
	in := append(exportFixtures(t)[:1], nil)
	_, err := object.Export(in)
	var ee *object.ExportError
	require.True(t, errors.As(err, &ee))
	assert.Equal(t, 1, ee.Index)
	assert.Equal(t, object.ExportMarshal, ee.Step)
}

func TestExportStep_String(t *testing.T) {
	assert.Equal(t, "cue export", object.ExportMarshal.String())
	assert.Equal(t, "json decode", object.ExportDecode.String())
	assert.Equal(t, "ExportStep(7)", object.ExportStep(7).String())
}

func TestExport_ResourceWithoutValue(t *testing.T) {
	in := append(exportFixtures(t)[:1], &object.Resource{Component: "x"})
	_, err := object.Export(in)
	var ee *object.ExportError
	require.True(t, errors.As(err, &ee))
	assert.Equal(t, 1, ee.Index)
	assert.Equal(t, object.ExportMarshal, ee.Step)
	assert.Equal(t, "<no value>", ee.Resource)
}
