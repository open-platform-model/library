## REMOVED Requirements

### Requirement: The helper copy is deprecated and kept until both frontends migrate

**Reason**: Both frontends have migrated: neither cli nor opm-operator imports `opm/helper/objectset` at `main` (cli a12373a5, opm-operator 7474778, checked 2026-10-08). The package is removed before v1.0.0, while a removal is still a prerelease break.

**Migration**: Import `github.com/open-platform-model/library/opm/k8s/object` in place of `github.com/open-platform-model/library/opm/helper/objectset`. `Identity`, `Producer`, `Duplicate`, `Duplicates` and `DuplicateIdentitiesError` keep their names, fields, order and message.
