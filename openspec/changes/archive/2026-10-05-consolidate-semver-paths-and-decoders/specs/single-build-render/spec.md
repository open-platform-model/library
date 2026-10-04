## MODIFIED Requirements

### Requirement: Version skew is detected from the two committed resolutions and the response is caller-configured

For each OPM-namespace path, the kernel SHALL compare the instance module's `cue.mod` requirement against the platform module's tidied entry (never the render module's promoted list). When the instance requires a NEWER build than the platform carries, the configured policy decides: warn-and-render (the default when no policy is supplied) marks that path's resolved-versions row as newer and proceeds; refuse fails the render before evaluation. Newer and older SHALL follow SemVer 2 precedence, prerelease builds included (`2.0.0-beta.2` is older than `2.0.0-beta.10`, which is older than `2.0.0`). A module requiring an OLDER build SHALL produce no such mark; the per-path resolved-versions comparison SHALL always be present in the result as plain data with no severity. The kernel SHALL NOT render the skew as a message string; a frontend formats the row.

#### Scenario: Newer module warns and renders by default

- **WHEN** the instance requires catalog `1.3.0`, the platform carries `1.2.0`, and no policy is supplied
- **THEN** the render proceeds against `1.2.0` and the result's resolved-versions row for that path names both versions and is marked newer, and the result carries no message string for it

#### Scenario: Refuse policy stops the render

- **WHEN** the same skew exists and the caller configured the refuse policy
- **THEN** `Render` fails before evaluation with an error naming the path and both versions

#### Scenario: Older module is data, not a warning

- **WHEN** the instance requires `1.1.0` and the platform carries `1.2.0`
- **THEN** the render proceeds, the resolved-versions row for that path is present in the result's diagnostics, and it is not marked newer

#### Scenario: Prerelease builds compare by SemVer precedence

- **WHEN** the skew comparison compares an instance requiring `2.0.0-beta.10` of a path against a platform carrying `2.0.0-beta.2`, and against a platform carrying `2.0.0`
- **THEN** the row against `2.0.0-beta.2` is marked newer and the row against `2.0.0` is not
