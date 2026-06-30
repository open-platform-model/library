package inventory

import (
	"os/exec"
	"strings"
	"testing"
)

// TestPackageImportsAreRuntimeNeutral enforces the runtime-neutrality contract
// (spec: Runtime-Neutral Inventory Entry Type): the package's non-test import
// graph MUST exclude controller-runtime, any fluxcd package, and
// apiextensions-apiserver, so that importing it never drags those frameworks
// into a consumer.
func TestPackageImportsAreRuntimeNeutral(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps .: %v\n%s", err, out)
	}

	forbidden := []string{
		"sigs.k8s.io/controller-runtime",
		"github.com/fluxcd/",
		"k8s.io/apiextensions-apiserver",
	}

	for _, line := range strings.Split(string(out), "\n") {
		dep := strings.TrimSpace(line)
		if dep == "" {
			continue
		}
		for _, f := range forbidden {
			if strings.HasPrefix(dep, f) {
				t.Errorf("forbidden dependency in import graph: %q (matched %q)", dep, f)
			}
		}
	}
}
