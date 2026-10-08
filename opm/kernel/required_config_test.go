package kernel_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	cueerrors "cuelang.org/go/cue/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
)

// unreadRequiredBody is a module whose #config declares required values no
// component reads (image, tag, any) next to an optional and a defaulted one;
// its only component reads replicas.
const unreadRequiredBody = "#config: {\n" +
	"\treplicas: int | *1\n" +
	"\timage:    string\n" +
	"\ttag!:     string\n" +
	"\topt?:     string\n" +
	"\tany:      _\n" +
	"}\n" +
	"debugValues: {}\n" +
	"#components: foo: {metadata: name: \"foo\", _r: #config.replicas & int}\n"

// positionsEndWith reports whether any position of the finding at path in
// err's CUE tree names a file whose path ends with suffix.
func positionsEndWith(err error, path, suffix string) bool {
	var cerr cueerrors.Error
	if !errors.As(err, &cerr) {
		return false
	}
	for _, e := range cueerrors.Errors(cerr) {
		if strings.Join(e.Path(), ".") != path {
			continue
		}
		for _, pos := range cueerrors.Positions(e) {
			if strings.HasSuffix(pos.Filename(), suffix) {
				return true
			}
		}
	}
	return false
}

// assertUnreadRequiredRefused checks the refusal of an instance of
// unreadRequiredBody whose values set only replicas (and maybe opt): no
// instance, the verb's concreteness framing, a finding for each unset
// required field at values.<field>, the declared ones positioned in the
// module's own file, and nothing for the optional or defaulted field.
func assertUnreadRequiredRefused(t *testing.T, verb string, inst *module.Instance, err error) {
	t.Helper()
	require.Error(t, err)
	assert.Nil(t, inst)
	assert.True(t, strings.HasPrefix(err.Error(), verb+`: instance "myrel": not fully concrete: `), "framing: %v", err)
	for _, field := range []string{"image", "tag", "any"} {
		assert.True(t, hasErrorPath(err, "values."+field), "no error names values.%s: %v", field, err)
	}
	for _, field := range []string{"image", "tag"} {
		assert.True(t, positionsEndWith(err, "values."+field, "/module.cue"),
			"values.%s is not positioned at its #config declaration: %v", field, err)
	}
	for _, field := range []string{"opt", "replicas"} {
		assert.False(t, hasErrorPath(err, "values."+field), "an error names values.%s: %v", field, err)
	}
	assert.Contains(t, cueerrors.Details(err, nil), "field is required but not present")
}

// kernel-runtime spec, "Synthesis refuses an unread required value": an
// instance that leaves a required #config value unset is refused even when
// no component reads that value, as ValidateConfigDetailed refuses the same
// values (library#211).
func TestKernel_SynthesizeInstance_UnreadRequiredConfigUnset(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", unreadRequiredBody)
	src := mustSource(t, k, "/values/a.cue", `replicas: 2`)

	_, vErr := k.ValidateConfigDetailed(mod.ConfigSchema(), []kernel.Source{src})
	require.Error(t, vErr)
	for _, field := range []string{"image", "tag", "any"} {
		assert.True(t, hasErrorPath(vErr, "#config."+field), "ValidateConfigDetailed does not name %s: %v", field, vErr)
	}

	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module: mod, Name: "myrel", Namespace: "default", Values: []kernel.Source{src},
	})
	assertUnreadRequiredRefused(t, "Kernel.SynthesizeInstance", inst, err)
}

// kernel-runtime spec, "Directory acquisition refuses an unread required
// value": the package's own values leave image, tag and any unset, with and
// without a trailing source, and the instance is refused.
func TestKernel_AcquireInstanceFromDir_UnreadRequiredConfigUnset(t *testing.T) {
	k, _, modPath := publishSynthModuleAt(t, "demo", "0.1.0", unreadRequiredBody)
	dir := writeImportedInstance(t, t.TempDir(), "authored.opmodel.dev/instance@v0", modPath, "0.1.0",
		"myrel", "default", "{replicas: 2}", nil)

	for name, sources := range map[string][]kernel.Source{
		"no sources":          nil,
		"an unrelated source": {mustSource(t, k, "/values/opt.cue", `opt: "x"`)},
	} {
		t.Run(name, func(t *testing.T) {
			inst, err := k.AcquireInstanceFromDir(context.Background(), dir, sources...)
			assertUnreadRequiredRefused(t, "Kernel.AcquireInstanceFromDir", inst, err)
		})
	}
}

