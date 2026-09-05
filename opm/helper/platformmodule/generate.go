package platformmodule

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	"cuelang.org/go/mod/modfile"

	"github.com/open-platform-model/library/opm/schema"
)

const (
	// CorePath is the major-qualified module path of the core schema the
	// generated module embeds.
	CorePath = "opmodel.dev/core@v2"

	// LanguageVersion is the generated module's declared CUE language
	// version: the floor every published first-party module declares and the
	// render build requires for cue.mod/local-module.cue.
	LanguageVersion = "v0.17.0"

	// ModuleFileName and PlatformFileName are the two files a generated
	// module consists of, relative to the module directory.
	ModuleFileName   = "cue.mod/module.cue"
	PlatformFileName = "platform.cue"
)

// Entry is one catalog subscription in the shape Generate consumes: the
// major-qualified catalog path (the registry key), the bare SemVer build the
// subscription names and whether the subscription is enabled.
type Entry struct {
	Path    string
	Version string
	Enable  bool
}

// Dep is one pinned dependency of the generated cue.mod: a major-qualified
// module path and a canonical "v"-prefixed version.
type Dep struct {
	Path    string
	Version string
}

// Input is everything Generate needs. Name and Type are the platform's
// metadata.name and type; ModulePath is the generated module's own identity
// (a reserved, never-published platforms path such as
// "opmodel.dev/platforms/cluster@v0", 0019 D6); Entries are the catalog
// subscriptions; Deps is the resolved dependency closure (see Closure), which
// MUST contain a pin for core and for every entry's catalog.
type Input struct {
	Name       string
	Type       string
	ModulePath string
	Entries    []Entry
	Deps       []Dep
}

// Files maps a path relative to the module directory to the file's bytes.
type Files map[string][]byte

// Roots returns the dependency roots the closure is derived from: the core
// pin plus every entry's catalog, disabled entries included (a disabled entry
// still imports its catalog). Core is pinned at [schema.DefaultSchemaVersion],
// the release the kernel was verified against; a caller that needs a
// different core build assembles its []Dep roots directly. Versions are
// canonicalised with the "v" prefix cue.mod requires; subscriptions carry
// bare SemVer.
func Roots(entries []Entry) []Dep {
	roots := make([]Dep, 0, len(entries)+1)
	roots = append(roots, Dep{Path: CorePath, Version: canonicalVersion(schema.DefaultSchemaVersion())})
	for _, e := range entries {
		roots = append(roots, Dep{Path: e.Path, Version: canonicalVersion(e.Version)})
	}
	sortDeps(roots)
	return roots
}

