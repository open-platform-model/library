// Package valuesfile renders a values cue.Value as the source of a package
// file declaring the top-level `values` field. It is the one renderer behind
// every place the library writes caller-supplied values into a CUE package
// (instance synthesis in opm/helper/synth, extra values layered onto an
// on-disk instance package by Kernel.AcquireInstanceFromDir), so the two
// paths cannot drift.
//
// It lives under opm/internal/ so it stays out of the library's public
// SemVer surface while remaining importable from opm/kernel and opm/helper.
package valuesfile

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/format"
)

// Render produces the source of a file in package pkg whose single
// declaration is `values: <rendered>`. The value is rendered back to
// canonical CUE via format.Node on the value's syntax, NEVER by
// string-interpolating raw input, so an attacker-influenced value cannot
// inject CUE source. It returns (nil, nil) when values does not exist,
// signalling the caller to omit the file entirely.
func Render(pkg string, values cue.Value) ([]byte, error) {
	if !values.Exists() {
		return nil, nil
	}

	node := values.Syntax(cue.Final(), cue.Concrete(false))
	rendered, err := format.Node(node)
	if err != nil {
		return nil, fmt.Errorf("rendering values to CUE source: %w", err)
	}

	var b strings.Builder
	b.WriteString("package ")
	b.WriteString(pkg)
	b.WriteString("\n\n")
	b.WriteString("values: ")
	b.Write(rendered)
	b.WriteString("\n")
	return []byte(b.String()), nil
}
