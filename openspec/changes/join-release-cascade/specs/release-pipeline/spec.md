## ADDED Requirements

### Requirement: A release notifies the library's downstream repos

The release workflow SHALL have a job `notify-downstream` that calls the org's shared `cascade-notify.yml` reusable workflow at `@main` with the tag the release-please job reports in `tag_name`. The job SHALL depend on the release-please job only, SHALL run only when that job's `releases_created` output is `true` and the repo variable `CASCADE_NOTIFY` is not `off`, and SHALL grant the called workflow no more than `contents: read`. It SHALL pass no secrets: the App key is an Environment secret that only the reusable workflow's own `cascade` job reads. The documentation-bundle job SHALL NOT gate it. Which repos it dispatches to, and the Go proxy wait before it does, are the reusable workflow's behaviour, not the library's. Source: workspace `RELEASING.md`, sections "Notify after publish" and "repository_dispatch"; owner selection 13 ("@main + ruleset").

#### Scenario: A release is cut

- **WHEN** a merged release PR makes release-please tag `v1.0.0-beta.4` and `CASCADE_NOTIFY` is unset
- **THEN** `notify-downstream` runs after the release-please job and calls `cascade-notify.yml` with `tag: v1.0.0-beta.4`

#### Scenario: No release is cut

- **WHEN** a push to `main` only opens or updates the release PR
- **THEN** `notify-downstream` is skipped

#### Scenario: Notify is switched off

- **WHEN** a release is cut while the repo variable `CASCADE_NOTIFY` is `off`
- **THEN** `notify-downstream` is skipped and no dispatch is sent

#### Scenario: The docs bundle fails

- **WHEN** a release is cut and the `publish-docs` job fails
- **THEN** `notify-downstream` still runs, because it does not depend on `publish-docs`
