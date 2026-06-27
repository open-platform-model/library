package schema

// AnnotationDefaultNamespace is the annotation key carrying a module's
// suggested default namespace. The annotation is advisory: tooling and
// operators MAY consult it to seed metadata.namespace on a #ModuleInstance,
// but the instance remains the authoritative owner of the actual deployed
// namespace. See adr/001-module-default-namespace-as-annotation.md.
const AnnotationDefaultNamespace = "module.opmodel.dev/default-namespace"
