package inventory_test

import (
	"bytes"
	"fmt"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/k8s/object"
)

// goldenRenderDigest is the render digest of the render fixture. Every
// frontend stores this kind of value. Changing it changes every stored render
// digest: that needs a new tag line in the encoding and a migration note in
// each frontend, never only a new constant here.
const goldenRenderDigest = "sha256:08ee7ae343d45ae939f98ff7328986866cc633c2c4f285b9917b658497fdfe68"

// renderFixture describes the objects of the render encoding fixture (design
// KI7): a Deployment carrying the managed-by label, an integer above 2^53 and
// an annotation with U+2028 (which Go's encoder always escapes), a core-group
// Service and a cluster-scoped Namespace, plus two Services whose namespace
// order and name order disagree, so the fixture pins namespace before name.
type renderFixture struct {
	managedBy *string // nil: the Deployment carries no managed-by label
	nameLabel string  // the Deployment's app.kubernetes.io/name value
	bigInt    string  // the Deployment's spec.progressDeadlineSeconds literal
}

func defaultFixture(managedBy string) renderFixture {
	return renderFixture{managedBy: &managedBy, nameLabel: "web", bigInt: "9007199254740993"}
}

// export builds the fixture through object.Export, so the digest is tested on
// real export bytes, and returns it out of its sorted order.
func (f renderFixture) export(t *testing.T) []object.Exported {
	t.Helper()
	managedBy := ""
	if f.managedBy != nil {
		managedBy = fmt.Sprintf("%q: %q\n", "app.kubernetes.io/managed-by", *f.managedBy)
	}
	deployment := fmt.Sprintf(`{
	apiVersion: "apps/v1"
	kind:       "Deployment"
	metadata: {
		name:      "web"
		namespace: "team"
		labels: {
			%s
			"app.kubernetes.io/name": %q
		}
		annotations: note: "a<b&c\u2028"
	}
	spec: {
		replicas:                2
		progressDeadlineSeconds: %s
	}
}`, managedBy, f.nameLabel, f.bigInt)
	ctx := cuecontext.New()
	mk := func(src string) *object.Resource {
		v := ctx.CompileString(src)
		require.NoError(t, v.Err())
		return &object.Resource{Value: v, Instance: "inst", Component: "web", Transformer: "opmodel.dev/x/t@1"}
	}
	out, err := object.Export([]*object.Resource{
		mk(deployment),
		mk(`{apiVersion: "v1", kind: "Service", metadata: {name: "web", namespace: "team"}, spec: ports: [{port: 80}]}`),
		mk(`{apiVersion: "v1", kind: "Namespace", metadata: name: "team"}`),
		mk(`{apiVersion: "v1", kind: "Service", metadata: {name: "y", namespace: "b"}}`),
		mk(`{apiVersion: "v1", kind: "Service", metadata: {name: "z", namespace: "a"}}`),
	})
	require.NoError(t, err)
	return out
}

func renderDigest(t *testing.T, objs []object.Exported) string {
	t.Helper()
	d, err := inventory.RenderDigest(objs)
	require.NoError(t, err)
	return d
}

// kubernetes-tier: "The two runtimes digest one render equally", second
// half. The kernel half is TestRender_RuntimeNameReachesOnlyManagedBy in
// opm/kernel: a runtime name changes only the managed-by value, and this test
// shows that the digest ignores exactly that value.
func TestRenderDigest_TheTwoRuntimesDigestEqually(t *testing.T) {
	cli := renderDigest(t, defaultFixture("opm-cli").export(t))
	ctrl := renderDigest(t, defaultFixture("opm-controller").export(t))
	assert.Equal(t, cli, ctrl)
}

// kubernetes-tier: "Any other label counts".
func TestRenderDigest_AnyOtherLabelCounts(t *testing.T) {
	other := defaultFixture("opm-cli")
	other.nameLabel = "api"
	assert.NotEqual(t,
		renderDigest(t, defaultFixture("opm-cli").export(t)),
		renderDigest(t, other.export(t)))
}

// kubernetes-tier: "Adding or removing the managed-by label counts".
func TestRenderDigest_ManagedByPresenceCounts(t *testing.T) {
	without := defaultFixture("opm-cli")
	without.managedBy = nil
	assert.NotEqual(t,
		renderDigest(t, defaultFixture("opm-cli").export(t)),
		renderDigest(t, without.export(t)))
}

