## MODIFIED Requirements

### Requirement: A render refuses a platform whose core predates the provider count

`Kernel.Render` SHALL check, before staging, that the platform's core floor is met: that the platform's `#contracts` carries `providedBy`, as recorded when the platform was constructed and reported by `Platform.CoreFloor()` (`platform-artifact`, "A platform records its core floor and contract inventory at construction"). When it does not (the platform module pins a core release older than `2.0.0-alpha.12`, or carries no `#contracts` at all), `Render` SHALL return an error wrapping the typed `PlatformCoreTooOldError` that names the platform, the missing field and the first core release carrying it. The refusal SHALL NOT be a `*RenderError`, SHALL leave no staging directory behind and SHALL NOT fall back to a count of the kernel's own: a render never runs on a platform whose inventory would disagree with it. The check is sound because the render build evaluates the platform module's own core pin, the same one its `Package` was built from. The check SHALL read the recorded floor through `CoreFloor()` and SHALL NOT read the platform's `Package`; apart from it, `Render` reads only the platform's `Metadata` and `Source`. One acquired platform therefore stays shareable, as data, across concurrent `Render` calls on one Kernel. Source: owner decision h4 of the beta.1 kernel checklist walkthrough (2026-10-03).

#### Scenario: An older-core platform is refused before staging

- **WHEN** a platform module identical to a served fixture but pinning core `2.0.0-alpha.10` is acquired and rendered
- **THEN** `Render` returns an error from which `errors.As` extracts a `PlatformCoreTooOldError` naming the platform, the field `providedBy` and the release `2.0.0-alpha.12`, the error is not a `*RenderError`, no result is returned, and no render staging directory remains

#### Scenario: A current-core platform is not affected

- **WHEN** a platform module pinning core `2.0.0-alpha.12` or later is rendered
- **THEN** the core floor raises no error and the render proceeds to staging

#### Scenario: A platform shared by concurrent renders stays race-free

- **WHEN** one acquired current-core platform is shared by several goroutines, each calling `Render` on one Kernel
- **THEN** every render passes the core floor and produces the same objects, with no data race reported under the race detector

#### Scenario: A render reads no platform Package

- **WHEN** a current-core platform is acquired, its `Package` is replaced with the zero `cue.Value`, and it is rendered with an acquired instance
- **THEN** the core floor raises no error and the render produces the same objects as a render of the unchanged platform
