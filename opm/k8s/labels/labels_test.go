package labels_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/library/opm/k8s/labels"
)

// TestVocabularyLiterals pins every key and value to its literal. The
// literals are byte-equal to the frontends' pkg/core/labels.go (cli 1338e700,
// opm-operator 9b83611), which a scripted comparison checked when the
// vocabulary moved here; a change to any of them changes what OPM writes to
// or recognises on a cluster.
func TestVocabularyLiterals(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"ManagedBy", labels.ManagedBy, "app.kubernetes.io/managed-by"},
		{"ManagedByCLI", labels.ManagedByCLI, "opm-cli"},
		{"ManagedByController", labels.ManagedByController, "opm-controller"},
		{"ManagedByLegacy", labels.ManagedByLegacy, "open-platform-model"},
		{"Component", labels.Component, "opmodel.dev/component"},
		{"ComponentName", labels.ComponentName, "component.opmodel.dev/name"},
		{"ModuleInstanceName", labels.ModuleInstanceName, "module-instance.opmodel.dev/name"},
		{"ModuleInstanceNamespace", labels.ModuleInstanceNamespace, "module-instance.opmodel.dev/namespace"},
		{"ModuleInstanceUUID", labels.ModuleInstanceUUID, "module-instance.opmodel.dev/uuid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}

func TestIsOPMManagedBy(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "cli actor", value: "opm-cli", want: true},
		{name: "controller actor", value: "opm-controller", want: true},
		{name: "legacy value", value: "open-platform-model", want: true},
		{name: "empty string", value: "", want: false},
		{name: "Helm", value: "Helm", want: false},
		{name: "helm", value: "helm", want: false},
		{name: "case differs", value: "OPM-CLI", want: false},
		{name: "arbitrary", value: "some-other-tool", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, labels.IsOPMManagedBy(tt.value))
		})
	}
}