// kubernetes-tier: "Large integers are not rounded". Both values decode to
// the same float64, so a digest over Exported.Object would merge them.
func TestRenderDigest_LargeIntegersAreNotRounded(t *testing.T) {
	smaller := defaultFixture("opm-cli")
	smaller.bigInt = "9007199254740992"
	assert.NotEqual(t,
		renderDigest(t, defaultFixture("opm-cli").export(t)),
		renderDigest(t, smaller.export(t)))
}

// kubernetes-tier: "The encoding is the one defined" (render digest).
func TestRenderDigest_EncodingIsTheOneDefined(t *testing.T) {
	// Sorted by group, kind, namespace, name: the core group first
	// (Namespace before the Services, the Services by namespace, so name z
	// before name y), then apps. Keys sorted, the managed-by value blanked,
	// numbers as written, no HTML escaping, U+2028 escaped as Go's encoder
	// writes it, one newline each.
	want := []byte("opm-render-v1\n" +
		`{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"team"}}` + "\n" +
		`{"apiVersion":"v1","kind":"Service","metadata":{"name":"z","namespace":"a"}}` + "\n" +
		`{"apiVersion":"v1","kind":"Service","metadata":{"name":"y","namespace":"b"}}` + "\n" +
		`{"apiVersion":"v1","kind":"Service","metadata":{"name":"web","namespace":"team"},"spec":{"ports":[{"port":80}]}}` + "\n" +
		`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{"note":"a<b&c\u2028"},"labels":{"app.kubernetes.io/managed-by":"","app.kubernetes.io/name":"web"},"name":"web","namespace":"team"},"spec":{"progressDeadlineSeconds":9007199254740993,"replicas":2}}` + "\n")

	got := renderDigest(t, defaultFixture("opm-cli").export(t))
	assert.Equal(t, sha256Digest(want), got, "the digest hashes the encoding written out above")
	assert.Equal(t, goldenRenderDigest, got, "the stored value of the fixture")
}

// kubernetes-tier: "The input is not changed".
func TestRenderDigest_InputIsNotChanged(t *testing.T) {
	objs := defaultFixture("opm-cli").export(t)
	type snapshot struct {
		json   []byte
		object map[string]any
	}
	before := make([]snapshot, len(objs))
	for i, o := range objs {
		before[i] = snapshot{json: bytes.Clone(o.JSON), object: o.Object.DeepCopy().Object}
	}
	_ = renderDigest(t, objs)
	for i, o := range objs {
		assert.Equal(t, before[i].json, o.JSON, "object %d: JSON unchanged", i)
		assert.Equal(t, before[i].object, o.Object.Object, "object %d: Object unchanged", i)
	}
	assert.Equal(t, "opm-cli", objs[0].Object.GetLabels()["app.kubernetes.io/managed-by"], "the managed-by value is still there")
}

// kubernetes-tier: "A malformed object fails with its position".
func TestRenderDigest_MalformedObjectFailsWithItsPosition(t *testing.T) {
	good := object.Exported{JSON: []byte(`{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"team"}}`)}
	cases := map[string]string{
		"a list":        `[{"kind":"Namespace"}]`,
		"null":          `null`,
		"invalid JSON":  `{"kind":`,
		"trailing data": `{"kind":"Namespace"} {}`,
		"a string":      `"Namespace"`,
	}
	for name, bad := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := inventory.RenderDigest([]object.Exported{good, {JSON: []byte(bad)}, good})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "render digest: object 1:")
		})
	}
}

func TestRenderDigest_InputOrderDoesNotMatter(t *testing.T) {
	objs := defaultFixture("opm-cli").export(t)
	want := renderDigest(t, objs)
	for _, perm := range permutations(objs) {
		assert.Equal(t, want, renderDigest(t, perm))
	}
}

// Two objects with one identity are ordered by their encoded bytes, so input
// order does not reach the digest even then.
func TestRenderDigest_SameIdentityOrderedByBytes(t *testing.T) {
	a := object.Exported{JSON: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"x"},"data":{"k":"a"}}`)}
	b := object.Exported{JSON: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"x"},"data":{"k":"b"}}`)}
	noMeta := object.Exported{JSON: []byte(`{"kind":7}`)}
	assert.Equal(t,
		renderDigest(t, []object.Exported{a, b, noMeta}),
		renderDigest(t, []object.Exported{noMeta, b, a}))
}

func TestRenderDigest_EmptyAndNilHashTheTagLine(t *testing.T) {
	want := sha256Digest([]byte("opm-render-v1\n"))
	assert.Equal(t, want, renderDigest(t, nil))
	assert.Equal(t, want, renderDigest(t, []object.Exported{}))
}