// kernel-runtime spec, "Optional and defaulted fields are not required":
// values that leave unset only an optional field and a defaulted one are
// accepted by both verbs.
func TestKernel_InstanceVerbs_OptionalAndDefaultedConfigUnsetAccepted(t *testing.T) {
	k, mod, modPath := publishSynthModuleAt(t, "demo", "0.1.0", unreadRequiredBody)
	const set = `{image: "nginx", tag: "v1", any: 3}`
	dir := writeImportedInstance(t, t.TempDir(), "authored.opmodel.dev/instance@v0", modPath, "0.1.0",
		"myrel", "default", set, nil)
	ctx := context.Background()

	inst, err := k.SynthesizeInstance(ctx, kernel.InstanceInput{
		Module: mod, Name: "myrel", Namespace: "default",
		Values: []kernel.Source{mustSource(t, k, "/values/a.cue", set)},
	})
	require.NoError(t, err)
	require.NotNil(t, inst)

	inst, err = k.AcquireInstanceFromDir(ctx, dir)
	require.NoError(t, err)
	require.NotNil(t, inst)
}

// parityBody declares, next to the unread required fields of
// unreadRequiredBody, a field a component reads with no default (host) and a
// read field with a constraint (port). The component reads both only through
// hidden fields, which the built-spec check does not validate, so the rows for
// host and port exercise the required-config check, not a build refusal.
const parityBody = "#config: {\n" +
	"\treplicas: int | *1\n" +
	"\timage:    string\n" +
	"\ttag!:     string\n" +
	"\topt?:     string\n" +
	"\tany:      _\n" +
	"\thost:     string\n" +
	"\tport:     int & >=1 & <=65535 | *80\n" +
	"}\n" +
	"debugValues: {}\n" +
	"#components: foo: {metadata: name: \"foo\", _r: #config.replicas & int, _h: #config.host & string, _p: #config.port & int}\n"

// kernel-runtime spec, "The refused set matches ValidateConfigDetailed": the
// operator checks one values source with ValidateConfigDetailed before
// SynthesizeInstance; synthesis alone refuses every value that check
// refuses, so the operator's pre-check adds nothing (library#211). The one
// difference runs the other way and predates the required-config check: a
// source that gives a defaulted field a bare type keeps the #config default
// for ValidateConfigDetailed, while the built-spec check refuses the bare
// type the instance's values carry.
func TestKernel_SynthesizeInstance_RequiredConfigMatchesValidateConfigDetailed(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", parityBody)
	const all = `image: "nginx", tag: "v1", any: 3, host: "h"`

	cases := []struct {
		name          string
		values        string
		validateFails bool
		synthFails    bool
	}{
		{"every field set", all + `, opt: "o", replicas: 3, port: 8080`, false, false},
		{"an unread string unset", `tag: "v1", any: 3, host: "h"`, true, true},
		{"an unread required marker unset", `image: "nginx", any: 3, host: "h"`, true, true},
		{"an unread _ unset", `image: "nginx", tag: "v1", host: "h"`, true, true},
		{"an optional field unset", all, false, false},
		{"a defaulted field unset", all + `, opt: "o"`, false, false},
		{"a field read only through a hidden field unset", `image: "nginx", tag: "v1", any: 3`, true, true},
		{"a key #config does not declare", all + `, extra: 1`, true, true},
		{"an unread field of the wrong type", `image: 5, tag: "v1", any: 3, host: "h"`, true, true},
		{"a field read only through a hidden field violating its constraint", all + `, port: 0`, true, true},
		{"the empty document", `{}`, true, true},
		{"a defaulted field given a bare type", all + `, port: int`, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := mustSource(t, k, "/values/a.cue", tc.values)

			_, vErr := k.ValidateConfigDetailed(mod.ConfigSchema(), []kernel.Source{src})
			inst, sErr := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
				Module: mod, Name: "myrel", Namespace: "default", Values: []kernel.Source{src},
			})

			assert.Equal(t, tc.validateFails, vErr != nil, "ValidateConfigDetailed: %v", vErr)
			assert.Equal(t, tc.synthFails, sErr != nil, "SynthesizeInstance: %v", sErr)
			if vErr != nil {
				assert.Error(t, sErr, "SynthesizeInstance accepts values ValidateConfigDetailed refuses: %v", vErr)
			}
			assert.Equal(t, sErr == nil, inst != nil)
		})
	}
}

