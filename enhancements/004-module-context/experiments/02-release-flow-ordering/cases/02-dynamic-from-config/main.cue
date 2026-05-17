package case02

import (
	v1 "opmodel.dev/exp004-02/schemas:v1alpha2"
)

// Hypothesis (D34): when a module builds components dynamically via a
// `for ... in #config.servers` comprehension, the 3-step #ModuleRelease flow
// — unify values into #config first, then feed #components to the builder —
// produces correct per-component #names for every materialised entry.

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

release: v1.#ModuleRelease & {
	metadata: {
		name:      "myrelease"
		namespace: "myns"
		uuid:      "00000000-0000-0000-0000-000000000001"
	}
	#module: _module
	values: {
		servers: {
			alpha: port: 8080
			beta: port:  8081
		}
	}
}

// Both dynamically-built components receive a #names injection with the
// correct per-component DNS cascade.
_assertAlphaFqdn: "myrelease-server-alpha.myns.svc.cluster.local" & release.components."server-alpha".#names.dns.fqdn
_assertBetaFqdn:  "myrelease-server-beta.myns.svc.cluster.local" & release.components."server-beta".#names.dns.fqdn

// And both appear in #ctx.runtime.components — the lock-step (D32) holds for
// dynamically-generated components, not just static ones.
_lockstepAlpha: release.components."server-alpha".#names & release.ctx.runtime.components."server-alpha"
_lockstepBeta:  release.components."server-beta".#names & release.ctx.runtime.components."server-beta"
