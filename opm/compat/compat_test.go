package compat

import (
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// compileX compiles src and returns the #X definition, the shape every case
// in the reference experiment (0011/experiments/03-d27-compat-gate) uses.
func compileX(t *testing.T, ctx *cue.Context, src string) cue.Value {
	t.Helper()
	v := ctx.CompileString(src).LookupPath(cue.ParsePath("#X"))
	require.NoError(t, v.Err(), "compiling %q", src)
	return v
}

type wantViolation struct {
	path, kind string
}

func toWant(vs []Violation) []wantViolation {
	var out []wantViolation
	for _, v := range vs {
		out = append(out, wantViolation{v.Path, v.Kind})
	}
	return out
}

// TestCheck ports the 14 experiment cases — every change class D27 names,
// nested variants, and the two label cases OQ16 turns on — plus edge cases
// beyond the experiment's dispatch. A nil want means the change is legal.
//
// The "change default" class reports twice at the same path: the explicit
// default comparison fires KindDefaultChanged, and the raw-mode leaf subsume
// — which must stay default-sensitive so a domain narrowed to its own default
// is caught — also fails there. Both reports are asserted; grouping them for
// presentation is the CLI's rendering work.
func TestCheck(t *testing.T) {
	tests := []struct {
		name, prev, next string
		want             []wantViolation
	}{
		// The 14 experiment cases.
		{"add optional field", `#X: {x: string}`, `#X: {x: string, y?: string}`, nil},
		{"add defaulted field", `#X: {x: string}`, `#X: {x: string, y: string | *"z"}`, nil},
		{"add required field", `#X: {x: string}`, `#X: {x: string, y: string}`,
			[]wantViolation{{"y", KindFieldAddedStrict}}},
		{"remove field", `#X: {x: string, y: string}`, `#X: {x: string}`,
			[]wantViolation{{"y", KindFieldRemoved}}},
		{"widen disjunction", `#X: {t: "a"|"b"}`, `#X: {t: "a"|"b"|"c"}`, nil},
		{"narrow disjunction", `#X: {t: "a"|"b"|"c"}`, `#X: {t: "a"|"b"}`,
			[]wantViolation{{"t", KindDomainNarrowed}}},
		{"change concrete value", `#X: {t: "a"}`, `#X: {t: "b"}`,
			[]wantViolation{{"t", KindDomainNarrowed}}},
		{"change default", `#X: {t: string | *"a"}`, `#X: {t: string | *"b"}`,
			[]wantViolation{{"t", KindDefaultChanged}, {"t", KindDomainNarrowed}}},
		{"identical", `#X: {x: string, t: "a"|"b"}`, `#X: {x: string, t: "a"|"b"}`, nil},
		{"nested field removed", `#X: {s: {a: string, b: string}}`, `#X: {s: {a: string}}`,
			[]wantViolation{{"s.b", KindFieldRemoved}}},
		{"nested option widened", `#X: {s: {t: "a"}}`, `#X: {s: {t: "a"|"b"}}`, nil},
		{"label disjunction narrowed",
			`#X: {metadata: labels: "wt": "stateless"|"stateful"|"daemon"}`,
			`#X: {metadata: labels: "wt": "stateless"|"stateful"}`,
			[]wantViolation{{"metadata.labels.wt", KindDomainNarrowed}}},
		{"label value changed",
			`#X: {metadata: labels: "wt": "stateful"}`,
			`#X: {metadata: labels: "wt": "daemon"}`,
			[]wantViolation{{"metadata.labels.wt", KindDomainNarrowed}}},
		{"label key added (optional)",
			`#X: {metadata: labels: {}}`, `#X: {metadata: labels: "tier"?: string}`, nil},

		// Edges beyond the experiment's dispatch.
		{"list leaf unchanged", `#X: {xs: [...string]}`, `#X: {xs: [...string]}`, nil},
		{"closed list element narrowed", `#X: {xs: [string]}`, `#X: {xs: [int]}`,
			[]wantViolation{{"xs[0]", KindDomainNarrowed}}},
		{"closed list element removed", `#X: {xs: [string, int]}`, `#X: {xs: [string]}`,
			[]wantViolation{{"xs", KindDomainNarrowed}}},
		{"closed list nested field removed", `#X: {xs: [{a: string, b: int}]}`, `#X: {xs: [{a: string}]}`,
			[]wantViolation{{"xs[0].b", KindFieldRemoved}}},
		{"matchN leaf unchanged", `#X: {s: matchN(1, [string, int])}`, `#X: {s: matchN(1, [string, int])}`, nil},
		// Known upstream blind spot, pinned deliberately (measured against
		// v0.17.1, identical before and after the identical-leaf rule): the
		// subsume cannot evaluate matchN, so narrowing its alternatives is not
		// detected and widening them false-positives. The identical-leaf rule
		// removes only the unchanged case. If a CUE upgrade starts judging
		// matchN, these two cases fail and the pin should be updated.
		{"matchN alternatives narrowed is not detected", `#X: {s: matchN(1, [string, int])}`, `#X: {s: matchN(1, [string])}`, nil},
		{"matchN alternatives widened false-positives", `#X: {s: matchN(1, [string])}`, `#X: {s: matchN(1, [string, int])}`,
			[]wantViolation{{"s", KindDomainNarrowed}}},
		{"root provenance skipped",
			`#X: {metadata: {name: "t", catalogVersion: "1.0.0", description: "a"}}`,
			`#X: {metadata: {name: "t", catalogVersion: "1.1.0", description: "b"}}`, nil},
		{"provenance skipped under nested metadata only",
			`#X: {a: {metadata: {catalogVersion: "1.0.0"}}, catalogVersion: "1.0.0"}`,
			`#X: {a: {metadata: {catalogVersion: "1.1.0"}}, catalogVersion: "1.1.0"}`,
			[]wantViolation{{"catalogVersion", KindDomainNarrowed}}},
		{"provenance removal under metadata not reported",
			`#X: {metadata: {name: "t", description: "a"}}`, `#X: {metadata: {name: "t"}}`, nil},
		// Known upstream blind spot, pinned deliberately: CUE's Subsume treats
		// open lists as mutually subsuming regardless of element type, under
		// every option combination (measured against v0.17.1). Element-domain
		// narrowing inside `[...T]` is therefore not detected at a leaf. If a
		// CUE upgrade starts reporting it, this case fails and the pin — plus
		// the package doc — should be updated to celebrate.
		{"open list element narrowed is not detected", `#X: {xs: [...string]}`, `#X: {xs: [...int]}`, nil},
		{"struct disjunction branch added", `#X: {s: {a: string}}`, `#X: {s: {a: string} | {b: string}}`, nil},
		{"struct disjunction branch removed", `#X: {s: {a: string} | {b: string}}`, `#X: {s: {a: string}}`,
			[]wantViolation{{"s", KindDomainNarrowed}}},
		{"hidden field removed is not reported", `#X: {x: string, _h: int}`, `#X: {x: string}`, nil},
		{"hidden field added is not reported", `#X: {x: string}`, `#X: {x: string, _h: int}`, nil},
		{"required-marker field removed", `#X: {x: string, y!: string}`, `#X: {x: string}`,
			[]wantViolation{{"y", KindFieldRemoved}}},
		{"optional field removed", `#X: {x: string, y?: string}`, `#X: {x: string}`,
			[]wantViolation{{"y", KindFieldRemoved}}},

		// Posture transitions (0010 D27: an optional field must not become
		// required). Judged on the selector; the value domain is judged
		// separately, so a posture change that also narrows reports both.
		{"optional made required (!)", `#X: {y?: string}`, `#X: {y!: string}`,
			[]wantViolation{{"y", KindFieldMadeRequired}}},
		{"optional made required (regular, no default)", `#X: {y?: string}`, `#X: {y: string}`,
			[]wantViolation{{"y", KindFieldMadeRequired}}},
		{"optional made required and narrowed", `#X: {y?: string}`, `#X: {y!: =~"^[a-z]"}`,
			[]wantViolation{{"y", KindFieldMadeRequired}, {"y", KindDomainNarrowed}}},
		// Not this rule's finding: the leaf subsume under cue.Raw is
		// default-sensitive and reports the added default as narrowing
		// (pre-existing verdict, same as "concrete-to-non-concrete default").
		{"optional gains a default", `#X: {y?: string}`, `#X: {y: string | *"z"}`,
			[]wantViolation{{"y", KindDomainNarrowed}}},
		{"required made optional", `#X: {y!: string}`, `#X: {y?: string}`, nil},
		{"regular gains a default", `#X: {y: string}`, `#X: {y: string | *"z"}`, nil},
		{"regular gains the required marker", `#X: {x: string}`, `#X: {x!: string}`, nil},
		{"required marker added as a new field", `#X: {x: string}`, `#X: {x: string, y!: string}`,
			[]wantViolation{{"y", KindFieldAddedStrict}}},
		// The incident this rule was written for: catalogs/opm alpha.5 ->
		// alpha.6 #ExposeSchema.name (measured 2026-08-28 on the published
		// builds; the transformer then read the field unconditionally).
		{"expose name alpha.5 to alpha.6",
			`#X: {ports: [string]: {targetPort: int}, type: "ClusterIP" | "NodePort", name?: string}`,
			`#X: {ports: [string]: {targetPort: int}, type: "ClusterIP" | "NodePort", name!: =~"^[a-z]([a-z0-9-]*[a-z0-9])?$"}`,
			[]wantViolation{{"name", KindFieldMadeRequired}, {"name", KindDomainNarrowed}}},

		// The default branches the design pins (non-concrete handling).
		{"default removed", `#X: {t: string | *"a"}`, `#X: {t: string}`,
			[]wantViolation{{"t", KindDefaultRemoved}}},
		{"non-concrete defaults equivalent", `#X: {t: *string | int}`, `#X: {t: *string | int}`, nil},
		{"non-concrete defaults asymmetric", `#X: {t: *string | int}`, `#X: {t: *"a" | int}`,
			[]wantViolation{{"t", KindDefaultChanged}, {"t", KindDomainNarrowed}}},
		{"concrete-to-non-concrete default", `#X: {t: *"a" | string}`, `#X: {t: *string | "a"}`,
			[]wantViolation{{"t", KindDefaultChanged}}},
	}

	ctx := cuecontext.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := compileX(t, ctx, tt.prev)
			next := compileX(t, ctx, tt.next)
			got := Check(prev, next)
			assert.Equal(t, tt.want, toWant(got))
		})
	}
}

