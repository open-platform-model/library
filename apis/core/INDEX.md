# core — Definition Index

CUE module: `opmodel.dev/core@v1`

---

## Project Structure

```
+-- v1alpha2/
    +-- docs/
    +-- testdata/
```

---

## V1alpha2

| Definition | File | Description |
|---|---|---|
| `#Blueprint` | `v1alpha2/blueprint.cue` | #Blueprint: Defines a reusable blueprint that composes resources and traits into a higher-level abstraction |
| `#BlueprintMap` | `v1alpha2/blueprint.cue` |  |
| `#Component` | `v1alpha2/component.cue` |  |
| `#ComponentMap` | `v1alpha2/component.cue` |  |
| `#LabelWorkloadType` | `v1alpha2/component.cue` | Workload type label key |
| `#OpmSecretsComponent` | `v1alpha2/helpers_autosecrets.cue` | #OpmSecretsComponent builds the opm-secrets component from grouped secret data |
| `#SecretsResourceFQN` | `v1alpha2/helpers_autosecrets.cue` | #SecretsResourceFQN is the canonical FQN for the secrets resource |
| `#Module` | `v1alpha2/module.cue` | #Module: The portable application blueprint created by developers and/or platform teams |
| `#ModuleMap` | `v1alpha2/module.cue` |  |
| `#ModuleRelease` | `v1alpha2/module_release.cue` | #ModuleRelease: The concrete deployment instance Contains: Reference to Module, values, target namespace Users/deployment systems create this to deploy a specific version |
| `#ModuleReleaseMap` | `v1alpha2/module_release.cue` |  |
| `#ModuleRegistration` | `v1alpha2/platform.cue` | #ModuleRegistration — single entry in #Platform |
| `#Platform` | `v1alpha2/platform.cue` | #Platform — registry of registered Modules and their computed projections |
| `#Resource` | `v1alpha2/resource.cue` | #Resource: Defines a resource of deployment within the system |
| `#ResourceMap` | `v1alpha2/resource.cue` |  |
| `#AutoSecrets` | `v1alpha2/schemas.cue` | #AutoSecrets discovers all #Secret instances from a resolved config and groups them by $secretName/$dataKey in one step |
| `#ConfigMapSchema` | `v1alpha2/schemas.cue` | #ConfigMapSchema: ConfigMap specification |
| `#ContentHash` | `v1alpha2/schemas.cue` | #ContentHash computes a deterministic 10-character hex hash of a string data map |
| `#DiscoverSecrets` | `v1alpha2/schemas.cue` | #DiscoverSecrets walks a resolved config (up to 10 levels deep) and collects all fields whose value is a #Secret |
| `#GroupSecrets` | `v1alpha2/schemas.cue` | #GroupSecrets takes a flat map of discovered secrets and groups them by $secretName, keyed by $dataKey |
| `#ImmutableName` | `v1alpha2/schemas.cue` | #ImmutableName computes the K8s resource name for a ConfigMap |
| `#Secret` | `v1alpha2/schemas.cue` | #Secret is the contract type that module authors place on sensitive fields |
| `#SecretContentHash` | `v1alpha2/schemas.cue` | #SecretContentHash normalizes #Secret entries and plain strings to a string map, then delegates to #ContentHash |
| `#SecretImmutableName` | `v1alpha2/schemas.cue` | #SecretImmutableName computes the K8s resource name for a Secret |
| `#SecretK8sRef` | `v1alpha2/schemas.cue` | #SecretK8sRef: points to a pre-existing K8s Secret in the cluster |
| `#SecretLiteral` | `v1alpha2/schemas.cue` | #SecretLiteral: user provides the actual value |
| `#SecretSchema` | `v1alpha2/schemas.cue` | #SecretSchema: Secret specification for K8s Secret resources |
| `#SecretType` | `v1alpha2/schemas.cue` |  |
| `#Trait` | `v1alpha2/trait.cue` | #Trait: Defines additional behavior or characteristics that can be attached to components |
| `#TraitMap` | `v1alpha2/trait.cue` |  |
| `#ComponentTransformer` | `v1alpha2/transformer.cue` | #ComponentTransformer: Declares how to convert OPM components into platform-specific resources |
| `#TransformerContext` | `v1alpha2/transformer.cue` | Provider context passed to transformers |
| `#TransformerMap` | `v1alpha2/transformer.cue` | Map of transformers by fully qualified name |
| `#ApiVersion` | `v1alpha2/types.cue` |  |
| `#BundleFQNType` | `v1alpha2/types.cue` | BundleFQNType: FQN for #Bundle — path/name:vN (major version) Example: "opmodel |
| `#FQNType` | `v1alpha2/types.cue` | FQNType: primitive definition FQN — path/name@version Example: "opmodel |
| `#KebabToCamel` | `v1alpha2/types.cue` | KebabToCamel converts a kebab-case string to camelCase |
| `#KebabToPascal` | `v1alpha2/types.cue` | KebabToPascal converts a kebab-case string to PascalCase |
| `#LabelsAnnotationsType` | `v1alpha2/types.cue` |  |
| `#MajorVersionType` | `v1alpha2/types.cue` | MajorVersionType: major version prefix used in primitive FQNs Example: "v1", "v0" |
| `#ModuleFQNType` | `v1alpha2/types.cue` | ModuleFQNType: container-style FQN for #Module — path/name:semver Example: "opmodel |
| `#ModulePathType` | `v1alpha2/types.cue` | ModulePathType: plain registry path without embedded version Example: "opmodel |
| `#NameType` | `v1alpha2/types.cue` | NameType: RFC 1123 DNS label — lowercase alphanumeric with hyphens, max 63 chars |
| `#UUIDType` | `v1alpha2/types.cue` | UUIDType: RFC 4122 UUID in standard format (lowercase hex) |
| `#VersionType` | `v1alpha2/types.cue` | Semver 2 |

---

