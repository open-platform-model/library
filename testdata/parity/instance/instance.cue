// The deployable artifact, authored the way a user writes one: the module
// enters by IMPORT and the values are concrete. Both renderers consume this
// package: the kernel through Kernel.LoadInstancePackage, the oracle through
// a plain import. That is what makes the comparison meaningful (spec
// render-parity: "Both renderers resolve identical inputs").
//
// Import rather than LookupPath+FillPath keeps #config / #components
// cross-references bound to the scope they were written in, so #names and
// #instance resolve here. Name and namespace match the kernel flow fixture so
// rendered object names are directly comparable; uuid is left for core to
// derive from the fqn on both sides.
package instance

import (
	c "opmodel.dev/core@v2"
	webapp "testing.opmodel.dev/library-parity/web_app"
)

c.#ModuleInstance

metadata: {
	name:      "web-app-demo"
	namespace: "default"
}

#module: webapp

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
