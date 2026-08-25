// The flow fixture's #ModuleInstance, authored the way a user writes one: the
// module enters by IMPORT and the values are concrete. Kernel.LoadInstancePackage
// loads this package; ProcessModuleInstance derives everything else.
//
// Import rather than LookupPath+FillPath keeps #config / #components
// cross-references bound to the scope they were written in, so every
// component's #instance and #names resolve (web.#names.dns.fqdn is
// "web.default.svc.cluster.local"). uuid is not authored: core derives it from
// the instance fqn (0019 D3; spec transform-input-fill).
//
// Intra-module import: this package lives inside the web_app fixture module,
// so it needs no registry and no second cue.mod. LoadModulePackage(web_app)
// loads only the root package; this directory does not change what the module
// fixture is.
package instance

import (
	c "opmodel.dev/core@v2"
	webapp "testing.opmodel.dev/modules/web_app@v1"
)

c.#ModuleInstance

metadata: {
	name:      "web-app-demo"
	namespace: "default"
}

#module: webapp

// Mirrors the module's debugValues.
values: {
	image: {
		repository: "nginx"
		tag:        "1.27"
		digest:     ""
	}
	replicas: 2
	port:     8080
	hostnames: ["web.example.test"]
}