// TestCheckDefaultRendering pins the Old/New rendering 0011's refusal
// message 9 consumes: `t: default changed ("a" -> "b")`.
func TestCheckDefaultRendering(t *testing.T) {
	ctx := cuecontext.New()
	prev := compileX(t, ctx, `#X: {t: string | *"a"}`)
	next := compileX(t, ctx, `#X: {t: string | *"b"}`)

	got := Check(prev, next)
	require.NotEmpty(t, got)
	require.Equal(t, KindDefaultChanged, got[0].Kind)
	assert.Equal(t, `"a"`, got[0].Old)
	assert.Equal(t, `"b"`, got[0].New)

	removed := Check(prev, compileX(t, ctx, `#X: {t: string}`))
	require.NotEmpty(t, removed)
	require.Equal(t, KindDefaultRemoved, removed[0].Kind)
	assert.Equal(t, `"a"`, removed[0].Old)
	assert.Equal(t, "", removed[0].New)
}

// TestCheckDomainNarrowedCarriesSubsumeError pins that KindDomainNarrowed's
// New carries the CUE subsumption diagnostic verbatim.
func TestCheckDomainNarrowedCarriesSubsumeError(t *testing.T) {
	ctx := cuecontext.New()
	prev := compileX(t, ctx, `#X: {t: "a"|"b"|"c"}`)
	next := compileX(t, ctx, `#X: {t: "a"|"b"}`)

	got := Check(prev, next)
	require.Len(t, got, 1)
	assert.NotEmpty(t, got[0].New, "subsume error text must be carried verbatim")
	assert.Empty(t, got[0].Old)
}

