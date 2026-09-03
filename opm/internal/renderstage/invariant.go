package renderstage

import (
	"fmt"
	"sort"
)

// CoverageError is the D13 refusal: the written render module lists no entry
// for an OPM-namespace path one of the inputs requires, so cue/load would
// answer that path from the module graph's maximum-version selection instead
// of from the render module's own roots. It is a kernel defect by definition;
// no caller can configure it away.
type CoverageError struct {
	// Path is the uncovered major-qualified module path.
	Path string

	// RequiredBy names the input(s) whose module file lists the path.
	RequiredBy []string
}

func (e *CoverageError) Error() string {
	return fmt.Sprintf("render module does not cover OPM path %q required by %v: promotion is defective (kernel defect, not a policy)", e.Path, e.RequiredBy)
}

// VerifyCoverage re-parses the render module's written module.cue and refuses
// when any OPM-namespace path present in either input's dependency list is
// absent from it. inputs maps a label ("platform", "instance") to the
// committed module file it was promoted from. The first uncovered path in
// lexical order is reported.
func VerifyCoverage(written []byte, filename string, inputs map[string]*ModFile) error {
	rendered, err := ParseModFile(written, filename)
	if err != nil {
		return fmt.Errorf("re-parsing the render module file: %w", err)
	}
	required := map[string][]string{}
	for label, mf := range inputs {
		if mf == nil {
			continue
		}
		for path := range mf.Deps {
			if IsOPMPath(path) {
				required[path] = append(required[path], label)
			}
		}
	}
	paths := make([]string, 0, len(required))
	for path := range required {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if _, ok := rendered.Deps[path]; ok {
			continue
		}
		by := required[path]
		sort.Strings(by)
		return &CoverageError{Path: path, RequiredBy: by}
	}
	return nil
}
