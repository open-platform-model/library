# v1alpha2 — Definition Index

CUE module: `opmodel.dev/core@v1`

---

## Project Structure

```
+-- docs/
```

---

## 

| Definition | File | Description |
|---|---|---|
| `#Blueprint` | `blueprint.cue` | #Blueprint: Defines a reusable blueprint that composes resources and traits into a higher-level abstraction |
| `#BlueprintMap` | `blueprint.cue` |  |
| `#Component` | `component.cue` |  |
| `#ComponentMap` | `component.cue` |  |
| `#LabelWorkloadType` | `component.cue` | Workload type label key |
| `#OpmSecretsComponent` | `helpers_autosecrets.cue` | #OpmSecretsComponent builds the opm-secrets component from grouped secret data |
| `#SecretsResourceFQN` | `helpers_autosecrets.cue` | #SecretsResourceFQN is the canonical FQN for the secrets resource |
| `#Module` | `module.cue` | #Module: The portable application blueprint created by developers and/or platform teams |
| `#ModuleMap` | `module.cue` |  |
| `#ModuleRelease` | `module_release.cue` | #ModuleRelease: The concrete deployment instance Contains: Reference to Module, values, target namespace Users/deployment systems create this to deploy a specific version |
| `#ModuleReleaseMap` | `module_release.cue` |  |
| `#Resource` | `resource.cue` | #Resource: Defines a resource of deployment within the system |
| `#ResourceMap` | `resource.cue` |  |
| `#AutoSecrets` | `schemas.cue` | #AutoSecrets discovers all #Secret instances from a resolved config and groups them by $secretName/$dataKey in one step |
| `#ConfigMapSchema` | `schemas.cue` | #ConfigMapSchema: ConfigMap specification |
| `#ContentHash` | `schemas.cue` | #ContentHash computes a deterministic 10-character hex hash of a string data map |
| `#DiscoverSecrets` | `schemas.cue` | #DiscoverSecrets walks a resolved config (up to 10 levels deep) and collects all fields whose value is a #Secret |
| `#GroupSecrets` | `schemas.cue` | #GroupSecrets takes a flat map of discovered secrets and groups them by $secretName, keyed by $dataKey |
| `#ImmutableName` | `schemas.cue` | #ImmutableName computes the K8s resource name for a ConfigMap |
| `#Secret` | `schemas.cue` | #Secret is the contract type that module authors place on sensitive fields |
| `#SecretContentHash` | `schemas.cue` | #SecretContentHash normalizes #Secret entries and plain strings to a string map, then delegates to #ContentHash |
| `#SecretImmutableName` | `schemas.cue` | #SecretImmutableName computes the K8s resource name for a Secret |
| `#SecretK8sRef` | `schemas.cue` | #SecretK8sRef: points to a pre-existing K8s Secret in the cluster |
| `#SecretLiteral` | `schemas.cue` | #SecretLiteral: user provides the actual value |
| `#SecretSchema` | `schemas.cue` | #SecretSchema: Secret specification for K8s Secret resources |
| `#SecretType` | `schemas.cue` |  |
| `#Trait` | `trait.cue` | #Trait: Defines additional behavior or characteristics that can be attached to components |
| `#TraitMap` | `trait.cue` |  |
| `#ComponentTransformer` | `transformer.cue` | #ComponentTransformer: Declares how to convert OPM components into platform-specific resources |
| `#TransformerContext` | `transformer.cue` | Provider context passed to transformers |
| `#TransformerMap` | `transformer.cue` | Map of transformers by fully qualified name |
| `#ApiVersion` | `types.cue` |  |
| `#BundleFQNType` | `types.cue` | BundleFQNType: FQN for #Bundle — path/name:vN (major version) Example: "opmodel |
| `#FQNType` | `types.cue` | FQNType: primitive definition FQN — path/name@version Example: "opmodel |
| `#KebabToCamel` | `types.cue` | KebabToCamel converts a kebab-case string to camelCase |
| `#KebabToPascal` | `types.cue` | KebabToPascal converts a kebab-case string to PascalCase |
| `#LabelsAnnotationsType` | `types.cue` |  |
| `#MajorVersionType` | `types.cue` | MajorVersionType: major version prefix used in primitive FQNs Example: "v1", "v0" |
| `#ModuleFQNType` | `types.cue` | ModuleFQNType: container-style FQN for #Module — path/name:semver Example: "opmodel |
| `#ModulePathType` | `types.cue` | ModulePathType: plain registry path without embedded version Example: "opmodel |
| `#NameType` | `types.cue` | NameType: RFC 1123 DNS label — lowercase alphanumeric with hyphens, max 63 chars |
| `#UUIDType` | `types.cue` | UUIDType: RFC 4122 UUID in standard format (lowercase hex) |
| `#VersionType` | `types.cue` | Semver 2 |

---

