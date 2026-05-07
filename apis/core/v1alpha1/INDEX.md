# v1alpha1 — Definition Index

CUE module: `opmodel.dev/core/v1alpha1@v1`

---

## Project Structure

```
+-- bundle/
+-- bundlerelease/
+-- component/
+-- helpers/
+-- matcher/
+-- module/
+-- modulerelease/
+-- policy/
+-- primitives/
+-- provider/
+-- schemas/
+-- transformer/
+-- types/
```

---

## Bundle

| Definition | File | Description |
|---|---|---|
| `#Bundle` | `bundle/bundle.cue` | #Bundle: Defines a collection of modules grouped for distribution |
| `#BundleDefinitionMap` | `bundle/bundle.cue` |  |
| `#BundleInstance` | `bundle/bundle.cue` | #BundleInstance: A single module instance within a #Bundle |
| `#Module` | `bundle/bundle.cue` | Local alias — workaround: CUE's import tracker does not always see module |

---

## Bundlerelease

| Definition | File | Description |
|---|---|---|
| `#BundleRelease` | `bundlerelease/bundle_release.cue` | #BundleRelease: The concrete deployment instance for a #Bundle |
| `#BundleReleaseMap` | `bundlerelease/bundle_release.cue` |  |

---

## Component

| Definition | File | Description |
|---|---|---|
| `#Component` | `component/component.cue` |  |
| `#ComponentMap` | `component/component.cue` |  |
| `#LabelWorkloadType` | `component/component.cue` | Workload type label key |

---

## Helpers

| Definition | File | Description |
|---|---|---|
| `#OpmSecretsComponent` | `helpers/autosecrets.cue` | #OpmSecretsComponent builds the opm-secrets component from grouped secret data |
| `#SecretsResourceFQN` | `helpers/autosecrets.cue` | #SecretsResourceFQN is the canonical FQN for the secrets resource |

---

## Matcher

| Definition | File | Description |
|---|---|---|
| `#MatchPlan` | `matcher/matcher.cue` | #MatchPlan evaluates every (component, transformer) pair and computes:   - matches:         full result matrix — used both to drive rendering and for diagnostics   - unmatched:       component names that had zero matching transformers (error condition)   - unhandledTraits: per component, traits present but not handled by any matched transformer (warning) Inputs are raw values; #MatchPlan has no coupling to #ModuleRelease |
| `#MatchResult` | `matcher/matcher.cue` | #MatchResult captures the outcome of evaluating a single (component, transformer) pair |

---

## Module

| Definition | File | Description |
|---|---|---|
| `#Module` | `module/module.cue` | #Module: The portable application blueprint created by developers and/or platform teams |
| `#ModuleMap` | `module/module.cue` |  |

---

## Modulerelease

| Definition | File | Description |
|---|---|---|
| `#ModuleRelease` | `modulerelease/module_release.cue` | #ModuleRelease: The concrete deployment instance Contains: Reference to Module, values, target namespace Users/deployment systems create this to deploy a specific version |
| `#ModuleReleaseMap` | `modulerelease/module_release.cue` |  |

---

## Policy

| Definition | File | Description |
|---|---|---|
| `#Policy` | `policy/policy.cue` | #Policy: Groups PolicyRules and Directives and targets them to a set of components via label matching or explicit references |
| `#PolicyMap` | `policy/policy.cue` |  |

---

## Primitives

| Definition | File | Description |
|---|---|---|
| `#Blueprint` | `primitives/blueprint.cue` | #Blueprint: Defines a reusable blueprint that composes resources and traits into a higher-level abstraction |
| `#BlueprintMap` | `primitives/blueprint.cue` |  |
| `#Directive` | `primitives/directive.cue` | #Directive: Describes operational behavior that the platform should execute on behalf of the module author |
| `#DirectiveMap` | `primitives/directive.cue` |  |
| `#PolicyRule` | `primitives/policy_rule.cue` | #PolicyRule: Encodes governance rules, security requirements, compliance controls, and operational guardrails |
| `#PolicyRuleMap` | `primitives/policy_rule.cue` |  |
| `#Resource` | `primitives/resource.cue` | #Resource: Defines a resource of deployment within the system |
| `#ResourceMap` | `primitives/resource.cue` |  |
| `#Trait` | `primitives/trait.cue` | #Trait: Defines additional behavior or characteristics that can be attached to components |
| `#TraitMap` | `primitives/trait.cue` |  |