// emptyDocBody is a module whose #config is fully defaulted, so the empty
// document the operator sends for a ModuleInstance without spec.values is
// accepted by both checks.
const emptyDocBody = "#config: {replicas: int | *1, opt?: string}\ndebugValues: {}\n" +
	"#components: foo: {metadata: name: \"foo\", _r: #config.replicas & int}\n"

// The operator's no-values source, the empty document, is accepted by both
// checks when every #config field has a default or is optional.
func TestKernel_SynthesizeInstance_EmptyDocumentOverDefaultsAccepted(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", emptyDocBody)
	src := mustSource(t, k, "/values/empty.cue", `{}`)

	_, vErr := k.ValidateConfigDetailed(mod.ConfigSchema(), []kernel.Source{src})
	require.NoError(t, vErr)
	inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module: mod, Name: "myrel", Namespace: "default", Values: []kernel.Source{src},
	})
	require.NoError(t, err)
	require.NotNil(t, inst)
}

// kernel-runtime spec, "Instances refused before keep their error": a bare
// constraint in the package's own values fails the built-spec check first,
// at the values file that holds it, and the required-config check never
// adds a finding at the module's #config declaration.
func TestKernel_AcquireInstanceFromDir_NonConcreteOwnValueKeepsSpecError(t *testing.T) {
	k := newRenderKernel(t)
	dir := copyRenderInstance(t, "instance_partial")
	valuesFile := writeValuesFile(t, dir, "replicas: >=1")

	inst, err := k.AcquireInstanceFromDir(context.Background(), dir)
	require.Error(t, err)
	assert.Nil(t, inst)
	assert.True(t, strings.HasPrefix(err.Error(), `Kernel.AcquireInstanceFromDir: instance "web-partial": not fully concrete: `), "framing: %v", err)
	assert.True(t, hasErrorPath(err, "values.replicas"), "no error names values.replicas: %v", err)
	assert.True(t, positionsName(err, valuesFile), "no position names the package's values.cue: %v", err)

	var cerr cueerrors.Error
	require.ErrorAs(t, err, &cerr)
	for _, e := range cueerrors.Errors(cerr) {
		for _, pos := range cueerrors.Positions(e) {
			assert.NotEqual(t, "module.cue", filepath.Base(pos.Filename()),
				"a finding is positioned at the module's #config declaration: %v", e)
		}
	}
}

// kernel-runtime spec, "Directory acquisition refuses an unread required
// value": the effective values include the trailing sources, so a source that
// supplies the required values the package's own values leave unset makes the
// instance acceptable.
func TestKernel_AcquireInstanceFromDir_TrailingSourceSuppliesRequiredConfig(t *testing.T) {
	k, _, modPath := publishSynthModuleAt(t, "demo", "0.1.0", unreadRequiredBody)
	dir := writeImportedInstance(t, t.TempDir(), "authored.opmodel.dev/instance@v0", modPath, "0.1.0",
		"myrel", "default", "{replicas: 2}", nil)
	src := mustSource(t, k, "/values/a.cue", `image: "n", tag: "v", any: 1`)

	inst, err := k.AcquireInstanceFromDir(context.Background(), dir, src)
	require.NoError(t, err)
	require.NotNil(t, inst)
}

