package renderstage

import (
	"fmt"
	"strings"

	"cuelang.org/go/mod/modfile"
	"github.com/Masterminds/semver/v3"
)

// RenderModulePath is the render module's own identity: a reserved, never
// published module path under a host no registry mapping serves. It never
// resolves anywhere because the render module is always the main module of
// the build it is generated for (0019 D9); it is fixed rather than derived per
// render so generated files are byte-stable across renders of the same
// inputs.
const RenderModulePath = "render.opmodel.dev/build@v0"

// MinLanguageVersion is the floor of the render module's declared
// language.version: v0.17.0 introduced cue.mod/local-module.cue, which
// carries the directory replacements that bring the inputs into the build.
const MinLanguageVersion = "v0.17.0"

// Promotion is the render module's derived dependency list (0019 D13): the
// platform module's tidied list adopted whole, the instance module's list
// unioned in for paths only the instance carries, the platform's entry
// winning every shared path. No tidy-equivalent and no registry consultation
// computes it; it is string-level mechanics over the two committed files.
type Promotion struct {
	// Deps is the promoted dependency list keyed by major-qualified path.
	Deps map[string]Dep

	// Language is the render module's language.version: the maximum of the
	// two inputs' declared versions, floored at MinLanguageVersion.
	Language string

	// Replacements maps each input module's qualified path to the absolute
	// directory cue/load serves it from (the local-module.cue replaceWith).
	Replacements map[string]string
}

// Promote derives the render module's dependency list from the platform's and
// the instance's committed module files. platformDir and instanceDir are the
// absolute directories the two inputs are served from during the build.
func Promote(platform, instance *ModFile, platformDir, instanceDir string) (*Promotion, error) {
	if platform == nil || instance == nil {
		return nil, fmt.Errorf("promotion needs both input module files")
	}
	if platform.Module == instance.Module {
		return nil, fmt.Errorf("platform and instance declare the same module path %q; the render build cannot replace one path with two directories", platform.Module)
	}
	if platformDir == "" || instanceDir == "" {
		return nil, fmt.Errorf("promotion needs both input directories")
	}

	deps := make(map[string]Dep, len(platform.Deps)+len(instance.Deps))
	// Platform list whole, markers intact.
	for path, dep := range platform.Deps {
		deps[path] = dep
	}
	// Instance-only paths join. A default-major marker on an instance-only
	// path survives only when the platform marks no default for the same
	// root path: two majors marked default for one path would be refused by
	// cue/load, and on that disagreement the platform wins (D13).
	platformDefaults := defaultRoots(platform.Deps)
	for path, dep := range instance.Deps {
		if _, shared := deps[path]; shared {
			continue
		}
		if dep.Default && platformDefaults[rootPath(path)] {
			dep.Default = false
		}
		deps[path] = dep
	}

	lang, err := maxLanguage(platform.Language, instance.Language)
	if err != nil {
		return nil, err
	}

	return &Promotion{
		Deps:     deps,
		Language: lang,
		Replacements: map[string]string{
			platform.Module: platformDir,
			instance.Module: instanceDir,
		},
	}, nil
}

// ModuleFile renders the render module's cue.mod/module.cue: identity,
// language version and the promoted dependency list, in modfile's canonical
// format.
func (p *Promotion) ModuleFile() ([]byte, error) {
	f := p.baseFile()
	data, err := modfile.Format(f)
	if err != nil {
		return nil, fmt.Errorf("formatting render module.cue: %w", err)
	}
	return data, nil
}

// LocalModuleFile renders the render module's cue.mod/local-module.cue: the
// main-module dependency view, which is the promoted list plus one
// replace-only placeholder per input directing cue/load to serve that module
// path from its staged directory. cue/load reads this file in place of
// module.cue's deps when present, so the promoted list is repeated here rather
// than patched in.
func (p *Promotion) LocalModuleFile() ([]byte, error) {
	base := p.baseFile()
	local := p.baseFile()
	for path, dir := range p.Replacements {
		if dep, shared := local.Deps[path]; shared {
			// An input that is also a promoted dependency keeps its version
			// and is served from its directory.
			dep.ReplaceWith = dir
			continue
		}
		local.Deps[path] = &modfile.Dep{ReplaceWith: dir}
	}
	data, err := modfile.FormatLocal(local, base)
	if err != nil {
		return nil, fmt.Errorf("formatting render local-module.cue: %w", err)
	}
	return data, nil
}

// baseFile builds the modfile.File view of the promotion (fresh per call so
// the two emitters cannot alias one another's dependency maps).
func (p *Promotion) baseFile() *modfile.File {
	f := &modfile.File{
		Module:   RenderModulePath,
		Language: &modfile.Language{Version: p.Language},
		Deps:     make(map[string]*modfile.Dep, len(p.Deps)),
	}
	for path, dep := range p.Deps {
		f.Deps[path] = &modfile.Dep{Version: dep.Version, Default: dep.Default}
	}
	return f
}

// rootPath strips the major qualifier from a dependency path.
func rootPath(path string) string {
	root, _, _ := strings.Cut(path, "@")
	return root
}

// defaultRoots returns the set of root paths some entry marks as default.
func defaultRoots(deps map[string]Dep) map[string]bool {
	out := map[string]bool{}
	for path, dep := range deps {
		if dep.Default {
			out[rootPath(path)] = true
		}
	}
	return out
}

// maxLanguage returns the later of two declared language versions, floored at
// MinLanguageVersion. An empty declaration counts as the floor.
func maxLanguage(a, b string) (string, error) {
	best, err := semver.NewVersion(MinLanguageVersion)
	if err != nil {
		return "", err
	}
	for _, v := range []string{a, b} {
		if v == "" {
			continue
		}
		parsed, err := semver.NewVersion(v)
		if err != nil {
			return "", fmt.Errorf("invalid language version %q: %w", v, err)
		}
		if parsed.GreaterThan(best) {
			best = parsed
		}
	}
	return "v" + best.String(), nil
}
