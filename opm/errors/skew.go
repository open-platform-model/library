package errors

import "fmt"

// SkewError is the refuse-mode diagnostic for catalog version skew
// (enhancement 0019 D7/D18): the instance module's cue.mod requires a NEWER
// build of an OPM-namespace path than the platform module carries, and the
// caller configured the render to refuse rather than warn. The render stops
// before evaluation; the platform's build is what would have executed.
type SkewError struct {
	// Path is the major-qualified module path in skew.
	Path string

	// ModuleVersion is the build the instance module requires.
	ModuleVersion string

	// PlatformVersion is the build the platform module carries.
	PlatformVersion string
}

func (e *SkewError) Error() string {
	return fmt.Sprintf("version skew on %q: module requires %s, platform carries %s (refused by policy)", e.Path, e.ModuleVersion, e.PlatformVersion)
}