// assertWrittenBareTypeReportedOnce checks the refusal of an instance of
// unreadRequiredBody where a source writes `image: string` and leaves tag and
// any unset: image is reported once, by the built-spec check, positioned
// where the values carry it and not at its #config declaration; tag and any
// are named next to it; nothing else is reported.
func assertWrittenBareTypeReportedOnce(t *testing.T, verb string, inst *module.Instance, err error) {
	t.Helper()
	require.Error(t, err)
	assert.Nil(t, inst)
	assert.True(t, strings.HasPrefix(err.Error(), verb+`: instance "myrel": not fully concrete: values.`), "framing: %v", err)
	var cerr cueerrors.Error
	require.ErrorAs(t, err, &cerr)
	var paths []string
	for _, e := range cueerrors.Errors(cerr) {
		paths = append(paths, strings.Join(e.Path(), "."))
	}
	assert.ElementsMatch(t, []string{"values.image", "values.tag", "values.any"}, paths, "findings: %v", err)
	assert.False(t, positionsEndWith(err, "values.image", "/module.cue"),
		"values.image is positioned at its #config declaration, so it is the required-config finding: %v", err)
}

// kernel-runtime spec, "A value the built spec refuses is reported once": a
// source writes a bare type for image while tag and any stay unset. Both
// checks refuse image; it is reported once, with the built-spec check's
// finding, and the two unset values are named with it, on both verbs.
func TestKernel_InstanceVerbs_WrittenBareTypeReportedOnce(t *testing.T) {
	k, mod, modPath := publishSynthModuleAt(t, "demo", "0.1.0", unreadRequiredBody)
	ctx := context.Background()

	inst, err := k.SynthesizeInstance(ctx, kernel.InstanceInput{
		Module: mod, Name: "myrel", Namespace: "default",
		Values: []kernel.Source{mustSource(t, k, "/values/a.cue", `replicas: 2, image: string`)},
	})
	assertWrittenBareTypeReportedOnce(t, "Kernel.SynthesizeInstance", inst, err)

	dir := writeImportedInstance(t, t.TempDir(), "authored.opmodel.dev/instance@v0", modPath, "0.1.0",
		"myrel", "default", "{replicas: 2}", nil)
	inst, err = k.AcquireInstanceFromDir(ctx, dir, mustSource(t, k, "/values/a.cue", `image: string`))
	assertWrittenBareTypeReportedOnce(t, "Kernel.AcquireInstanceFromDir", inst, err)
}

// readBody is a module whose component reads #config values into regular
// fields, the normal case in a real module: message twice (once through an
// interpolation), db.host and port. other is read by no component.
const readBody = "#config: {\n" +
	"\tmessage: string\n" +
	"\tother:   int\n" +
	"\tport:    int & >0 | *80\n" +
	"\tdb: {host: string, size: int | *1}\n" +
	"}\n" +
	"debugValues: {}\n" +
	"#components: foo: {metadata: name: \"foo\", metadata: annotations: {" +
	"message: #config.message, again: \"m-\\(#config.message)\", host: #config.db.host, port: \"\\(#config.port)\"}}\n"

// looseBody is readBody's shape with a defect of the module's own: the
// component declares a regular field, loose, that no #config value completes.
const looseBody = "#config: {\n" +
	"\tmessage: string\n" +
	"\tother:   int\n" +
	"}\n" +
	"debugValues: {}\n" +
	"#components: foo: {metadata: name: \"foo\", metadata: annotations: {message: #config.message, loose: string}}\n"

// findingLines returns `<path>: <message>` for every finding of err's CUE
// tree, in the order the kernel returned them (cueerrors.Details sorts by
// position; this does not).
func findingLines(t *testing.T, err error) []string {
	t.Helper()
	var cerr cueerrors.Error
	require.ErrorAs(t, err, &cerr)
	var lines []string
	for _, e := range cueerrors.Errors(cerr) {
		lines = append(lines, e.Error())
	}
	return lines
}

const (
	synthFrame       = `Kernel.SynthesizeInstance: instance "myrel": `
	notConcreteFrame = synthFrame + `not fully concrete: `
	annotations      = "components.foo.metadata.annotations."
)

