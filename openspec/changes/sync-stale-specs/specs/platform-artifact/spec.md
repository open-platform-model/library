## REMOVED Requirements

### Requirement: Platform Constructor from cue.Value

**Reason**: It specified `NewPlatformFromValue(k *kernel.Kernel, v cue.Value)` with API version detection, a binding lookup, an `APIVersion` field and `apiversion.ErrUnknownAPIVersion`. None of these exist: the constructor takes the value alone and detects no version. Its scenario "Unknown apiVersion" names an error that cannot occur. The constructor as it exists is restated under "Platform constructor takes a bare value".
**Migration**: None. Callers already call `platform.NewPlatformFromValue(v)` with one argument.

## ADDED Requirements

### Requirement: Platform constructor takes a bare value

The library SHALL expose `func NewPlatformFromValue(v cue.Value) (*Platform, error)` in `opm/platform`. The constructor SHALL take only the platform value. It SHALL decode `Metadata` from the value's `metadata` field, hoist the root-level `type` into `Metadata.Type`, set `Package` to the supplied value unchanged, and leave `Source` nil. It SHALL perform no API version detection and no binding lookup. When `metadata` is absent or does not decode, it SHALL return an error and a nil `*Platform`, never a partial one.

#### Scenario: Successful construction

- **WHEN** a caller invokes `platform.NewPlatformFromValue(v)` with a `#Platform` value carrying `metadata.name`, `metadata.description`, labels, annotations and a root `type: "kubernetes"`
- **THEN** the returned `*Platform` has those fields in `Metadata`, `Metadata.Type == "kubernetes"`, `Package` equal to `v`, and `Source == nil`

#### Scenario: Missing metadata is refused

- **WHEN** a caller invokes `platform.NewPlatformFromValue(v)` with a value that has no `metadata` field
- **THEN** it returns an error stating that the platform metadata field is required, and a nil `*Platform`

#### Scenario: The constructor takes one argument

- **WHEN** a consumer inspects the signature of `platform.NewPlatformFromValue`
- **THEN** it takes a single `cue.Value` and no kernel, context or option argument
- **AND** no `opm/apiversion` package exists to detect a version with
