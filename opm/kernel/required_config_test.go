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
// read field with a constraint (port).
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
		{"a read field unset", `image: "nginx", tag: "v1", any: 3`, true, true},
		{"a key #config does not declare", all + `, extra: 1`, true, true},
		{"an unread field of the wrong type", `image: 5, tag: "v1", any: 3, host: "h"`, true, true},
		{"a read field violating its constraint", all + `, port: 0`, true, true},
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
