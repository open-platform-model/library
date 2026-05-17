package case01

import (
	v1 "opmodel.dev/exp004-02/schemas:v1alpha2"
)

// Hypothesis (positive baseline): the 3-step #ModuleRelease flow correctly
// populates the release/module identity layer of #ctx.runtime (D11) and
// injects #names into every statically-declared component (D32).

_module: v1.#Module & {
	metadata: {
		name:    "myapp"
		version: "1.0.0"
		fqn:     "example.com/modules/myapp:1.0.0"
		uuid:    "00000000-0000-0000-0000-0000000000a1"
	}
	#components: {
		web: {metadata: name: "web"}
	}
}

release: v1.#ModuleRelease & {
	metadata: {
		name:      "myrelease"
		namespace: "myns"
		uuid:      "00000000-0000-0000-0000-000000000001"
	}
	#module: _module
	values: {}
}

// D11 — release/module identity propagates verbatim.
_assertReleaseName: "myrelease" & release.ctx.runtime.release.name
_assertReleaseNs:   "myns" & release.ctx.runtime.release.namespace
_assertModuleName:  "myapp" & release.ctx.runtime.module.name
_assertModuleVer:   "1.0.0" & release.ctx.runtime.module.version
_assertModuleFqn:   "example.com/modules/myapp:1.0.0" & release.ctx.runtime.module.fqn

// D32 — static component gets #names; baseline cascade through dns.fqdn.
_assertWebFqdn: "myrelease-web.myns.svc.cluster.local" & release.components.web.#names.dns.fqdn