func TestCheckAtLevel(t *testing.T) {
	ctx := cuecontext.New()
	// Incompatible operands: a field removal, refused at every enforced level.
	prev := compileX(t, ctx, `#X: {x: string, y: string}`)
	next := compileX(t, ctx, `#X: {x: string}`)

	t.Run("alpha is not gated", func(t *testing.T) {
		for _, av := range []string{"v1alpha1", "v1alpha2"} {
			vs, err := CheckAtLevel(av, prev, next)
			require.NoError(t, err)
			assert.Nil(t, vs, "alpha promises nothing (D34)")
		}
	})

	t.Run("beta and GA are gated", func(t *testing.T) {
		want := Check(prev, next)
		require.NotEmpty(t, want)
		for _, av := range []string{"v1beta1", "v1", "v2"} {
			vs, err := CheckAtLevel(av, prev, next)
			require.NoError(t, err)
			assert.Equal(t, want, vs)
		}
	})

	t.Run("unparseable apiVersion is an error, not a violation", func(t *testing.T) {
		for _, av := range []string{"v1alpha", "1.2.0", "", "V1"} {
			vs, err := CheckAtLevel(av, prev, next)
			require.ErrorIs(t, err, ErrUnparseableAPIVersion, "apiVersion %q", av)
			assert.Nil(t, vs)
		}
	})

	t.Run("non-struct top-level operands are an error", func(t *testing.T) {
		leaf := compileX(t, ctx, `#X: string`)
		vs, err := CheckAtLevel("v1", leaf, next)
		require.ErrorIs(t, err, ErrNotStruct)
		assert.Nil(t, vs)

		vs, err = CheckAtLevel("v1", prev, leaf)
		require.ErrorIs(t, err, ErrNotStruct)
		assert.Nil(t, vs)
	})
}

