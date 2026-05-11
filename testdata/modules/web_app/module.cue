// web_app — a v1alpha2 #Module that consumes opm Resources, Traits, and
// Blueprints. Drives the plan / match / compile integration tests against
// the opm-platform fixture (which registers opmodel.dev/modules/opm).
package web_app

import (
	m "opmodel.dev/core/v1alpha2@v1"
	res "opmodel.dev/modules/opm/resources@v1"
)

m.#Module

metadata: {
	modulePath:  "opmodel.dev/library/testdata/modules"
	name:        "web-app"
	version:     "0.1.0"
	description: "Stateless web application fixture exercising opm primitives end-to-end"
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
