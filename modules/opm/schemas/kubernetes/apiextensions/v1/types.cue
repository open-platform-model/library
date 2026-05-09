// Kubernetes apiextensions.k8s.io/v1 API group re-exports
package v1

import apiextv1 "cue.dev/x/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

#CustomResourceDefinition:        apiextv1.#CustomResourceDefinition
#CustomResourceDefinitionList:    apiextv1.#CustomResourceDefinitionList
#CustomResourceDefinitionSpec:    apiextv1.#CustomResourceDefinitionSpec
#CustomResourceDefinitionNames:   apiextv1.#CustomResourceDefinitionNames
#CustomResourceDefinitionVersion: apiextv1.#CustomResourceDefinitionVersion
#CustomResourceValidation:        apiextv1.#CustomResourceValidation
#CustomResourceSubresources:      apiextv1.#CustomResourceSubresources
#CustomResourceColumnDefinition:  apiextv1.#CustomResourceColumnDefinition
#JSONSchemaProps:                 apiextv1.#JSONSchemaProps
