// COPY of testdata/modules/web_app/module.cue for the render-parity harness, extended
// with the `worker` component and its #config fields (guarded-env fixture for
// 0019 D14). Everything else is byte-identical to the source; both pin the
// same published core and catalog builds. Edit the source, then refresh the
// shared part of this copy.

// web_app — a core@v2 #Module that consumes opm Resources, Traits, and
// Blueprints from the consolidated catalogs/opm v2 line. Drives the
// plan / match / compile integration tests against the opm_platform fixture
// (which subscribes to opmodel.dev/catalogs/opm).
package web_app

import (
	m "opmodel.dev/core@v2"
	res "opmodel.dev/catalogs/opm/resources/v1beta1"
)

m.#Module

metadata: {
	// v2 identity: modulePath is the FULL module path, major included, and
	// name is its snake_case leaf (enhancement 0010 D1/D8).
	modulePath:  "testing.opmodel.dev/modules/web_app@v1"
	name:        "web_app"
	version:     "1.0.0"
	description: "Stateless web application fixture exercising fixture-catalog primitives end-to-end"
}

#config: {
	image: res.#Image & {
		repository: string | *"nginx"
		tag:        string | *"1.27"
		digest:     string | *""
	}

	replicas: int | *2
	port:     int & >0 & <=65535 | *8080
	hostnames: [...string] | *["web.example.test"]

	// worker (guarded-env fixture): a feature flag that guards an env block,
	// and a passthrough map folded into env by comprehension.
	metrics: bool | *true
	extraEnv: [string]: string
	extraEnv: *{LOG_FORMAT: "json", REGION: "eu-north-1"} | _
}

debugValues: {
	image: {
		repository: "nginx"
		tag:        "1.27"
		digest:     ""
	}
	replicas: 2
	port:     8080
	hostnames: ["web.example.test"]
	metrics: true
	extraEnv: {LOG_FORMAT: "json", REGION: "eu-north-1"}
}