// Generate renders the module's two files from in. It is pure and
// deterministic: entries and dependencies are emitted in sorted path order
// whatever order they arrive in, so the same input always produces
// byte-identical content. Each registry entry stamps the subscription's
// version as the entry's expected `version`, which unifies with the schema's
// readout of the imported catalog so wrong bytes are a build conflict naming
// the entry (0019 D13 tripwire).
func Generate(in Input) (Files, error) {
	if in.Name == "" {
		return nil, errors.New("platform name is required")
	}
	if in.Type == "" {
		return nil, errors.New("platform type is required")
	}
	if in.ModulePath == "" {
		return nil, errors.New("module path is required")
	}

	entries := append([]Entry(nil), in.Entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	for i := 1; i < len(entries); i++ {
		if entries[i].Path == entries[i-1].Path {
			return nil, fmt.Errorf("duplicate registry entry %q", entries[i].Path)
		}
	}

	pinned := make(map[string]string, len(in.Deps))
	for _, d := range in.Deps {
		if d.Path == "" || d.Version == "" {
			return nil, fmt.Errorf("dependency %q has an empty path or version", d.Path)
		}
		if prev, dup := pinned[d.Path]; dup && prev != d.Version {
			return nil, fmt.Errorf("dependency %q pinned twice (%s and %s)", d.Path, prev, d.Version)
		}
		pinned[d.Path] = d.Version
	}
	if _, ok := pinned[CorePath]; !ok {
		return nil, fmt.Errorf("dependency closure does not pin %s", CorePath)
	}
	for _, e := range entries {
		if e.Path == "" || e.Version == "" {
			return nil, fmt.Errorf("registry entry %q has an empty path or version", e.Path)
		}
		if _, ok := pinned[e.Path]; !ok {
			return nil, fmt.Errorf("dependency closure does not pin registry entry %s", e.Path)
		}
	}

	moduleFile, err := renderModuleFile(in.ModulePath, pinned)
	if err != nil {
		return nil, err
	}
	return Files{
		ModuleFileName:   moduleFile,
		PlatformFileName: renderPlatformFile(in.Name, in.Type, entries),
	}, nil
}

// renderModuleFile emits cue.mod/module.cue in modfile's canonical format,
// which sorts dependencies by path. No dependency carries a default-major
// marker: the platform imports nothing unqualified, and this matches what
// `cue mod tidy` writes for the same roots.
func renderModuleFile(modulePath string, pinned map[string]string) ([]byte, error) {
	f := &modfile.File{
		Module:   modulePath,
		Language: &modfile.Language{Version: LanguageVersion},
		Deps:     make(map[string]*modfile.Dep, len(pinned)),
	}
	for path, version := range pinned {
		f.Deps[path] = &modfile.Dep{Version: version}
	}
	data, err := modfile.Format(f)
	if err != nil {
		return nil, fmt.Errorf("formatting %s: %w", ModuleFileName, err)
	}
	return data, nil
}

// renderPlatformFile emits platform.cue: the core.#Platform embedding, one
// unqualified catalog import per entry under a positional alias (cat<N>, so
// two catalogs sharing a last path element cannot collide), and one
// #registry entry per subscription carrying enable, the stamped expected
// version and the imported catalog. CUE names an unqualified import after
// the path's last element, the convention both first-party catalogs follow;
// a catalog whose root package deviates fails the build naming the import.
func renderPlatformFile(name, typ string, entries []Entry) []byte {
	var b bytes.Buffer
	b.WriteString("// Generated by opm/helper/platformmodule from catalog coordinates. Never edit, never publish.\n")
	b.WriteString("package platform\n\n")
	b.WriteString("import (\n")
	fmt.Fprintf(&b, "\tcore %s\n", quote(CorePath))
	for i, e := range entries {
		fmt.Fprintf(&b, "\tcat%d %s\n", i, quote(e.Path))
	}
	b.WriteString(")\n\n")
	b.WriteString("core.#Platform\n\n")
	fmt.Fprintf(&b, "metadata: name: %s\n", quote(name))
	fmt.Fprintf(&b, "type: %s\n\n", quote(typ))
	b.WriteString("#registry: {")
	if len(entries) == 0 {
		b.WriteString("}\n")
		return b.Bytes()
	}
	b.WriteString("\n")
	for i, e := range entries {
		fmt.Fprintf(&b, "\t%s: {\n", quote(e.Path))
		fmt.Fprintf(&b, "\t\tenable:   %t\n", e.Enable)
		fmt.Fprintf(&b, "\t\tversion:  %s\n", quote(e.Version))
		fmt.Fprintf(&b, "\t\t#catalog: cat%d\n", i)
		b.WriteString("\t}\n")
	}
	b.WriteString("}\n")
	return b.Bytes()
}

// quote renders s as a CUE string literal. Every value quoted here is a
// module path, a SemVer string or a DNS-style name, none of which contains
// a quote or a backslash; the escape is defensive.
func quote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// canonicalVersion adds the "v" prefix cue.mod requires to a bare SemVer
// string; an already-prefixed or empty version is returned unchanged.
func canonicalVersion(v string) string {
	if v == "" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

func sortDeps(deps []Dep) {
	sort.Slice(deps, func(i, j int) bool { return deps[i].Path < deps[j].Path })
}
