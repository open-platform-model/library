package inventory_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/k8s/inventory"
)

func deploy(ns, name, component string) inventory.Entry {
	return inventory.Entry{Group: "apps", Kind: "Deployment", Namespace: ns, Name: name, Version: "v1", Component: component}
}

// kubernetes-tier: "A component rename is not stale".
func TestStaleSet_ComponentRenameIsNotStale(t *testing.T) {
	got := inventory.StaleSet(
		[]inventory.Entry{deploy("team", "web", "api")},
		[]inventory.Entry{deploy("team", "web", "web")},
	)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

// kubernetes-tier: "An API version change is not stale".
func TestStaleSet_VersionChangeIsNotStale(t *testing.T) {
	hpa := func(version string) inventory.Entry {
		return inventory.Entry{Group: "autoscaling", Kind: "HorizontalPodAutoscaler", Namespace: "team", Name: "web", Version: version, Component: "web"}
	}
	got := inventory.StaleSet([]inventory.Entry{hpa("v2beta2")}, []inventory.Entry{hpa("v2")})
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

// kubernetes-tier: "Removed objects are stale in previous order".
func TestStaleSet_RemovedInPreviousOrder(t *testing.T) {
	a, b, c, d := deploy("team", "a", "x"), deploy("team", "b", "x"), deploy("team", "c", "x"), deploy("team", "d", "x")
	assert.Equal(t, []inventory.Entry{a, c, d}, inventory.StaleSet([]inventory.Entry{a, b, c, d}, []inventory.Entry{b}))
}

// kubernetes-tier: "Nothing stale is an empty list, not nil".
func TestStaleSet_NothingStaleIsEmptyNotNil(t *testing.T) {
	kept := []inventory.Entry{deploy("team", "web", "web")}
	cases := map[string][2][]inventory.Entry{
		"empty previous": {{}, kept},
		"nil previous":   {nil, kept},
		"nil both":       {nil, nil},
		"all kept":       {kept, kept},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			got := inventory.StaleSet(in[0], in[1])
			assert.NotNil(t, got)
			assert.Len(t, got, 0)
		})
	}
}

func TestStaleSet_DuplicatesEachReturned(t *testing.T) {
	a := deploy("team", "a", "x")
	assert.Equal(t, []inventory.Entry{a, a}, inventory.StaleSet([]inventory.Entry{a, a}, nil))
}

func TestStaleSet_InputsUnchanged(t *testing.T) {
	previous := []inventory.Entry{deploy("team", "a", "x"), deploy("team", "b", "x")}
	current := []inventory.Entry{deploy("team", "b", "y")}
	prevCopy, curCopy := slices.Clone(previous), slices.Clone(current)
	_ = inventory.StaleSet(previous, current)
	assert.Equal(t, prevCopy, previous)
	assert.Equal(t, curCopy, current)
}

// cliStaleSet is the cli's rule before 0012:D7, ported as an oracle (cli
// pkg/inventory ComputeStaleSet, then internal/inventory
// ApplyComponentRenameSafetyCheck): a component-aware stale set, then a filter
// that rescues every stale entry some current entry names under another
// component.
func cliStaleSet(previous, current []inventory.Entry) []inventory.Entry {
	stale := make([]inventory.Entry, 0)
	for _, p := range previous {
		found := false
		for _, c := range current {
			if inventory.SameObject(p, c) && p.Component == c.Component {
				found = true
				break
			}
		}
		if !found {
			stale = append(stale, p)
		}
	}
	filtered := make([]inventory.Entry, 0, len(stale))
	for _, s := range stale {
		renamed := false
		for _, c := range current {
			if inventory.SameObject(s, c) && s.Component != c.Component {
				renamed = true
				break
			}
		}
		if !renamed {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// The component-blind rule gives the cli's outcome: dropping its rename filter
// changes no behaviour (0012:D7).
func TestStaleSet_EqualsTheCLIRuleWithItsRenameFilter(t *testing.T) {
	versioned := func(e inventory.Entry, v string) inventory.Entry { e.Version = v; return e }
	previous := []inventory.Entry{
		deploy("team", "renamed", "old"),
		deploy("team", "removed", "web"),
		versioned(deploy("team", "moved-version", "web"), "v1beta1"),
		deploy("team", "kept", "web"),
		{Kind: "Namespace", Name: "team", Version: "v1"},
		{Kind: "ConfigMap", Namespace: "team", Name: "gone", Version: "v1", Component: "config"},
		deploy("other", "kept", "web"),
		deploy("team", "renamed-and-split", "a"),
	}
	current := []inventory.Entry{
		deploy("team", "renamed", "new"),
		deploy("team", "moved-version", "web"),
		deploy("team", "kept", "web"),
		{Kind: "Namespace", Name: "team", Version: "v1", Component: "ns"},
		deploy("team", "renamed-and-split", "b"),
		deploy("team", "renamed-and-split", "c"),
		deploy("team", "added", "web"),
	}
	want := cliStaleSet(previous, current)
	require.Equal(t, []inventory.Entry{
		deploy("team", "removed", "web"),
		{Kind: "ConfigMap", Namespace: "team", Name: "gone", Version: "v1", Component: "config"},
		deploy("other", "kept", "web"),
	}, want, "the oracle sees the fixture as the cli did")
	assert.Equal(t, want, inventory.StaleSet(previous, current))
}
