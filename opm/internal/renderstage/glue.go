package renderstage

import (
	"bytes"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/literal"
)

// renderTemplate is the embedded glue: static matching, execution and
// diagnostics text plus four generated slots (the two imports, the runtime
// name literal and the skip-unprovided switch literal). Nothing else is
// filled into the build.
//
//go:embed render.cue.tmpl
var renderTemplate string

// RenderFileName is the generated glue file's name inside the render module.
const RenderFileName = "render.cue"

// Delimiters are << >> because CUE's own `{{...}}` (a struct inside a list
// comprehension) appears throughout the glue.
var glueTmpl = template.Must(template.New(RenderFileName).Delims("<<", ">>").Parse(renderTemplate))

// GlueInputs are the generated slots of the render.cue template.
type GlueInputs struct {
	// InstancePath is the instance package's import path: the instance
	// module's qualified path plus the package directory, with an explicit
	// package qualifier when the package name differs from the directory.
	InstancePath string

	// PlatformPath is the platform package's import path, formed the same way.
	PlatformPath string

	// RuntimeName is the executing runtime's identity, entering the build as
	// a CUE string literal.
	RuntimeName string

	// SkipUnprovided is the caller's switch to skip unprovided
	// provider-fulfilled demands, entering the build as the bool literal
	// the matching verdicts and the module's own gate read.
	SkipUnprovided bool
}

// RenderGlue renders the glue file for the given inputs. Caller-supplied
// strings enter as quoted CUE literals, never by raw interpolation.
func RenderGlue(in GlueInputs) ([]byte, error) {
	if in.InstancePath == "" || in.PlatformPath == "" {
		return nil, fmt.Errorf("render glue needs both import paths")
	}
	if in.RuntimeName == "" {
		return nil, fmt.Errorf("render glue needs a runtime name")
	}
	var buf bytes.Buffer
	err := glueTmpl.Execute(&buf, struct {
		InstanceImport string
		PlatformImport string
		RuntimeName    string
		SkipUnprovided string
	}{
		InstanceImport: literal.String.Quote(in.InstancePath),
		PlatformImport: literal.String.Quote(in.PlatformPath),
		RuntimeName:    literal.String.Quote(in.RuntimeName),
		SkipUnprovided: strconv.FormatBool(in.SkipUnprovided),
	})
	if err != nil {
		return nil, fmt.Errorf("rendering glue: %w", err)
	}
	return buf.Bytes(), nil
}

// ImportPath forms the import path of a package inside a module: the
// module's root path, the package directory (slash-separated, "" for the
// root package), the module's major, and the package name as an explicit
// qualifier when it differs from the last path element.
func ImportPath(qualifiedModule, pkgDir, pkgName string) (string, error) {
	root, major, ok := strings.Cut(qualifiedModule, "@")
	if !ok || root == "" || major == "" {
		return "", fmt.Errorf("module path %q carries no major version", qualifiedModule)
	}
	if pkgName == "" {
		return "", fmt.Errorf("package name is required for %q", qualifiedModule)
	}
	path := root
	if dir := strings.Trim(strings.ReplaceAll(pkgDir, "\\", "/"), "/"); dir != "" && dir != "." {
		path = root + "/" + dir
	}
	ip := ast.ImportPath{Path: path, Version: major, Qualifier: pkgName}
	if last := path[strings.LastIndex(path, "/")+1:]; last != pkgName {
		ip.ExplicitQualifier = true
	}
	return ip.String(), nil
}
