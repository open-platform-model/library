package transformers

import (
	c "opmodel.dev/core/v1alpha2@v1"
	security_resources "opmodel.dev/modules/opm/resources/security@v1"
)

// RoleTransformer converts OPM Role resources to Kubernetes RBAC objects.
// Generates both the role and its binding from a single OPM resource:
//   scope: "namespace" → k8s Role + RoleBinding
//   scope: "cluster"   → k8s ClusterRole + ClusterRoleBinding
#RoleTransformer: c.#ComponentTransformer & {
	metadata: {
		modulePath:  "opmodel.dev/modules/opm/transformers"
		version:     "v1"
		name:        "role-transformer"
		description: "Converts Role resources to Kubernetes RBAC Role/ClusterRole and RoleBinding/ClusterRoleBinding"

		labels: {
			"core.opmodel.dev/resource-category": "security"
			"core.opmodel.dev/resource-type":     "role"
		}
	}

	requiredLabels: {}

	// Required resources - Role resource MUST be present
	requiredResources: {
		"opmodel.dev/modules/opm/resources/security/role@v1": security_resources.#RoleResource
	}

	optionalResources: {}
	requiredTraits: {}
	optionalTraits: {}

	#transform: {
		#component: _
		#context:   c.#TransformerContext

		_role: #component.spec.role

		// Build k8s-shaped rules from OPM PolicyRules
		_k8sRules: [for r in _role.rules {
			apiGroups: r.apiGroups
			resources: r.resources
			verbs:     r.verbs
		}]

		// Build k8s-shaped subjects from CUE-referenced identities
		_k8sSubjects: [for s in _role.subjects {
			kind:      "ServiceAccount"
			name:      s.name
			namespace: #context.#moduleReleaseMetadata.namespace
		}]

		// Common metadata for both objects
		_commonLabels: #context.labels
		_commonAnnotations: {
			if len(#context.componentAnnotations) > 0 {
				#context.componentAnnotations
			}
		}

		output: {
			if _role.scope == "namespace" {
				"Role/\(_role.name)": {
					apiVersion: "rbac.authorization.k8s.io/v1"
					kind:       "Role"
					metadata: {
						name:      _role.name
						namespace: #context.#moduleReleaseMetadata.namespace
						labels:    _commonLabels
						if len(_commonAnnotations) > 0 {
							annotations: _commonAnnotations
						}
					}
					rules: _k8sRules
				}
				"RoleBinding/\(_role.name)": {
					apiVersion: "rbac.authorization.k8s.io/v1"
					kind:       "RoleBinding"
					metadata: {
						name:      _role.name
						namespace: #context.#moduleReleaseMetadata.namespace
						labels:    _commonLabels
						if len(_commonAnnotations) > 0 {
							annotations: _commonAnnotations
						}
					}
					roleRef: {
						apiGroup: "rbac.authorization.k8s.io"
						kind:     "Role"
						name:     _role.name
					}
					subjects: _k8sSubjects
				}
			}

			if _role.scope == "cluster" {
				"ClusterRole/\(_role.name)": {
					apiVersion: "rbac.authorization.k8s.io/v1"
					kind:       "ClusterRole"
					metadata: {
						name:   _role.name
						labels: _commonLabels
						if len(_commonAnnotations) > 0 {
							annotations: _commonAnnotations
						}
					}
					rules: _k8sRules
				}
				"ClusterRoleBinding/\(_role.name)": {
					apiVersion: "rbac.authorization.k8s.io/v1"
					kind:       "ClusterRoleBinding"
					metadata: {
						name:   _role.name
						labels: _commonLabels
						if len(_commonAnnotations) > 0 {
							annotations: _commonAnnotations
						}
					}
					roleRef: {
						apiGroup: "rbac.authorization.k8s.io"
						kind:     "ClusterRole"
						name:     _role.name
					}
					subjects: _k8sSubjects
				}
			}
		}
	}
}

/////////////////////////////////////////////////////////////////
//// Test Data
/////////////////////////////////////////////////////////////////

// Test: namespace-scoped role
_testNsRoleComponent: security_resources.#Role & {
	spec: role: {
		name:  "pod-reader"
		scope: "namespace"
		rules: [{
			apiGroups: [""]
			resources: ["pods"]
			verbs: ["get", "list", "watch"]
		}]
		subjects: [{
			name:           "ci-bot"
			automountToken: false
		}]
	}
}

_testNsRoleTransformer: (#RoleTransformer.#transform & {
	#component: _testNsRoleComponent
	#context: {
		namespace: "default"
		labels: app: "ci-bot"
		componentAnnotations: {}
	}
}).output

// Test: cluster-scoped role
_testClusterRoleComponent: security_resources.#Role & {
	spec: role: {
		name:  "cluster-reader"
		scope: "cluster"
		rules: [{
			apiGroups: [""]
			resources: ["namespaces"]
			verbs: ["get", "list"]
		}]
		subjects: [{
			name:           "admin-bot"
			automountToken: false
		}]
	}
}

_testClusterRoleTransformer: (#RoleTransformer.#transform & {
	#component: _testClusterRoleComponent
	#context: {
		namespace: "kube-system"
		labels: app: "admin-bot"
		componentAnnotations: {}
	}
}).output
