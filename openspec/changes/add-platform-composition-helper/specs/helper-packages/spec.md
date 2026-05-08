## ADDED Requirements

### Requirement: Compose Function

The library SHALL expose `func Compose(k *kernel.Kernel, shell *platform.Platform, modules []*module.Module) (*platform.Platform, error)` in `pkg/helper/platform/`. The function SHALL produce a fully-composed Platform by FillPath-injecting each Module into `shell.Package` at `binding.Paths().Registry[<id>]`, evaluating the result, and returning a new `*Platform`.

#### Scenario: Successful composition

- **WHEN** a caller invokes `Compose(k, shell, []*Module{m1, m2})` where `shell` has an empty `#registry` and `m1`, `m2` register transformers without conflict
- **THEN** the returned `*Platform` has `Package` carrying a `#registry` with two entries (keyed by `m1.Metadata.Name` and `m2.Metadata.Name`)
- **AND** the computed views (`#composedTransformers`, `#matchers`, `#knownResources`, `#knownTraits`) include contributions from both Modules

#### Scenario: ID derived from module metadata name

- **WHEN** a Module is registered
- **THEN** the `#registry` key is the Module's `metadata.name` (kebab-case per catalog 014 D16)

#### Scenario: enabled defaults to true

- **WHEN** a Module is registered without explicit enable/disable instruction
- **THEN** the `#ModuleRegistration.enabled` field is set to `true` explicitly

#### Scenario: Inputs not mutated

- **WHEN** `Compose` is called twice with the same inputs
- **THEN** both calls return semantically identical `*Platform` values
- **AND** the input `shell` and `modules` are unchanged after each call

### Requirement: Multi-Fulfiller Error Surface

When two registered Modules' transformers claim the same primitive FQN (violating catalog 014 D13), `Compose` SHALL return a non-nil `*MultiFulfillerError` carrying the offending FQN(s) and the Module names that contributed conflicting transformers.

#### Scenario: Multi-fulfiller failure

- **WHEN** two Modules each register a transformer with `requiredResources["<fqn>"]`
- **THEN** `Compose` returns a `*MultiFulfillerError`
- **AND** the error includes the FQN, both Module names, and both transformer FQNs

### Requirement: Kernel Convenience Method

The Kernel SHALL expose `(k *Kernel) ComposePlatform(shell *Platform, modules []*Module) (*Platform, error)` delegating to `pkg/helper/platform.Compose`.

#### Scenario: Kernel method matches helper

- **WHEN** a caller invokes `k.ComposePlatform(shell, modules)`
- **THEN** the result is identical to calling `helper/platform.Compose(k, shell, modules)` directly
