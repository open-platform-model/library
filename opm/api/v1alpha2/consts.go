package v1alpha2

// AnnotationDefaultNamespace is the v1alpha2 annotation key carrying a
// module's suggested default namespace. The annotation is advisory: tooling
// and operators MAY consult it to seed metadata.namespace on a #ModuleRelease,
// but the release remains the authoritative owner of the actual deployed
// namespace. See adr/001-module-default-namespace-as-annotation.md.
const AnnotationDefaultNamespace = "module.opmodel.dev/default-namespace"
