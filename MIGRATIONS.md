# API history & migration guide

Moved. Migration documentation now lives in [`migrations/`](./migrations/README.md) as
per-change fragments; that README carries the policy (dormant until GA, CI-enforced after).

The pre-GA entries that used to live here were deleted on 2026-08-31: every break was
alpha-to-alpha, both consumers (`cli`, `opm-operator`) migrated in the same PR wave, and the
record survives in [`CHANGELOG.md`](./CHANGELOG.md) and `openspec/changes/archive/`. Git
history has the old text if you need it.
