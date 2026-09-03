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
//
// The sibling import carries its major qualifier (@v0). Inside the parity
// module both spellings resolve, but Kernel.Render serves this module to
// the render build through a version-less cue.mod/local-module.cue
// replacement, and cue/load resolves an UNQUALIFIED import of a replaced
// module's package only when the main module lists that path with a major
// (measured 2026-09-03 on cue v0.17.1: "cannot find module providing
// package"). The qualifier keeps one fixture importable by the oracle, the
// old path and Render alike.
package instance

import (
	c "opmodel.dev/core@v2"
	webapp "testing.opmodel.dev/library-parity/web_app@v0"
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
