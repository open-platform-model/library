// web_app — a core@v2 #Module that consumes fixture-catalog Resources,
// Traits, and Blueprints. Drives the plan / match / compile integration tests
// against the opm_platform fixture (which subscribes to
// testing.opmodel.dev/catalogs/opm, the library's v2 fixture catalog).
package web_app

import (
	m "opmodel.dev/core@v2"
	res "testing.opmodel.dev/catalogs/opm/resources"
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
}