// kernel-runtime spec, "A value a component reads is named at its field",
// "Every unset value is named, read or not, nested or not", "A component
// defect no unset value explains stays in the report" and "Other values
// defects keep their text": the exact one-line text and the exact findings of
// SynthesizeInstance for each class of defect.
//
// The rows named unchanged carry the text the kernel returned before it
// reported both concreteness checks together (library origin/main d21adc9);
// was holds the earlier one-line text of a row that changed.
func TestKernel_SynthesizeInstance_RefusalText(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		values   string
		was      string
		text     string
		findings []string
	}{
		{
			name: "every required value unset", body: readBody, values: `{}`,
			was:  notConcreteFrame + annotations + "message: incomplete value string (and 2 more errors)",
			text: notConcreteFrame + "values.message: incomplete value string (and 2 more errors)",
			findings: []string{
				"values.message: incomplete value string",
				"values.other: incomplete value int",
				"values.db.host: incomplete value string",
			},
		},
		{
			name: "a value a component reads unset", body: readBody, values: `other: 1, db: host: "h"`,
			was:      notConcreteFrame + annotations + "message: incomplete value string (and 1 more errors)",
			text:     notConcreteFrame + "values.message: incomplete value string",
			findings: []string{"values.message: incomplete value string"},
		},
		{
			name: "a nested value a component reads unset", body: readBody, values: `message: "m", other: 1`,
			was:      notConcreteFrame + annotations + "host: incomplete value string",
			text:     notConcreteFrame + "values.db.host: incomplete value string",
			findings: []string{"values.db.host: incomplete value string"},
		},
		{
			name: "unchanged a value no component reads unset", body: readBody, values: `message: "m", db: host: "h"`,
			text:     notConcreteFrame + "values.other: incomplete value int",
			findings: []string{"values.other: incomplete value int"},
		},
		{
			name: "unchanged a value of the wrong type", body: readBody, values: `message: 7, other: 1, db: host: "h"`,
			text:     synthFrame + "#module.#config.message: conflicting values string and 7 (mismatched types string and int)",
			findings: []string{"#module.#config.message: conflicting values string and 7 (mismatched types string and int)"},
		},
		{
			name: "unchanged a failed constraint", body: readBody, values: `message: "m", other: 1, db: host: "h", port: -1`,
			text: synthFrame + "#module.#config.port: 2 errors in empty disjunction: (and 2 more errors)",
			findings: []string{
				"#module.#config.port: 2 errors in empty disjunction:",
				"#module.#config.port: conflicting values 80 and -1",
				"#module.#config.port: invalid value -1 (out of bound >0)",
			},
		},
		{
			name: "unchanged an undeclared key", body: readBody, values: `message: "m", other: 1, db: host: "h", extra: 1`,
			text:     synthFrame + "field not allowed",
			findings: []string{"field not allowed"},
		},
		{
			name: "unchanged a defaulted field given a bare type", body: readBody, values: `message: "m", other: 1, db: host: "h", port: int`,
			text:     notConcreteFrame + "values.port: incomplete value int",
			findings: []string{"values.port: incomplete value int"},
		},
		{
			name: "unchanged a bare type written for a read value with the rest set", body: readBody, values: `message: string, other: 1, db: host: "h"`,
			text: notConcreteFrame + "values.message: incomplete value string (and 2 more errors)",
			findings: []string{
				"values.message: incomplete value string",
				annotations + "message: incomplete value string",
				"unifiedModule.#" + annotations + "again: invalid interpolation: non-concrete value string (type string)",
			},
		},
		{
			name: "a bare type written for a read value and another value unset", body: readBody, values: `message: string, db: host: "h"`,
			was:  notConcreteFrame + "values.message: incomplete value string (and 2 more errors)",
			text: notConcreteFrame + "values.message: incomplete value string (and 3 more errors)",
			findings: []string{
				"values.message: incomplete value string",
				"values.other: incomplete value int",
				annotations + "message: incomplete value string",
				"unifiedModule.#" + annotations + "again: invalid interpolation: non-concrete value string (type string)",
			},
		},
		{
			name: "unchanged a component defect with every value set", body: looseBody, values: `message: "m", other: 1`,
			text:     notConcreteFrame + annotations + "loose: incomplete value string",
			findings: []string{annotations + "loose: incomplete value string"},
		},
		{
			name: "a component defect and a read value unset", body: looseBody, values: `other: 1`,
			was:  notConcreteFrame + annotations + "message: incomplete value string (and 1 more errors)",
			text: notConcreteFrame + "values.message: incomplete value string (and 1 more errors)",
			findings: []string{
				"values.message: incomplete value string",
				annotations + "loose: incomplete value string",
			},
		},
		{
			name: "a component defect and an unread value unset", body: looseBody, values: `message: "m"`,
			was:  notConcreteFrame + annotations + "loose: incomplete value string",
			text: notConcreteFrame + "values.other: incomplete value int (and 1 more errors)",
			findings: []string{
				"values.other: incomplete value int",
				annotations + "loose: incomplete value string",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, mod := publishSynthModule(t, "demo", "0.1.0", tc.body)
			inst, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
				Module: mod, Name: "myrel", Namespace: "default",
				Values: []kernel.Source{mustSource(t, k, "/values/a.cue", tc.values)},
			})
			require.Error(t, err)
			assert.Nil(t, inst)
			assert.Equal(t, tc.text, err.Error())
			assert.Equal(t, tc.findings, findingLines(t, err))
			assert.NotEqual(t, tc.was, err.Error(), "the refusal still has the text it had before both checks were reported")
		})
	}
}

