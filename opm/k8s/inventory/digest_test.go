package inventory_test

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/k8s/inventory"
)

// goldenInventoryDigest is the inventory digest of inventoryFixture. Every
// frontend stores this kind of value. Changing it changes every stored
// inventory digest: that needs a new tag line in the encoding and a migration
// note in each frontend (0012:D7:R4), never only a new constant here.
const goldenInventoryDigest = "sha256:0d6041a9bb58abb677bac0d2ab3d0461dcebf9a6e8c5c325fbfc41f6cd46acc3"

// The entries of the encoding fixture (design KI7): a core-group,
// cluster-scoped entry with empty group and namespace, the case the two
// frontends' JSON digests disagreed on; a namespaced apps entry; and an entry
// with an empty component. Four pairs make each sort field decide an order
// that the fields after it would decide the other way, so a sort with any
// two fields swapped or any field dropped gives other bytes: two Deployments
// whose namespace order and name order disagree; two ConfigMaps in one
// namespace whose name order and component order disagree; two Secrets with
// one identity whose component order and version order disagree; and two
// Services that differ only in version, given in the wrong order.
var (
	fixtureNamespace  = inventory.Entry{Kind: "Namespace", Name: "team", Version: "v1", Component: "ns"}
	fixtureDeployment = inventory.Entry{Group: "apps", Kind: "Deployment", Namespace: "team", Name: "web", Version: "v1", Component: "web"}
	fixtureConfigMap  = inventory.Entry{Kind: "ConfigMap", Namespace: "team", Name: "app", Version: "v1"}
	fixtureDeployAZ   = inventory.Entry{Group: "apps", Kind: "Deployment", Namespace: "a", Name: "z", Version: "v1", Component: "web"}
	fixtureDeployBY   = inventory.Entry{Group: "apps", Kind: "Deployment", Namespace: "b", Name: "y", Version: "v1", Component: "web"}
	fixtureConfigAZ   = inventory.Entry{Kind: "ConfigMap", Namespace: "team", Name: "a", Version: "v1", Component: "z"}
	fixtureConfigBY   = inventory.Entry{Kind: "ConfigMap", Namespace: "team", Name: "b", Version: "v1", Component: "y"}
	fixtureSecretAV2  = inventory.Entry{Kind: "Secret", Namespace: "team", Name: "s", Version: "v2", Component: "a"}
	fixtureSecretBV1  = inventory.Entry{Kind: "Secret", Namespace: "team", Name: "s", Version: "v1", Component: "b"}
	fixtureServiceV1  = inventory.Entry{Kind: "Service", Namespace: "team", Name: "web", Version: "v1", Component: "web"}
	fixtureServiceV2  = inventory.Entry{Kind: "Service", Namespace: "team", Name: "web", Version: "v2", Component: "web"}
)

// inventoryFixture is the fixture out of its sorted order. Each pair is given
// in the order its tie-breaking field would not choose.
func inventoryFixture() []inventory.Entry {
	return []inventory.Entry{
		fixtureDeployment, fixtureSecretBV1, fixtureServiceV2, fixtureConfigBY, fixtureDeployBY,
		fixtureNamespace, fixtureSecretAV2, fixtureServiceV1, fixtureConfigMap, fixtureConfigAZ, fixtureDeployAZ,
	}
}

