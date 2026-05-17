package case03

import (
	v1 "opmodel.dev/exp004-02/schemas:v1alpha2"
)

// Hypothesis (D34 — counter-fixture documenting the silent-failure mode):
// reading `#module.#components` *before* `#config: values` unification yields
// an empty component map for dynamic modules. #ContextBuilder sees no
// components, produces empty `ctx.runtime.components`, and no #names is
// injected for the would-be dynamic entries. Failure is SILENT — no CUE
// error, just missing data — which is why the 3-step ordering matters.
//
// This case asserts the silent-failure shape: the bare-module builder result
// has zero components, while the same module unified with values first
// produces the expected entries.

_module: v1.#Module & {
	metadata: {
		name:    "fleet"
		version: "1.0.0"
		fqn:     "example.com/modules/fleet:1.0.0"
		uuid:    "00000000-0000-0000-0000-0000000000a1"
	}
	#config: {
		servers: [string]: {port: int}
	}
	#components: {
		for _srvName, _c in #config.servers {
			"server-\(_srvName)": {metadata: name: "server-\(_srvName)"}
		}
	}
}

_values: servers: {
	alpha: port: 8080
	beta: port:  8081
}

// WRONG ORDER — feed bare-module components to the builder. #config.servers
// is the pattern constraint with no concrete entries, so the comprehension
// produces zero entries.
_wrongResult: (v1.#ContextBuilder & {
	#release: {
		name:      "myrelease"
		namespace: "myns"
		uuid:      "00000000-0000-0000-0000-000000000001"
	}
	#module: {
		name:    _module.metadata.name
		version: _module.metadata.version
		fqn:     _module.metadata.fqn
		uuid:    _module.metadata.uuid
	}
	#components: _module.#components
}).out

// RIGHT ORDER — values unified into #config first.
_withConfig: _module & {#config: _values}
_rightResult: (v1.#ContextBuilder & {
	#release: {
		name:      "myrelease"
		namespace: "myns"
		uuid:      "00000000-0000-0000-0000-000000000001"
	}
	#module: {
		name:    _withConfig.metadata.name
		version: _withConfig.metadata.version
		fqn:     _withConfig.metadata.fqn
		uuid:    _withConfig.metadata.uuid
	}
	#components: _withConfig.#components
}).out

// Wrong-order yields empty components map (silent failure).
_assertWrongEmpty: 0 & len(_wrongResult.ctx.runtime.components)

// Right-order yields both dynamic entries with correct DNS cascade.
_assertRightCount: 2 & len(_rightResult.ctx.runtime.components)
_assertRightAlpha: "myrelease-server-alpha.myns.svc.cluster.local" & _rightResult.ctx.runtime.components."server-alpha".dns.fqdn
