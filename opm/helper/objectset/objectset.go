package objectset

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/kernel"
)

// Identity is a rendered object's apply identity: the four fields a
// Kubernetes apply addresses an object by. Namespace is empty for a
// cluster-scoped object, or for one that names no namespace.
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
// rendered object.
type Producer struct {
	Component   string
	Transformer string
}

// String renders the producer as: component "web" (…/deployment@1.2.0).
func (p Producer) String() string {
	return fmt.Sprintf("component %q (%s)", p.Component, p.Transformer)
}

// Duplicate is one apply identity that two or more rendered objects share,
// with every producer of it in render order.
type Duplicate struct {
	Identity  Identity
	Producers []Producer
}

// Duplicates scans a render's compiled objects and returns every apply
// identity two or more of them share, in the order each identity was first
// rendered. A value carrying no kind or no metadata.name is not a Kubernetes
// object and is skipped; nothing else about the objects is validated. A
// render whose objects all have distinct identities returns no rows.
func Duplicates(compiled []*kernel.Compiled) []Duplicate {
	order := make([]Identity, 0, len(compiled))
	producers := make(map[Identity][]Producer, len(compiled))

	for _, c := range compiled {
		if c == nil {
			continue
		}
		id, ok := identityOf(c.Value)
		if !ok {
			continue
		}
		if _, seen := producers[id]; !seen {
			order = append(order, id)
		}
		producers[id] = append(producers[id], Producer{
			Component:   c.Component,
			Transformer: c.Transformer,
		})
	}

	var rows []Duplicate
	for _, id := range order {
		if len(producers[id]) < 2 {
			continue
		}
		rows = append(rows, Duplicate{Identity: id, Producers: producers[id]})
	}
	return rows
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
// runtime that applies to Kubernetes.
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
	parts := make([]string, len(producers))
	for i, p := range producers {
		parts[i] = p.String()
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
