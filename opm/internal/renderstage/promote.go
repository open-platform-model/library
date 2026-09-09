package renderstage

import (
	"fmt"
	"sort"
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
// winning every shared path, and each input module's own path entered as a
// replace-only, default-marked entry. No tidy-equivalent and no registry
// consultation computes it; it is string-level mechanics over the two
// committed files.
type Promotion struct {
	// Deps is the promoted dependency list keyed by major-qualified path. It
	// includes the two input modules themselves: cue/load resolves an
	// unqualified import inside a dependency (a module importing its own
	// subpackage, "example.com/mod/identity" from within example.com/mod@v0,
	// the ordinary authoring shape) through the MAIN module's default-major
	// table, which it reads from cue.mod/module.cue only. A replacement in
	// local-module.cue carries no default of its own, so without this entry
	// the input's self-imports fail with "cannot find module providing
	// package". The entry's version is a placeholder ([ReplacedVersion]):
	// the directory replacement serves the module and the version is never
	// resolved; the modfile decoder refuses a null version.
	Deps map[string]Dep

	// Language is the render module's language.version: the maximum of the
	// two inputs' declared versions, floored at MinLanguageVersion.
	Language string

	// Replacements maps each replaced qualified path to its target (the
	// local-module.cue replaceWith): each input module's own path to the
	// absolute directory cue/load serves it from, plus every promoted
	// replacement from the inputs' own local views, an absolute directory
	// or a module path.
	Replacements map[string]string

	// Rows is one row per promoted local replacement, in path order: the
	// developer's redirections the render honours, for the kernel to report
	// as data. The two input directories are the mechanism, not a row.
	Rows []ReplacementRow
}

// ReplacementRow names one local replacement the render module honours: the
// replaced major-qualified path, its target (an absolute directory, or a
// module path for a module replacement) and the input whose
// cue.mod/local-module.cue supplied it.
type ReplacementRow struct {
	Path   string
	Target string

	// By is "platform" or "instance".
	By string
}

// Input labels on a ReplacementRow and in promotion errors.
const (
	byPlatform = "platform"
	byInstance = "instance"
)

// Promote derives the render module's dependency list from the platform's and
// the instance's committed module files, and its main-module view from their
// local views (nil when an input carries no cue.mod/local-module.cue, or
// when the caller did not enable local replacements). platformDir and
// instanceDir are the absolute directories the two inputs are served from
// during the build.
//
// Local replacements promote under the precedence dependencies do: the
// platform's whole, the instance's only for paths the platform's dependency
// list does not name (an instance replacement on a platform-named path is
// inert: the platform decides which bytes execute for every path it names).
// Entries a local file lists that the promoted list lacks join it, so a
// module-path replacement's target arrives with its version. Every replaced
// path is listed with a placeholder version of its major when the inputs
// give it none, so the coverage invariant holds for a replaced OPM path; a
// version-less dependency no promoted replacement covers is refused.
func Promote(platform, instance *ModFile, platformLocal, instanceLocal *LocalModFile, platformDir, instanceDir string) (*Promotion, error) {
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

	// Local replacements, platform whole, instance on instance-only paths.
	// A replacement of either input's own path is refused: the inputs are
	// served from their staged directories by construction.
	replacements := map[string]string{}
	var rows []ReplacementRow
	views := []struct {
		by    string
		local *LocalModFile
	}{{byPlatform, platformLocal}, {byInstance, instanceLocal}}
	for _, v := range views {
		if v.local == nil {
			continue
		}
		for _, path := range sortedPaths(v.local.Replacements) {
			if path == platform.Module || path == instance.Module {
				return nil, fmt.Errorf("%s %s replaces %q, a render input; inputs are served from their own directories", v.by, LocalModFileName, path)
			}
			if v.by == byInstance {
				if _, named := platform.Deps[path]; named {
					continue
				}
			}
			replacements[path] = v.local.Replacements[path]
			rows = append(rows, ReplacementRow{Path: path, Target: v.local.Replacements[path], By: v.by})
		}
		for path, dep := range v.local.Deps {
			if _, listed := deps[path]; listed {
				continue
			}
			if dep.Default && defaultRoots(deps)[rootPath(path)] {
				dep.Default = false
			}
			deps[path] = dep
		}
	}
	for path := range replacements {
		dep := deps[path]
		if dep.Version == "" {
			v, err := ReplacedVersion(path)
			if err != nil {
				return nil, err
			}
			dep.Version = v
		}
		deps[path] = dep
	}
	// A version-less entry no promoted replacement covers would only fail
	// later, inside modfile formatting, with an error naming neither the
	// path nor the input.
	for _, path := range sortedPaths(deps) {
		if deps[path].Version != "" {
			continue
		}
		by := byInstance
		if _, ok := platform.Deps[path]; ok {
			by = byPlatform
		}
		return nil, fmt.Errorf("%s dependency %q carries no version and no promoted local replacement covers it", by, path)
	}

	// Each input module's own path: a replace-only entry marked default so
	// the input's unqualified self-imports resolve, the platform first. The
	// default yields (as above) when some promoted entry already marks the
	// same root path default; an input that is also a promoted dependency
	// keeps its version.
	for _, in := range []*ModFile{platform, instance} {
		dep, listed := deps[in.Module]
		if !listed {
			v, err := ReplacedVersion(in.Module)
			if err != nil {
				return nil, err
			}
			dep = Dep{Version: v}
		}
		if !dep.Default && !defaultRoots(deps)[rootPath(in.Module)] {
			dep.Default = true
		}
		deps[in.Module] = dep
	}
	replacements[platform.Module] = platformDir
	replacements[instance.Module] = instanceDir

	lang, err := maxLanguage(platform.Language, instance.Language)
	if err != nil {
		return nil, err
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })
	return &Promotion{
		Deps:         deps,
		Language:     lang,
		Replacements: replacements,
		Rows:         rows,
	}, nil
}

// sortedPaths returns the keys of a path-keyed map in lexical order.
func sortedPaths[V any](m map[string]V) []string {
	paths := make([]string, 0, len(m))
	for path := range m {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
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
// main-module dependency view, which is the promoted list with each input's
// entry directing cue/load to serve that module path from its staged
// directory and every promoted local replacement written as the input wrote
// it. cue/load reads this file in place of module.cue's deps when present,
// so the promoted list is repeated here rather than patched in.
func (p *Promotion) LocalModuleFile() ([]byte, error) {
	base := p.baseFile()
	local := p.baseFile()
	for path, target := range p.Replacements {
		if dep, listed := local.Deps[path]; listed {
			// Promote lists every replaced path; the entry keeps its
			// version and marker and is served from its target.
			dep.ReplaceWith = target
			continue
		}
		// A hand-built Promotion without the entry: replace-only.
		local.Deps[path] = &modfile.Dep{ReplaceWith: target}
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

// ReplacedVersion is the placeholder version the render module lists for an
// input module it serves from a directory: "vN.0.0" for a path qualified
// "@vN". cue/load never resolves it (the replacement wins), but module.cue
// must carry a well-formed version of the entry's own major for the entry,
// and so its default-major marker, to be accepted.
func ReplacedVersion(qualifiedPath string) (string, error) {
	_, major, ok := strings.Cut(qualifiedPath, "@")
	if !ok || !strings.HasPrefix(major, "v") || strings.ContainsAny(major, "./") {
		return "", fmt.Errorf("module path %q carries no major qualifier", qualifiedPath)
	}
	if _, err := semver.NewVersion(major + ".0.0"); err != nil {
		return "", fmt.Errorf("module path %q: %w", qualifiedPath, err)
	}
	return major + ".0.0", nil
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
