## 1. Kernel: one concreteness report

- [x] 1.1 `opm/kernel/required_config_test.go`: pin the exact text of every refusal that must not change (wrong type, failed constraint, undeclared key, a defaulted field given a bare type, a written bare type with the rest set, an unread unset value, a component defect with every value set); they pass on the code as it is
- [x] 1.2 Same file: add the failing cases of the delta scenarios (a read value, every unset value on the empty document, a nested value, a component defect next to an unset value, both verbs) and change `TestKernel_InstanceVerbs_BuiltSpecCheckRunsFirst` to the new "reported once" scenario; see them fail on the code as it is
- [x] 1.3 `opm/kernel/process.go`: `processInstance` returns one report of both checks through an unexported helper, as design.md describes
- [x] 1.4 Godoc of `processInstance`, `SynthesizeInstance` and `AcquireInstanceFromDir`: state the report as it is now
- [x] 1.5 Consumer build against throwaway copies of cli and opm-operator `origin/main` with `.tasks/consumer-build.sh`; `task api:diff`
- [x] 1.6 `task check` green, then commit `fix(kernel): name every unset required config value`