// kernel-runtime spec, "Every unset value is named, read or not, nested or
// not": each values finding of the empty document carries the position of its
// #config declaration, and the findings name every field that
// ValidateConfigDetailed names for the same source.
func TestKernel_SynthesizeInstance_UnsetValuesPositionedAtConfig(t *testing.T) {
	k, mod := publishSynthModule(t, "demo", "0.1.0", readBody)
	_, err := k.SynthesizeInstance(context.Background(), kernel.InstanceInput{
		Module: mod, Name: "myrel", Namespace: "default",
		Values: []kernel.Source{mustSource(t, k, "/values/a.cue", `{}`)},
	})
	require.Error(t, err)
	for _, path := range []string{"values.message", "values.other", "values.db.host"} {
		assert.True(t, positionsEndWith(err, path, "/module.cue"), "%s is not positioned at its #config declaration: %v", path, err)
	}

	// Every field ValidateConfigDetailed names at #config.<field> for the
	// same source is named here at values.<field>.
	_, vErr := k.ValidateConfigDetailed(mod.ConfigSchema(), []kernel.Source{mustSource(t, k, "/values/a.cue", `{}`)})
	var cerr cueerrors.Error
	require.ErrorAs(t, vErr, &cerr)
	named := 0
	for _, e := range cueerrors.Errors(cerr) {
		field, ok := strings.CutPrefix(strings.Join(e.Path(), "."), "#config.")
		require.True(t, ok, "ValidateConfigDetailed finding outside #config: %v", e)
		assert.True(t, hasErrorPath(err, "values."+field), "values.%s is not named: %v", field, err)
		named++
	}
	assert.Equal(t, 3, named, "ValidateConfigDetailed findings: %v", vErr)
}

// kernel-runtime spec, "A value a component reads is named at its field" and
// "A component defect no unset value explains stays in the report", on the
// directory verb: the package's own values leave message unset, and the
// refusal is the one SynthesizeInstance gives for the same values.
func TestKernel_AcquireInstanceFromDir_ReadRequiredConfigUnset(t *testing.T) {
	const frame = `Kernel.AcquireInstanceFromDir: instance "myrel": not fully concrete: `
	cases := []struct {
		name     string
		body     string
		values   string
		text     string
		findings []string
	}{
		{
			name: "a value a component reads unset", body: readBody, values: `{other: 1, db: host: "h"}`,
			text:     frame + "values.message: incomplete value string",
			findings: []string{"values.message: incomplete value string"},
		},
		{
			name: "a component defect and a read value unset", body: looseBody, values: `{other: 1}`,
			text: frame + "values.message: incomplete value string (and 1 more errors)",
			findings: []string{
				"values.message: incomplete value string",
				annotations + "loose: incomplete value string",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k, _, modPath := publishSynthModuleAt(t, "demo", "0.1.0", tc.body)
			dir := writeImportedInstance(t, t.TempDir(), "authored.opmodel.dev/instance@v0", modPath, "0.1.0",
				"myrel", "default", tc.values, nil)

			inst, err := k.AcquireInstanceFromDir(context.Background(), dir)
			require.Error(t, err)
			assert.Nil(t, inst)
			assert.Equal(t, tc.text, err.Error())
			assert.Equal(t, tc.findings, findingLines(t, err))
			assert.True(t, positionsEndWith(err, "values.message", "/module.cue"),
				"values.message is not positioned at its #config declaration: %v", err)
		})
	}
}
