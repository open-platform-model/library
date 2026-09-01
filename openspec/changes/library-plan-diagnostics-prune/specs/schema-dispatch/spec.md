# schema-dispatch Delta

## MODIFIED Requirements

### Requirement: Transformer-context builder

The library SHALL expose `schema.BuildTransformerContext(ctx, rel, compName, schemaComp, runtimeName)` that constructs the `#TransformerContext` value for a single (instance, component, transformer) tuple. The caller's job is to fill the returned value at `schema.Context` on the unified transformer.

The function MUST accept any value implementing `schema.InstanceView` (`InstanceName/Namespace/InstanceUUID/InstanceFQN/ModuleVersion/Labels/Annotations`); the interface SHALL NOT carry a `ModuleFQN` member (the context builder never read it, and the `#moduleInstanceMetadata.fqn` it fills is the instance's own `InstanceFQN`). It MUST surface metadata-decode failures as non-fatal warnings rather than errors.

#### Scenario: Successful context construction
- **WHEN** `BuildTransformerContext` is called with a non-nil context, a valid `InstanceView`, a non-empty `compName`, a schema-preserving component value, and a non-empty `runtimeName`
- **THEN** it returns a `cue.Value` carrying `#moduleInstanceMetadata`, `#componentMetadata`, `#runtimeName` and no error

#### Scenario: Empty runtimeName is fatal
- **WHEN** `BuildTransformerContext` is called with `runtimeName=""`
- **THEN** it returns the zero `cue.Value` and an error

#### Scenario: Bad metadata.labels surfaces as warning
- **WHEN** the supplied `schemaComp` has a `metadata.labels` field that cannot be decoded as `map[string]string`
- **THEN** the returned warnings slice contains a message naming the component and the labels field, and no error is returned
