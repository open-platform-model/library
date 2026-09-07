## REMOVED Requirements

### Requirement: Loader helpers return only the loaded value

**Reason**: The requirement pinned the signature of three public loader functions and their kernel wrappers; none of them is exported any more. Directory loading is a kernel internal behind the acquire verbs, which return typed, source-carrying artifacts rather than bare values.

**Migration**: Use `Kernel.AcquireModuleFromDir`, `Kernel.AcquirePlatformFromDir` or `Kernel.AcquireInstanceFromDir`; the loaded value is the artifact's `Package` field.
