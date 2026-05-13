// Package core defines the platform-neutral contract for OPM-compiled
// artifacts. Each target platform — Kubernetes, docker-compose, Nomad,
// Terraform, Crossplane, ... — provides its own implementation of Resource
// that maps a Compiled output to its native identity.
//
// The library never depends on a specific platform's vocabulary: it produces
// *Compiled values and lets adapters wrap them.
package core

import (
	"cuelang.org/go/cue"
)

// Resource is a platform-compiled artifact carrying both OPM provenance
// and platform-native identity. Adapters (k8s, compose, ...) implement
// this interface by wrapping a Compiled.
type Resource interface {
	// Release is the name of the ModuleRelease that produced this resource.
	Release() string

	// Component is the source component name within the release.
	Component() string

	// Transformer is the FQN of the transformer that produced this resource.
	Transformer() string

	// Value returns the underlying CUE value the transformer produced.
	// Concrete and fully evaluated — safe to encode directly.
	Value() cue.Value

	// Identity returns the platform-native identity of this resource.
	Identity() Identity
}

// Identity is the platform-native handle for a resource: a flat tuple every
// platform fills with its own taxonomy. Unused fields stay zero.
//
// Examples:
//   - k8s Deployment:    {Type: "Deployment", Name: "web", Scope: "default", Group: "apps", Version: "v1"}
//   - compose service:   {Type: "service", Name: "web", Scope: "myproject"}
//   - nomad job:         {Type: "job", Name: "web", Scope: "default"}
//   - terraform aws_*:   {Type: "aws_instance", Name: "web", Group: "aws"}
//   - crossplane XR:     {Type: "XPostgreSQL", Name: "db", Group: "example.org", Version: "v1"}
type Identity struct {
	// Type is the resource type in the platform's taxonomy.
	Type string

	// Name is the platform-local name of the resource.
	Name string

	// Scope is the platform's optional grouping handle (k8s namespace,
	// compose project, nomad namespace, ...).
	Scope string

	// Group is the platform's optional API group / category (k8s API
	// group, terraform provider, ...).
	Group string

	// Version is the platform's optional API version.
	Version string
}

// String returns a human-readable summary using whichever fields are set.
// Format: "[Group/]Type[/Scope]/Name" with empty fields elided.
func (i Identity) String() string {
	typ := i.Type
	if i.Group != "" {
		typ = i.Group + "/" + i.Type
	}
	if i.Scope != "" {
		return typ + "/" + i.Scope + "/" + i.Name
	}
	return typ + "/" + i.Name
}