// The measured Subsume/Fields options are load-bearing (design: "MUST be
// preserved with a test asserting each"). Each test below fails if its option
// is dropped from the walk.

// TestOptionAllIsLoadBearing: without cue.All() on field iteration, optional
// fields are invisible and this removal goes unreported.
func TestOptionAllIsLoadBearing(t *testing.T) {
	ctx := cuecontext.New()
	prev := compileX(t, ctx, `#X: {x: string, y?: string}`)
	next := compileX(t, ctx, `#X: {x: string}`)
	assert.Equal(t, []wantViolation{{"y", KindFieldRemoved}}, toWant(Check(prev, next)))
}

// TestOptionSchemaIsLoadBearing: an optional field dropped inside a
// disjunction branch. The branch pair is walked as a leaf, and under
// cue.Schema() the subsumption accepts it — this is the documented consequence
// of leaf treatment for struct disjunctions. Without cue.Schema() the case
// reports a spurious domain narrowing and this test fails.
func TestOptionSchemaIsLoadBearing(t *testing.T) {
	ctx := cuecontext.New()
	prev := compileX(t, ctx, `#X: {s: {a?: int} | string}`)
	next := compileX(t, ctx, `#X: {s: {} | string}`)
	assert.Empty(t, Check(prev, next))
}

