module: "opmodel.dev/library/testdata/modules/web-app@v1"
language: {
	version: "v0.16.0"
}
source: {
	kind: "self"
}
deps: {
	"cue.dev/x/k8s.io@v0": {
		v: "v0.7.0"
	}
	"opmodel.dev/catalogs/opm@v0": {
		v: "v0.1.0"
	}
	"opmodel.dev/core@v0": {
		v: "v0.3.0"
	}
}
