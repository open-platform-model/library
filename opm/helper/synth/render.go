package synth

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue/literal"

	"github.com/open-platform-model/library/opm/module"
)

// corePath is the module path of the OPM core schema the synthesized instance
// imports. The import major is derived from the core version the schema cache
// resolved (the v2 line, schema.DefaultSchemaModule); the concrete core
// version the import resolves to comes from the module's own
// cue.mod/module.cue (design D4), not from a fabricated pin.
const corePath = "opmodel.dev/core"

// moduleImportPath returns the CUE registry module path — major suffix
// included — the synthesized package imports the module by: the module's
// metadata.modulePath verbatim (enhancement 0010 D1: fqn = modulePath = the
// import path; nothing is recombined). The schema requires the major-suffixed
// form, so a value that somehow lacks it fails the build with CUE's own
// import error naming the path.
func moduleImportPath(m *module.Module) string {
	return m.Metadata.ModulePath
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
	modImport := moduleImportPath(in.Module)

	var b strings.Builder
	b.WriteString("package instance\n\n")
	b.WriteString("import (\n")
	fmt.Fprintf(&b, "\tcore %s\n", literal.String.Quote(corePath+"@"+major(coreVersion)))
	fmt.Fprintf(&b, "\topmModule %s\n", literal.String.Quote(modImport))
	b.WriteString(")\n\n")
	b.WriteString("core.#ModuleInstance\n\n")
	b.WriteString("metadata: {\n")
	fmt.Fprintf(&b, "\tname:      %s\n", literal.String.Quote(in.Name))
	fmt.Fprintf(&b, "\tnamespace: %s\n", literal.String.Quote(in.Namespace))
	writeStringMap(&b, "\t", "labels", in.Labels)
	writeStringMap(&b, "\t", "annotations", in.Annotations)
	b.WriteString("}\n\n")
	b.WriteString("#module: opmModule\n")
	return b.String()
}

// writeStringMap writes a "<field>: { ... }" block when m is non-empty. Keys
// and values are emitted as CUE string literals (literal.String.Quote, the
// quoting CUE itself parses, so a non-ASCII or control rune round-trips);
// identity strings (name, namespace) are formatted the same way, keeping
// every caller string a literal rather than interpolated CUE source.
func writeStringMap(sb *strings.Builder, indent, field string, m map[string]string) {
	if len(m) == 0 {
		return
	}
	fmt.Fprintf(sb, "%s%s: {\n", indent, field)
	for k, v := range m {
		fmt.Fprintf(sb, "%s\t%s: %s\n", indent, literal.String.Quote(k), literal.String.Quote(v))
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
