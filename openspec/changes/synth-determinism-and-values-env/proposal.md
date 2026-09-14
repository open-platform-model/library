## Why

Two kernel behaviours contradict the constitution, found by the 2026-09-11 kernel audit and re-confirmed at alpha.30. Synthesis writes `metadata.labels` and `metadata.annotations` into the staged `instance.cue` in Go map iteration order, so identical inputs can produce different bytes and violate Principle I (deterministic given identical inputs). A file-backed values `Source` is loaded through `cue/load` without the kernel's registry environment, so `WithRegistry` is silently ignored for a values file that imports a registry module, contradicting the documented "one mapping every operation resolves through" contract. Both fixes are small and need no design decision from the maintainer.

While threading the environment, the audit found that the omission survived because the kernel has five hand-rolled `load.Instances` sequences; the registry module loader is a near copy of `LoadDir`, the routine documented as "the kernel's one evaluate-and-shape-gate step". Folding it onto `LoadDir` closes the class, not only the instance.

## What Changes

- Synthesis emits labels and annotations in ascending key order. For identical `InstanceInput` the staged `instance.cue` bytes are identical across calls and processes.
- Every path that compiles `Source` values applies the kernel's registry mapping to the file-backed load: `ValidateConfigDetailed`, `AcquireInstanceFromDir` with trailing values, `SynthesizeInstance`, and the internal attribution pass. Non-file sources (compiled from bytes) carry no imports and are unaffected. The process environment is never mutated.
- The registry module loader evaluates and shape-gates the fetched module through `LoadDir` instead of its own copy of the overlay, load, build, gate sequence. No behaviour change intended; the existing loader and kernel tests are the gate.
- Not changed, and why: `renderstage.Build` keeps its own load because it must tolerate an evaluation error (the glue's `gate` field is bottom on refusal while `diagnostics` stays readable) and runs no shape gate; `schema.OCILoader` loads by import path, not directory. The file-backed values load is a single file at its directory, not a package root, so it keeps its own `load.Instances` call and gains the environment.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `instance-synthesis`: the labels and annotations requirement gains a determinism clause and a scenario for byte-identical output under different map insertion orders.
- `config-validation`: adds a requirement that file-backed sources resolve imports through the kernel's registry mapping on every compiling path.
- `kernel-runtime`: the Registry Configuration Option requirement lists values-source compilation among the operations the one mapping covers.
- `registry-module-loading`: the shape-gate requirement states that registry and directory acquisition share one evaluate-and-shape-gate routine, and drops the stale `opm/helper/loader/file` reference.

## Impact

- Packages: `opm/internal/synth` (the instance file writer), `opm/kernel` (source compilation, validation, the values-layered acquire path, synthesis), `opm/internal/loader` (the registry loader).
- Public surface: no signature changes. `ValidateConfigDetailed` and the two values-accepting verbs now honour the receiver's `WithRegistry` mapping for file-backed sources; previously they read the process `CUE_REGISTRY` for that load regardless of the option.
- SemVer: PATCH. Ships as `fix` commits plus one `refactor`; the PR title carries `fix`.
- Downstream: cli and opm-operator need no code change. Both set `WithRegistry` and the process `CUE_REGISTRY` to the same mapping today, so no observable change until a consumer passes a mapping that differs from the process environment, which is exactly the contract the option documents.
- Breaking: none.
- Complexity: none added. Four unexported functions gain an `env []string` parameter; one function loses about twenty lines.