---

## Provider

| Definition | File | Description |
|---|---|---|
| `#Provider` | `provider/provider.cue` |  |

---

## Schemas

| Definition | File | Description |
|---|---|---|
| `#AutoSecrets` | `schemas/schemas.cue` | #AutoSecrets discovers all #Secret instances from a resolved config and groups them by $secretName/$dataKey in one step |
| `#ConfigMapSchema` | `schemas/schemas.cue` | #ConfigMapSchema: ConfigMap specification |
| `#ContentHash` | `schemas/schemas.cue` | #ContentHash computes a deterministic 10-character hex hash of a string data map |
| `#DiscoverSecrets` | `schemas/schemas.cue` | #DiscoverSecrets walks a resolved config (up to 10 levels deep) and collects all fields whose value is a #Secret |
| `#GroupSecrets` | `schemas/schemas.cue` | #GroupSecrets takes a flat map of discovered secrets and groups them by $secretName, keyed by $dataKey |
| `#ImmutableName` | `schemas/schemas.cue` | #ImmutableName computes the K8s resource name for a ConfigMap |
| `#Secret` | `schemas/schemas.cue` | #Secret is the contract type that module authors place on sensitive fields |
| `#SecretContentHash` | `schemas/schemas.cue` | #SecretContentHash normalizes #Secret entries and plain strings to a string map, then delegates to #ContentHash |
| `#SecretImmutableName` | `schemas/schemas.cue` | #SecretImmutableName computes the K8s resource name for a Secret |
| `#SecretK8sRef` | `schemas/schemas.cue` | #SecretK8sRef: points to a pre-existing K8s Secret in the cluster |
| `#SecretLiteral` | `schemas/schemas.cue` | #SecretLiteral: user provides the actual value |
| `#SecretSchema` | `schemas/schemas.cue` | #SecretSchema: Secret specification for K8s Secret resources |
| `#SecretType` | `schemas/schemas.cue` |  |

---

## Transformer

| Definition | File | Description |
|---|---|---|
| `#Transformer` | `transformer/transformer.cue` | #Transformer: Declares how to convert OPM components into platform-specific resources |
| `#TransformerContext` | `transformer/transformer.cue` | Provider context passed to transformers |
| `#TransformerMap` | `transformer/transformer.cue` | Map of transformers by fully qualified name |

---

## Types

| Definition | File | Description |
|---|---|---|
| `#BundleFQNType` | `types/types.cue` | BundleFQNType: FQN for #Bundle — path/name:vN (major version) Example: "opmodel |
| `#FQNType` | `types/types.cue` | FQNType: primitive definition FQN — path/name@version Example: "opmodel |
| `#KebabToCamel` | `types/types.cue` | KebabToCamel converts a kebab-case string to camelCase |
| `#KebabToPascal` | `types/types.cue` | KebabToPascal converts a kebab-case string to PascalCase |
| `#LabelsAnnotationsType` | `types/types.cue` |  |
| `#MajorVersionType` | `types/types.cue` | MajorVersionType: major version prefix used in primitive FQNs Example: "v1", "v0" |
| `#ModuleFQNType` | `types/types.cue` | ModuleFQNType: container-style FQN for #Module — path/name:semver Example: "opmodel |
| `#ModulePathType` | `types/types.cue` | ModulePathType: plain registry path without embedded version Example: "opmodel |
| `#NameType` | `types/types.cue` | NameType: RFC 1123 DNS label — lowercase alphanumeric with hyphens, max 63 chars |
| `#UUIDType` | `types/types.cue` | UUIDType: RFC 4122 UUID in standard format (lowercase hex) |
| `#VersionType` | `types/types.cue` | Semver 2 |

---

