## 1. `opm/compat` walk

- [x] 1.1 Add `leafIdentical` and call it before the leaf subsume in `walk`; unit test with an `#Image`-shaped guarded struct byte-identical on both sides (was `domain narrowed`, now nothing) and with a changed leaf (still reports)
- [x] 1.2 Thread `underMetadata` through `walkStruct`; skip `provenanceDenylist` fields when set; test with a member reference nested two levels deep whose `catalogVersion` default differs
- [x] 1.3 Add `walkList` for equal-length lists with `name[i]` paths; tests: nested element field removed reports at `list[i].field`; unequal length stays a `domain narrowed` leaf; list-level default still checked

## 2. Fixture regression

- [x] 2.1 Add testdata mirroring `#ExposeTrait` (`appliesTo` with an embedded member carrying `metadata.catalogVersion`) and `#StatefulWorkloadBlueprint` (`composedResources`, `composedTraits`) at two "catalog versions" differing only in provenance; assert `Check` is empty
- [x] 2.2 Same fixture with one genuine narrowing (`spec.expose.name` constrained); assert exactly that one violation

## 3. Spec and docs

- [x] 3.1 Known-limitation paragraph narrowed in the delta spec (measured: unchanged leaves silent; changed `matchN` leaves miss narrowing and false-positive on widening, pinned in `TestCheck`); archive sync carries it into the main spec
- [x] 3.2 Update the `Check` doc comment to state the three rules (provenance at depth, equal-length lists, identical-leaf short-circuit)

## 4. Validation gates

- [x] 4.1 `task fmt`
- [x] 4.2 `task vet`
- [x] 4.3 `task lint`
- [x] 4.4 `task test`
