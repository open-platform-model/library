## 1. opm/compat

- [x] 1.1 Add `KindFieldMadeRequired = "field made required"` beside the other kinds.
- [x] 1.2 Add `required(sel cue.Selector, v cue.Value) bool` per design D1; use it for the existing added-field rule.
- [x] 1.3 In `walkStruct`, collect the new side's `(selector, value)` by name first (design D2); in the prior-side loop, when the prior selector is `?` and `required(nextSel, nextVal)`, append `KindFieldMadeRequired` at the path, then continue the walk as before.

## 2. Tests

- [x] 2.1 `TestCheck` table: `y?: string`→`y!: string` (made required); `y?: string`→`y: string` (made required); `y?: string`→`y!: =~"^[a-z]"` (made required + domain narrowed); `y!: string`→`y?: string` (nil); `y: string`→`y: string | *"z"` (nil); `x: string`→`x!: string` (nil: both required).
- [x] 2.2 Regression case from the incident: alpha.5 vs alpha.6 `#ExposeSchema` shapes inline (`name?: string` vs `name!: =~"^[a-z]([a-z0-9-]*[a-z0-9])?$"`), expecting made required + domain narrowed at `name`.

## 3. Enhancement declaration

- [x] 3.1 Create `enhancement.yaml` declaring `implements: [{enhancement: "0010", decisions: [D27]}, {enhancement: "0011", decisions: [D9]}]`.

## 4. Validation gates

- [x] 4.1 `task fmt`
- [x] 4.2 `task vet`
- [x] 4.3 `task lint`
- [x] 4.4 `task test`
