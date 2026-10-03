package objectset

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/kernel"
)

// Identity names a rendered object by its apiVersion, kind, namespace and
// name. Namespace is empty for a cluster-scoped object, or for one that names
// no namespace. A Kubernetes apply addresses an object by group, kind,
// namespace and name, so two identities that differ only in the version part
// of APIVersion address one object; [Duplicates] matches them on the group.
type Identity struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
}

// String renders the identity as "apps/v1 Deployment web-system/web", or
// "apps/v1 Deployment web" when there is no namespace.
func (i Identity) String() string {
	name := i.Name
	if i.Namespace != "" {
		name = i.Namespace + "/" + i.Name
	}
	return strings.TrimSpace(i.APIVersion+" "+i.Kind) + " " + name
}

// Producer is the (component, transformer) pair the kernel recorded on a
// rendered object, with that object's own apiVersion. Within one row the
// producers' APIVersion values differ when one object was rendered under two
// versions of its group.
type Producer struct {
	Component   string
	Transformer string
	APIVersion  string
}

// String renders the producer as: component "web" (…/deployment@1.2.0).
func (p Producer) String() string {
	return fmt.Sprintf("component %q (%s)", p.Component, p.Transformer)
}

// Duplicate is one apply identity that two or more rendered objects share,
// with every producer of it in render order. Identity is the first-rendered
// object's, its APIVersion verbatim.
type Duplicate struct {
	Identity  Identity
	Producers []Producer
}

// Duplicates scans a render's compiled objects and returns every apply
// identity two or more of them share, in the order each identity was first
// rendered. Two objects share an apply identity when their API group (the
// part of apiVersion before the first "/", empty for the core group and for
// an object with no apiVersion), kind, namespace and name match; the version
// does not distinguish them, because the API server serves one object under
// every version of its group. A value carrying no kind or no metadata.name is not a Kubernetes
// object and is skipped; nothing else about the objects is validated. A
// render whose objects all have distinct identities returns no rows.
func Duplicates(compiled []*kernel.Compiled) []Duplicate {
	order := make([]key, 0, len(compiled))
	first := make(map[key]Identity, len(compiled))
	producers := make(map[key][]Producer, len(compiled))

	for _, c := range compiled {
		if c == nil {
			continue
		}
		id, ok := identityOf(c.Value)
		if !ok {
			continue
		}
		k := keyOf(id)
		if _, seen := first[k]; !seen {
			order = append(order, k)
			first[k] = id
		}
		producers[k] = append(producers[k], Producer{
			Component:   c.Component,
			Transformer: c.Transformer,
			APIVersion:  id.APIVersion,
		})
	}

	var rows []Duplicate
	for _, k := range order {
		if len(producers[k]) < 2 {
			continue
		}
		rows = append(rows, Duplicate{Identity: first[k], Producers: producers[k]})
	}
	return rows
}

// key is what two objects must share to be one object to the API server.
type key struct{ group, kind, namespace, name string }

func keyOf(id Identity) key {
	return key{
		group:     groupOf(id.APIVersion),
		kind:      id.Kind,
		namespace: id.Namespace,
		name:      id.Name,
	}
}

// groupOf returns the API group of an apiVersion: the part before the first
// "/", or "" for a bare version such as "v1", which names the core group.
func groupOf(apiVersion string) string {
	group, _, found := strings.Cut(apiVersion, "/")
	if !found {
		return ""
	}
	return group
}

// identityOf reads the four identity fields off a rendered value. A field
// that is absent or not a string is left empty; a value with no kind or no
// metadata.name is not an object, and ok is false.
func identityOf(v cue.Value) (Identity, bool) {
	id := Identity{
		APIVersion: stringAt(v, "apiVersion"),
		Kind:       stringAt(v, "kind"),
		Namespace:  stringAt(v, "metadata", "namespace"),
		Name:       stringAt(v, "metadata", "name"),
	}
	if id.Kind == "" || id.Name == "" {
		return Identity{}, false
	}
	return id, true
}

func stringAt(v cue.Value, selectors ...string) string {
	path := make([]cue.Selector, len(selectors))
	for i, s := range selectors {
		path[i] = cue.Str(s)
	}
	s, err := v.LookupPath(cue.MakePath(path...)).String()
	if err != nil {
		return ""
	}
	return s
}

// DuplicateIdentitiesError is the refusal a runtime raises from the rows
// [Duplicates] returned, before apply. The kernel never returns it: it is
// raised by the frontend that calls the helper.
type DuplicateIdentitiesError struct {
	Duplicates []Duplicate
}

// Error names each shared identity once, on its own line, with every
// component and transformer that produced it, so one wording serves every
// runtime that applies to Kubernetes. When a row's producers rendered the
// object under different apiVersions, each producer is followed by its own
// version, so the reader sees the mismatch that made them one object.
func (e *DuplicateIdentitiesError) Error() string {
	objects := 0
	for _, d := range e.Duplicates {
		objects += len(d.Producers)
	}

	var b strings.Builder
	if len(e.Duplicates) == 1 {
		fmt.Fprintf(&b, "%d rendered objects share one identity, "+
			"so the last apply would silently overwrite the first:", objects)
	} else {
		fmt.Fprintf(&b, "%d rendered objects share %d identities, "+
			"so the last apply would silently overwrite the first:",
			objects, len(e.Duplicates))
	}
	for _, d := range e.Duplicates {
		fmt.Fprintf(&b, "\n  %s rendered by %s", d.Identity, joinProducers(d.Producers))
	}
	return b.String()
}

func joinProducers(producers []Producer) string {
	mixed := mixedVersions(producers)
	parts := make([]string, len(producers))
	for i, p := range producers {
		parts[i] = p.String()
		if mixed {
			parts[i] += " as " + versionLabel(p.APIVersion)
		}
	}
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
	}
}

// mixedVersions reports whether the producers do not all carry one
// apiVersion.
func mixedVersions(producers []Producer) bool {
	for i := 1; i < len(producers); i++ {
		if producers[i].APIVersion != producers[0].APIVersion {
			return true
		}
	}
	return false
}

func versionLabel(apiVersion string) string {
	if apiVersion == "" {
		return "<no apiVersion>"
	}
	return apiVersion
}
