package labels

// Label keys and values. The values are byte-equal to the copies the cli and
// the operator carried in their pkg/core before the tier existed.
const (
	// ManagedBy is the standard Kubernetes managed-by label key. Core's CUE
	// stamps it on every rendered object with the rendering runtime's value.
	ManagedBy = "app.kubernetes.io/managed-by"

	// ManagedByCLI is the managed-by value of the cli, core's #runtimeName
	// when the cli renders.
	ManagedByCLI = "opm-cli"

	// ManagedByController is the managed-by value of the operator, core's
	// #runtimeName when the operator renders.
	ManagedByController = "opm-controller"

	// ManagedByLegacy is the managed-by value OPM stamped before each runtime
	// stamped its own name. It is recognised for objects applied back then and
	// never stamped on a new object.
	ManagedByLegacy = "open-platform-model"

	// Component is the OPM object category (the cli writes "inventory"), not
	// a component name: it marks what kind of OPM bookkeeping object carries
	// it. Only the cli writes it, on its own inventory objects. The component
	// a rendered object came from is [ComponentName].
	Component = "opmodel.dev/component"

	// ComponentName records the module component that produced a rendered
	// object; its value is the component name. Core's CUE stamps it at
	// render. Inventory records it on each entry as provenance; the stale set
	// ignores it.
	ComponentName = "component.opmodel.dev/name"

	// ModuleInstanceName is the name of the module instance an object
	// belongs to. Core's CUE stamps it at render.
	ModuleInstanceName = "module-instance.opmodel.dev/name"

	// ModuleInstanceNamespace is the namespace of the module instance an
	// object belongs to. Only the cli writes it, on its own inventory
	// objects.
	ModuleInstanceNamespace = "module-instance.opmodel.dev/namespace"

	// ModuleInstanceUUID is the identity UUID of the module instance an
	// object belongs to, used to discover its objects. Core's CUE stamps it
	// at render.
	ModuleInstanceUUID = "module-instance.opmodel.dev/uuid"
)

// Annotation keys.
const (
	// AnnotationAdopt is the adopt annotation: a user sets it on an existing
	// live object to hand that object to a module instance whose apply would
	// otherwise refuse it. Its value is the adopting instance's
	// [ModuleInstanceUUID] value. No OPM runtime writes it; a user writes it
	// by hand (0012:D8:R3).
	AnnotationAdopt = "opmodel.dev/adopt"
)

// IsOPMManagedBy reports whether a managed-by label value identifies an OPM
// runtime: the cli's value, the operator's value, or the legacy value objects
// applied before runtime-owned values still carry. The match is exact and
// case-sensitive.
func IsOPMManagedBy(value string) bool {
	switch value {
	case ManagedByCLI, ManagedByController, ManagedByLegacy:
		return true
	default:
		return false
	}
}
