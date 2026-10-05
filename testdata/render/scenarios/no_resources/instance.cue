// A plain CUE package, deliberately NOT a core #ModuleInstance: core's
// #Component declares #resources, so no acquired instance can lack it. The
// render test builds a struct-literal *module.Instance whose Source names
// this directory, to pin that a component without #resources refuses the
// render instead of reading as a component with no demand.
package no_resources

metadata: {
	name:      "no-resources-demo"
	namespace: "default"
}

components: web: metadata: name: "web"

values: {}
