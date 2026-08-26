module: "testing.opmodel.dev/modules/web_app@v1"
language: {
	version: "v0.17.0"
}
source: {
	kind: "self"
}
deps: {
	"opmodel.dev/catalogs/opm@v2": {
		v: "v2.0.0-alpha.5"
	}
	"opmodel.dev/core@v2": {
		v: "v2.0.0-alpha.6"
	}
}
