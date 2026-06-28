package synth

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"

	"github.com/open-platform-model/library/opm/module"
)

// corePath is the module path of the OPM core schema the synthesized instance
// imports. The import major is derived from the caller-resolved core version
// (currently @v1); the concrete core version the import resolves to comes from
// the module's own cue.mod/module.cue (design D4), not from a fabricated pin.
const corePath = "opmodel.dev/core"

// moduleImportPath returns the CUE registry module path the synthesized package
// imports the module by. Per the OPM module publishing convention
// (enhancements/0003), a module is published at
// metadata.modulePath + "/" + metadata.nameSnakeCase, and its CUE package name
// equals nameSnakeCase — so a bare import of "<path>@<major>" resolves to it.
// Using nameSnakeCase (not the kebab metadata.name) is what makes the address
// derivable for modules whose name carries hyphens (e.g. "zot-registry-ttl"
// published at ".../zot_registry_ttl").
func moduleImportPath(m *module.Module) string {
	return m.Metadata.ModulePath + "/" + moduleSnakeName(m)
}

// moduleSnakeName returns the module's snake_case name. It prefers the
// schema-derived metadata.nameSnakeCase (core ≥ v0.6.0, the authoritative
// projection) and falls back to computing it for modules published against an
// older core that predates the field. The projection is core's #KebabToSnake —
// a total, deterministic hyphens→underscores transform — so the computed
// fallback matches the schema value exactly.
func moduleSnakeName(m *module.Module) string {
	if v := m.Package.LookupPath(cue.ParsePath("metadata.nameSnakeCase")); v.Exists() {
		if s, err := v.String(); err == nil && s != "" {
			return s
		}
	}
	return strings.ReplaceAll(m.Metadata.Name, "-", "_")
}

// renderInstanceFile produces the instance.cue source for the synthesized
// package. It imports core and the caller's module, embeds #ModuleInstance at
// the package root (matching an authored instance.cue), stamps caller-supplied
// identity metadata as regex-constrained literals, and writes the module by
// IMPORT reference (#module: <import>) rather than inlining a value — so the
// module enters the single build with its full type-embedding chain intact and
// the schema's `unifiedModule = #module & {#config: values}` performs the
// values merge in CUE.
//
// Because this file is overlaid INSIDE the module's own staged main module, the
// `opmModule` import (the module's own path@major) resolves LOCALLY to the
// module's root package, and `core` resolves from the module's own tidied
// cue.mod/module.cue — no fabricated dependency declaration is involved.
//
// Was: renderReleaseFile
func renderInstanceFile(in InstanceInput, coreVersion string) string {
	modImport := moduleImportPath(in.Module) + "@" + major(in.Module.Metadata.Version)

	var b strings.Builder
	b.WriteString("package instance\n\n")
	b.WriteString("import (\n")
	fmt.Fprintf(&b, "\tcore %q\n", corePath+"@"+major(coreVersion))
	fmt.Fprintf(&b, "\topmModule %q\n", modImport)
	b.WriteString(")\n\n")
	b.WriteString("core.#ModuleInstance\n\n")
	b.WriteString("metadata: {\n")
	fmt.Fprintf(&b, "\tname:      %q\n", in.Name)
	fmt.Fprintf(&b, "\tnamespace: %q\n", in.Namespace)
	writeStringMap(&b, "\t", "labels", in.Labels)
	writeStringMap(&b, "\t", "annotations", in.Annotations)
	b.WriteString("}\n\n")
	b.WriteString("#module: opmModule\n")
	return b.String()
}

// renderValuesFile produces the values.cue source for the synthesized package
// by rendering in.Values back to canonical CUE via format.Node on the value's
// syntax — NEVER by string-interpolating raw input, so an attacker-influenced
// value cannot inject CUE source. It returns (nil, nil) when no values were
// supplied (in.Values is the zero value or does not exist), signalling the
// caller to omit the file entirely; the schema's values path then stays open
// and concreteness is enforced downstream by Kernel.ProcessModuleInstance.
func renderValuesFile(in InstanceInput) ([]byte, error) {
	if !in.Values.Exists() {
		return nil, nil
	}

	node := in.Values.Syntax(cue.Final(), cue.Concrete(false))
	rendered, err := format.Node(node)
	if err != nil {
		return nil, fmt.Errorf("rendering values to CUE source: %w", err)
	}

	var b strings.Builder
	b.WriteString("package instance\n\n")
	b.WriteString("values: ")
	b.Write(rendered)
	b.WriteString("\n")
	return []byte(b.String()), nil
}

// writeStringMap writes a "<field>: { ... }" block when m is non-empty. Keys
// and values are emitted as quoted literals; identity strings (name, namespace)
// are formatted the same way, keeping every caller string a literal rather than
// interpolated CUE source.
func writeStringMap(sb *strings.Builder, indent, field string, m map[string]string) {
	if len(m) == 0 {
		return
	}
	fmt.Fprintf(sb, "%s%s: {\n", indent, field)
	for k, v := range m {
		fmt.Fprintf(sb, "%s\t%q: %q\n", indent, k, v)
	}
	fmt.Fprintf(sb, "%s}\n", indent)
}

// major returns the major-version selector (e.g. "v0") for a SemVer string in
// either "0.1.0" or "v0.1.0" form. The synthesized import path and dep key are
// major-qualified, matching CUE module-path conventions.
func major(version string) string {
	v := strings.TrimPrefix(version, "v")
	if i := strings.IndexByte(v, '.'); i >= 0 {
		v = v[:i]
	}
	return "v" + v
}