// TestOptionRawIsLoadBearing: a domain narrowed to its own prior default.
// Without cue.Raw() the subsumption resolves prev toward its default and the
// narrowing from `string` to `"a"` goes unreported.
func TestOptionRawIsLoadBearing(t *testing.T) {
	ctx := cuecontext.New()
	prev := compileX(t, ctx, `#X: {t: string | *"a"}`)
	next := compileX(t, ctx, `#X: {t: "a"}`)
	got := toWant(Check(prev, next))
	assert.Contains(t, got, wantViolation{"t", KindDomainNarrowed})
}

// The member-reference shapes measured on catalog_opm PR 51 (cli issue 165):
// a trait's appliesTo and a blueprint's composedResources embed whole
// members, whose metadata.catalogVersion differs between builds by
// construction. Before the walk applied D30 at depth, the list leaf's
// subsume named the nested catalogVersion and refused an unchanged member.
func memberRefSrc(version, nameField string) string {
	return `
#Container: {
	kind: "Resource"
	metadata: {
		name: "container", apiVersion: "v1beta1"
		catalogVersion: *"` + version + `" | string
		description: "A container"
		fqn: "x/resources/container@v1beta1"
	}
	spec: {image!: string, ports?: [...int]}
}
#Scaling: {
	kind: "Trait"
	metadata: {name: "scaling", apiVersion: "v1beta1", catalogVersion: "` + version + `", description: "d"}
	spec: scaling: {replicas: int | *1}
}
#X: {
	kind: "Blueprint"
	metadata: {name: "expose", apiVersion: "v1beta1", catalogVersion: "` + version + `", description: "d"}
	appliesTo:         [#Container]
	composedResources: [#Container, #Container]
	composedTraits:    [#Scaling]
	spec: expose: {` + nameField + `}
}
`
}

func TestCheckMemberReferencesIgnoreProvenance(t *testing.T) {
	ctx := cuecontext.New()
	prev := compileX(t, ctx, memberRefSrc("1.0.0", "name?: string"))
	next := compileX(t, ctx, memberRefSrc("1.1.0", "name?: string"))
	assert.Empty(t, Check(prev, next))
}

func TestCheckMemberReferencesStillReportRealChanges(t *testing.T) {
	ctx := cuecontext.New()
	prev := compileX(t, ctx, memberRefSrc("1.0.0", "name?: string"))
	next := compileX(t, ctx, memberRefSrc("1.1.0", `name?: =~"^[a-z]+$"`))
	// Exactly the genuine tightening, nothing from the references.
	assert.Equal(t, []wantViolation{{"spec.expose.name", KindDomainNarrowed}}, toWant(Check(prev, next)))

	// A change inside a referenced member reports at the element path.
	broken := strings.Replace(memberRefSrc("1.1.0", "name?: string"), "ports?: [...int]", "", 1)
	got := toWant(Check(prev, compileX(t, ctx, broken)))
	assert.Contains(t, got, wantViolation{"appliesTo[0].spec.ports", KindFieldRemoved})
	assert.Contains(t, got, wantViolation{"composedResources[1].spec.ports", KindFieldRemoved})
}

// The #Image shape: a struct of if-guards over not-yet-concrete siblings,
// unchanged on both sides, must be silent at every path.
func TestCheckGuardedStructUnchanged(t *testing.T) {
	const src = `
#Image: {
	repository: string
	tag:        string | *""
	digest:     string | *""
	if digest != "" && tag != "" {reference: "\(repository):\(tag)@\(digest)"}
	if digest != "" && tag == "" {reference: "\(repository)@\(digest)"}
	if digest == "" && tag != "" {reference: "\(repository):\(tag)"}
	if digest == "" && tag == "" {reference: repository}
}
#X: {spec: container: {image!: #Image, updateStrategy: *"RollingUpdate" | "OnDelete"}}
`
	ctx := cuecontext.New()
	assert.Empty(t, Check(compileX(t, ctx, src), compileX(t, ctx, src)))
}
