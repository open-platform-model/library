package renderstage

import (
	"fmt"
	"sort"

	"github.com/Masterminds/semver/v3"
)

// VersionRow is one resolved-versions comparison row (0019 D18): for an
// OPM-namespace path the instance module requires, the build the instance
// asked for and the build the platform carries. It is plain data with no
// severity; Newer flags the one case D7 makes a policy question.
type VersionRow struct {
	// Path is the major-qualified module path compared.
	Path string

	// ModuleVersion is the version the instance module's cue.mod requires.
	ModuleVersion string

	// PlatformVersion is the version the platform module's tidied list
	// carries. Empty when the platform does not list the path (the instance's
	// own entry is then what the render resolves).
	PlatformVersion string

	// Newer is true when the instance requires a build newer than the
	// platform carries: the D7 skew case the caller's policy decides.
	Newer bool
}

// CompareSkew compares the instance module's committed dependency list against
// the platform module's, per OPM-namespace path the instance requires, and
// returns the rows in lexical path order. The render module's promoted list is
// never an input here: the platform wins every shared path there by
// construction, so skew would be invisible (D18).
func CompareSkew(platform, instance *ModFile) ([]VersionRow, error) {
	if platform == nil || instance == nil {
		return nil, fmt.Errorf("skew comparison needs both input module files")
	}
	rows := make([]VersionRow, 0, len(instance.Deps))
	for path, dep := range instance.Deps {
		if !IsOPMPath(path) {
			continue
		}
		row := VersionRow{Path: path, ModuleVersion: dep.Version}
		if pdep, ok := platform.Deps[path]; ok {
			row.PlatformVersion = pdep.Version
			newer, err := isNewer(dep.Version, pdep.Version)
			if err != nil {
				return nil, fmt.Errorf("comparing %q: %w", path, err)
			}
			row.Newer = newer
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })
	return rows, nil
}

// isNewer reports whether a is a strictly newer SemVer than b.
func isNewer(a, b string) (bool, error) {
	va, err := semver.NewVersion(a)
	if err != nil {
		return false, fmt.Errorf("invalid version %q: %w", a, err)
	}
	vb, err := semver.NewVersion(b)
	if err != nil {
		return false, fmt.Errorf("invalid version %q: %w", b, err)
	}
	return va.GreaterThan(vb), nil
}
