package object_test

// This file checks the duplicate-object-identities requirement that the
// deprecated opm/helper/objectset and opm/k8s/object return identical rows
// and error messages for the same render. Delete it in the change that
// removes opm/helper/objectset.

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/library/opm/helper/objectset" //nolint:depguard,staticcheck // parity with the deprecated copy until its removal (duplicate-object-identities spec)
	"github.com/open-platform-model/library/opm/k8s/object"
	"github.com/open-platform-model/library/opm/kernel"
)

// row is a home-neutral view of one duplicate row, so the two homes' types
// compare field by field.
type row struct {
	Identity  object.Identity
	Producers []object.Producer
}

func fromObject(rows []object.Duplicate) []row {
	var out []row
	for _, d := range rows {
		out = append(out, row{Identity: d.Identity, Producers: d.Producers})
	}
	return out
}

func fromObjectset(rows []objectset.Duplicate) []row { //nolint:staticcheck // parity with the deprecated copy until its removal
	var out []row
	for _, d := range rows {
		ps := make([]object.Producer, 0, len(d.Producers))
		for _, p := range d.Producers {
			ps = append(ps, object.Producer(p))
		}
		out = append(out, row{Identity: object.Identity(d.Identity), Producers: ps})
	}
	return out
}

func TestDuplicates_BothHomesAgree(t *testing.T) {
	renders := map[string]func() []*kernel.Compiled{
		"cluster-scoped pair": func() []*kernel.Compiled {
			return []*kernel.Compiled{registration(t, "registration"), registration(t, "registration-copy")}
		},
		"three objects, two identities": func() []*kernel.Compiled {
			return []*kernel.Compiled{
				deployment(t, "web", "…/deployment@1.2.0", "web"),
				configMap(t, "web", `apiVersion: "v1"`),
				deployment(t, "web", "…/legacy-deployment@1.0.0", "web"),
			}
		},
		"clean render": func() []*kernel.Compiled {
			return []*kernel.Compiled{
				deployment(t, "web", "…/deployment@1.2.0", "web"),
				deployment(t, "api", "…/deployment@1.2.0", "api"),
			}
		},
		"nameless values and a nil entry": func() []*kernel.Compiled {
			nameless := `
				apiVersion: "opmodel.dev/v1alpha2"
				kind:       "Platform"
				metadata: labels: tier: "core"
			`
			return []*kernel.Compiled{
				compiled(t, "platform", "…/platform@1.0.0", nameless),
				nil,
				compiled(t, "platform-copy", "…/platform@1.0.0", nameless),
			}
		},
		"two rows in first-seen order": func() []*kernel.Compiled {
			return []*kernel.Compiled{
				registration(t, "registration"),
				deployment(t, "web", "…/deployment@1.2.0", "web"),
				registration(t, "registration-copy"),
				deployment(t, "web", "…/legacy-deployment@1.0.0", "web"),
			}
		},
		"one object under two versions of its group": func() []*kernel.Compiled {
			return []*kernel.Compiled{
				deploymentAt(t, "a", "…/deployment@1.0.0", "apps/v1", "web"),
				deploymentAt(t, "b", "…/legacy@1.0.0", "apps/v1beta2", "web"),
			}
		},
		"missing apiVersion falls in the core group": func() []*kernel.Compiled {
			return []*kernel.Compiled{configMap(t, "a", `apiVersion: "v1"`), configMap(t, "b", "")}
		},
	}

	for name, render := range renders {
		t.Run(name, func(t *testing.T) {
			in := render()
			got := object.Duplicates(in)
			old := objectset.Duplicates(in) //nolint:staticcheck // parity with the deprecated copy until its removal

			assert.Equal(t, fromObjectset(old), fromObject(got))
			if len(got) > 0 {
				assert.Equal(t,
					(&objectset.DuplicateIdentitiesError{Duplicates: old}).Error(), //nolint:staticcheck // parity with the deprecated copy until its removal
					(&object.DuplicateIdentitiesError{Duplicates: got}).Error())
			}
		})
	}
}
