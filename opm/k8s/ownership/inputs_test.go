package ownership_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/library/opm/k8s/ownership"
)

// TestVerdictInputsCarryNoOverride pins the fields of the two verdict inputs
// by name. Neither carries a field by which a caller asserts, with no evidence
// on the live object, that a refusal or a skip does not apply: the adopt
// annotation on the live object is the only override (0012:D8:R3). A new
// input field fails here first.
func TestVerdictInputsCarryNoOverride(t *testing.T) {
	assert.Equal(t, []string{"Object", "Live", "InInventory", "InstanceUUID"}, fieldNames(reflect.TypeFor[ownership.ApplyInput]()))
	assert.Equal(t, []string{"Object", "Live", "InstanceUUID"}, fieldNames(reflect.TypeFor[ownership.DeleteInput]()))
}

func fieldNames(t reflect.Type) []string {
	names := make([]string, 0, t.NumField())
	for i := range t.NumField() {
		names = append(names, t.Field(i).Name)
	}
	return names
}
