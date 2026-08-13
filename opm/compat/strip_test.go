package compat

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// primitiveSrc builds a definition-unified value the way a catalog member
// reaches the comparator: a closed definition whose metadata declares
// catalogVersion as required, unified with a concrete instance.
func primitiveSrc(catalogVersion string) string {
	return `
#Resource: {
	metadata: {
		name!:           string
		catalogVersion!: string
		description?:    string
		labels?: [string]: string
	}
	spec: {
		size:        string
		description: string | *"none"
		retention:   string | *"30d"
	}
}
out: #Resource & {
	metadata: {
		name:           "volume"
		catalogVersion: "` + catalogVersion + `"
		description:    "a volume"
		labels: tier: "storage"
	}
	spec: size: "10Gi"
}
`
}

func compileOut(t *testing.T, ctx *cue.Context, src string) cue.Value {
	t.Helper()
	v := ctx.CompileString(src).LookupPath(cue.ParsePath("out"))
	require.NoError(t, v.Err())
	return v
}

func TestStripProvenance(t *testing.T) {
	ctx := cuecontext.New()
	stripped, err := StripProvenance(compileOut(t, ctx, primitiveSrc("1.0.0")))
	require.NoError(t, err)

	t.Run("instance fields removed", func(t *testing.T) {
		assert.False(t, stripped.LookupPath(cue.ParsePath("metadata.catalogVersion")).Exists())
		assert.False(t, stripped.LookupPath(cue.ParsePath("metadata.description")).Exists())
	})

	t.Run("definition-side required field removed", func(t *testing.T) {
		// The Validate(cue.Concrete(false)) blind spot: had only the
		// instance's field been deleted, the definition's catalogVersion!
		// would remain as a required field nothing satisfies. The strip must
		// reach the definition, leaving a value that validates.
		require.NoError(t, stripped.Validate(cue.Concrete(false)))
		assert.False(t, stripped.LookupPath(cue.MakePath(cue.Str("metadata"), cue.Str("catalogVersion").Optional())).Exists(),
			"no catalogVersion constraint may survive on the definition side")
	})

	t.Run("all other fields untouched", func(t *testing.T) {
		name, err := stripped.LookupPath(cue.ParsePath("metadata.name")).String()
		require.NoError(t, err)
		assert.Equal(t, "volume", name)
		tier, err := stripped.LookupPath(cue.ParsePath("metadata.labels.tier")).String()
		require.NoError(t, err)
		assert.Equal(t, "storage", tier)
		size, err := stripped.LookupPath(cue.ParsePath("spec.size")).String()
		require.NoError(t, err)
		assert.Equal(t, "10Gi", size)
	})

	t.Run("denylist names outside metadata untouched", func(t *testing.T) {
		// The denylist is metadata.catalogVersion + metadata.description —
		// a spec field that happens to be called description stays.
		assert.True(t, stripped.LookupPath(cue.ParsePath("spec.description")).Exists())
	})
}

// TestStripProvenanceClosedness: the round-trip must not open the value. Both
// spec-body styles are exercised — the body authored inline in the definition,
// and the body reaching the definition through a referenced sub-definition.
func TestStripProvenanceClosedness(t *testing.T) {
	ctx := cuecontext.New()

	styles := map[string]string{
		"inline body": primitiveSrc("1.0.0"),
		"referenced body": `
#Spec: {
	size: string
}
#Resource: {
	metadata: {
		name!:           string
		catalogVersion!: string
		description?:    string
	}
	spec: #Spec
}
out: #Resource & {
	metadata: {name: "volume", catalogVersion: "1.0.0"}
	spec: size: "10Gi"
}
`,
	}

	for name, src := range styles {
		t.Run(name, func(t *testing.T) {
			stripped, err := StripProvenance(compileOut(t, ctx, src))
			require.NoError(t, err)

			undeclared := ctx.CompileString(`{spec: {undeclared: "x"}}`)
			require.NoError(t, undeclared.Err())
			unified := stripped.Unify(undeclared)
			assert.Error(t, unified.Validate(cue.Concrete(false)),
				"an undeclared field must still be refused after the strip")
		})
	}
}

// TestStripProvenanceInlinesImports: a value whose definition comes from an
// imported package must rebuild self-contained. cue.InlineImports(true) is
// what makes the emitted syntax carry the imported definition's body instead
// of an unresolvable package reference — without it, this rebuild fails and
// so does the test.
func TestStripProvenanceInlinesImports(t *testing.T) {
	const dir = "/striptest"
	overlay := map[string]load.Source{
		dir + "/cue.mod/module.cue": load.FromString(`module: "example.com/striptest"
language: version: "v0.17.0"
`),
		dir + "/meta/meta.cue": load.FromString(`package meta

#Meta: {
	name!:           string
	catalogVersion!: string
	description?:    string
}
`),
		dir + "/main.cue": load.FromString(`package main

import "example.com/striptest/meta"

out: {
	metadata: meta.#Meta
	spec: size: string
} & {
	metadata: {name: "volume", catalogVersion: "1.0.0"}
	spec: size: "10Gi"
}
`),
	}

	insts := load.Instances([]string{"."}, &load.Config{Dir: dir, ModuleRoot: dir, Overlay: overlay})
	require.Len(t, insts, 1)
	require.NoError(t, insts[0].Err)

	ctx := cuecontext.New()
	v := ctx.BuildInstance(insts[0]).LookupPath(cue.ParsePath("out"))
	require.NoError(t, v.Err())

	stripped, err := StripProvenance(v)
	require.NoError(t, err)
	require.NoError(t, stripped.Validate(cue.Concrete(false)))
	assert.False(t, stripped.LookupPath(cue.ParsePath("metadata.catalogVersion")).Exists())
	name, err := stripped.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "volume", name)
}

// TestStripProvenanceEnablesComparison is the integration-shaped case: two
// builds of an identical primitive differing only in provenance compare with
// violations unstripped — catalogVersion changes on every release by
// construction, so an unstripped gate would fire on every publish — and
// cleanly after StripProvenance on both operands.
func TestStripProvenanceEnablesComparison(t *testing.T) {
	ctx := cuecontext.New()
	prev := compileOut(t, ctx, primitiveSrc("1.0.0"))
	next := compileOut(t, ctx, primitiveSrc("1.1.0"))

	require.NotEmpty(t, Check(prev, next), "unstripped operands must report — provenance differs")

	sprev, err := StripProvenance(prev)
	require.NoError(t, err)
	snext, err := StripProvenance(next)
	require.NoError(t, err)
	assert.Empty(t, Check(sprev, snext))
}