// lengthPrefixed writes the KI4 encoding of one field: its byte length as 8
// big-endian bytes, then its bytes. Written out by hand rather than shared
// with the package, so that the test states the definition.
func lengthPrefixed(s string) []byte {
	n := uint64(len(s))
	b := []byte{byte(n >> 56), byte(n >> 48), byte(n >> 40), byte(n >> 32), byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
	return append(b, s...)
}

func encodedEntry(group, kind, namespace, name, version, component string) []byte {
	var b []byte
	for _, f := range []string{group, kind, namespace, name, version, component} {
		b = append(b, lengthPrefixed(f)...)
	}
	return b
}

func sha256Digest(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// kubernetes-tier: "The encoding is the one defined" (inventory digest).
func TestDigest_EncodingIsTheOneDefined(t *testing.T) {
	// Sorted by group, kind, namespace, name, component, version: the
	// core-group entries first (the ConfigMaps by name, so component z
	// before component y; Namespace; the two Secrets by component, so v2
	// before v1; the two Services by version), then apps (by namespace, so
	// name z before name y).
	want := []byte("opm-inventory-v1\n")
	want = append(want, encodedEntry("", "ConfigMap", "team", "a", "v1", "z")...)
	want = append(want, encodedEntry("", "ConfigMap", "team", "app", "v1", "")...)
	want = append(want, encodedEntry("", "ConfigMap", "team", "b", "v1", "y")...)
	want = append(want, encodedEntry("", "Namespace", "", "team", "v1", "ns")...)
	want = append(want, encodedEntry("", "Secret", "team", "s", "v2", "a")...)
	want = append(want, encodedEntry("", "Secret", "team", "s", "v1", "b")...)
	want = append(want, encodedEntry("", "Service", "team", "web", "v1", "web")...)
	want = append(want, encodedEntry("", "Service", "team", "web", "v2", "web")...)
	want = append(want, encodedEntry("apps", "Deployment", "a", "z", "v1", "web")...)
	want = append(want, encodedEntry("apps", "Deployment", "b", "y", "v1", "web")...)
	want = append(want, encodedEntry("apps", "Deployment", "team", "web", "v1", "web")...)

	got := inventory.Digest(inventoryFixture())
	assert.Equal(t, sha256Digest(want), got, "the digest hashes the encoding written out above")
	assert.Equal(t, goldenInventoryDigest, got, "the stored value of the fixture")
}

// kubernetes-tier: "Input order does not matter".
func TestDigest_InputOrderDoesNotMatter(t *testing.T) {
	want := inventory.Digest(inventoryFixture())
	for _, perm := range orders(inventoryFixture()) {
		assert.Equal(t, want, inventory.Digest(perm), "%v", perm)
	}
}

// kubernetes-tier: "Every field counts".
func TestDigest_EveryFieldCounts(t *testing.T) {
	base := inventoryFixture()
	baseDigest := inventory.Digest(base)
	edit := func(edit func(*inventory.Entry)) []inventory.Entry {
		out := slices.Clone(base)
		edit(&out[0])
		return out
	}
	cases := map[string][]inventory.Entry{
		"group":     edit(func(e *inventory.Entry) { e.Group = "extensions" }),
		"kind":      edit(func(e *inventory.Entry) { e.Kind = "StatefulSet" }),
		"namespace": edit(func(e *inventory.Entry) { e.Namespace = "other" }),
		"name":      edit(func(e *inventory.Entry) { e.Name = "api" }),
		"version":   edit(func(e *inventory.Entry) { e.Version = "v2" }),
		"component": edit(func(e *inventory.Entry) { e.Component = "api" }),
		"added":     append(slices.Clone(base), inventory.Entry{Kind: "Service", Namespace: "team", Name: "api", Version: "v1"}),
		"removed":   base[1:],
		// Entries are not de-duplicated: a repeated entry is a different
		// inventory (0012:D7:R3).
		"duplicated": append(slices.Clone(base), base[0]),
	}
	for name, entries := range cases {
		t.Run(name, func(t *testing.T) {
			assert.NotEqual(t, baseDigest, inventory.Digest(entries))
		})
	}

	// A byte moving across a field boundary: the length prefixes are what
	// keep these two apart.
	t.Run("byte across a field boundary", func(t *testing.T) {
		a := []inventory.Entry{{Group: "ab", Kind: "", Name: "x"}}
		b := []inventory.Entry{{Group: "a", Kind: "b", Name: "x"}}
		assert.NotEqual(t, inventory.Digest(a), inventory.Digest(b))
	})
}

// kubernetes-tier: "Empty and nil inventories agree".
func TestDigest_EmptyAndNilAgree(t *testing.T) {
	want := sha256Digest([]byte("opm-inventory-v1\n"))
	assert.Equal(t, want, inventory.Digest(nil))
	assert.Equal(t, want, inventory.Digest([]inventory.Entry{}))
}

func TestDigest_InputUnchanged(t *testing.T) {
	in := inventoryFixture()
	before := slices.Clone(in)
	_ = inventory.Digest(in)
	assert.Equal(t, before, in)
}

func TestDigest_Form(t *testing.T) {
	got := inventory.Digest(inventoryFixture())
	require.Len(t, got, len("sha256:")+64)
	assert.Regexp(t, `^sha256:[0-9a-f]{64}$`, got)
}

// orders returns in, its reverse, every rotation of both, and 200 shuffles
// from a fixed seed: enough orders to put every pair both ways round many
// times, without the factorial cost of every permutation.
func orders[T any](in []T) [][]T {
	var out [][]T
	rev := slices.Clone(in)
	slices.Reverse(rev)
	for _, base := range [][]T{in, rev} {
		for i := range base {
			out = append(out, append(slices.Clone(base[i:]), base[:i]...))
		}
	}
	r := rand.New(rand.NewPCG(1, 2))
	for range 200 {
		s := slices.Clone(in)
		r.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
		out = append(out, s)
	}
	return out
}
