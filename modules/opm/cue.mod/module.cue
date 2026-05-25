module: "opmodel.dev/catalogs/opm@v1"
language: {
	version: "v0.16.0"
}
source: {
	kind: "self"
}
deps: {
	"cue.dev/x/k8s.io@v0": {
		v:       "v0.7.0"
		default: true
	}
	"opmodel.dev/core@v1": {
		v: "v1.0.6"
	}
}
