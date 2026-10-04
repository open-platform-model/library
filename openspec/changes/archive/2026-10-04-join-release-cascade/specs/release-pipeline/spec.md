## ADDED Requirements

### Requirement: A release notifies the library's downstream repos

The release workflow SHALL have a job `notify-downstream`, the last job of the workflow, that runs the org's `cascade-notify` composite action, pinned to the library's `.github` `main` SHA, with the tag the release-please job reports in `tag_name`. The job SHALL depend on the release-please job only, SHALL run only when that job's `releases_created` output is `true` and the repo variable `CASCADE_NOTIFY` is not `off`, SHALL run on `ubuntu-latest` with a 20-minute timeout, and SHALL be granted only `contents: read`. The key SHALL be read only in the caller-owned `notify-downstream` job, which declares `environment: cascade` and passes `secrets.CASCADE_APP_PRIVATE_KEY` only as the `private-key` input of the SHA-pinned cascade action, with `vars.CASCADE_APP_CLIENT_ID` as `client-id`; that job SHALL have exactly that one step, no checkout or `run:` of its own, and no `env:`, `container:` or `services:`; no reusable call SHALL pass `secrets:` or `secrets: inherit`. The documentation-bundle job SHALL NOT gate it. Which repos it dispatches to, and the Go proxy wait before it does, are the action's behaviour, not the library's. Source: workspace `RELEASING.md`, sections "Notify after publish" and "repository_dispatch"; owner decision 24; Phase 3 wiring contract (version 3.1) §4.6 and §10.1 items 1 and 9.

#### Scenario: A release is cut

- **WHEN** a merged release PR makes release-please tag `v1.0.0-beta.5` and `CASCADE_NOTIFY` is unset
- **THEN** `notify-downstream` runs after the release-please job in the `cascade` Environment and runs `cascade-notify` with `tag: v1.0.0-beta.5`

#### Scenario: No release is cut

- **WHEN** a push to `main` only opens or updates the release PR
- **THEN** `notify-downstream` is skipped

#### Scenario: Notify is switched off

- **WHEN** a release is cut while the repo variable `CASCADE_NOTIFY` is `off`
- **THEN** `notify-downstream` is skipped and no dispatch is sent

#### Scenario: The docs bundle fails

- **WHEN** a release is cut and the `publish-docs` job fails
- **THEN** `notify-downstream` still runs, because it does not depend on `publish-docs`
