package primitives

import (
	"strings"
	t "opmodel.dev/core/v1alpha1/types@v1"
)

// #PolicyRule: Encodes governance rules, security requirements,
// compliance controls, and operational guardrails.
// PolicyRules define what MUST be true, not suggestions.
#PolicyRule: {
	apiVersion: "opmodel.dev/core/v1alpha1"
	kind:       "PolicyRule"

	metadata: {
		modulePath!: t.#ModulePathType   // Example: "opmodel.dev/opm/policies/security"
		version!:    t.#MajorVersionType // Example: "v1"
		name!:       t.#NameType         // Example: "encryption"
		#definitionName: (t.#KebabToPascal & {"in": name}).out

		fqn: t.#FQNType & "\(modulePath)/\(name)@\(version)" // Example: "opmodel.dev/opm/policies/security/encryption@v1"

		description?: string

		labels?:      t.#LabelsAnnotationsType
		annotations?: t.#LabelsAnnotationsType
	}

	// Policy enforcement configuration
	// Note: CUE always validates the structure/schema of the policy spec itself.
	// This field controls WHERE and WHEN the policy is ENFORCED by the platform.
	enforcement!: {
		// When enforcement happens
		// deployment: Enforced when resources are deployed (admission controllers, pre-flight checks)
		// runtime: Enforced continuously while running (monitoring, auditing, ongoing validation)
		// both: Enforced at both deployment time and continuously at runtime
		mode!: "deployment" | "runtime" | "both"

		// What happens when policy is violated
		// block: Reject the operation (deployment fails, request denied)
		// warn: Log warning but allow operation to proceed
		// audit: Record violation for compliance review without blocking
		onViolation!: "block" | "warn" | "audit"

		// Optional: platform-specific enforcement configuration
		// This is where platforms specify HOW to enforce (Kyverno, OPA, admission webhooks, etc.)
		// The structure is intentionally flexible to support different enforcement mechanisms
		platform?: _
	}

	// MUST be an OpenAPIv3 compatible schema
	// The field and schema exposed by this definition
	spec!: (strings.ToCamel(metadata.name)): _
}

#PolicyRuleMap: [string]: _
