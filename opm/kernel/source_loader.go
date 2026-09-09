package kernel

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue/parser"

	"github.com/open-platform-model/library/opm/schema"
)

// LoadSourceFromBytes wraps b as a [Source] with Origin origin after checking
// that it parses as CUE. Nothing is evaluated: the kernel compiles the source
// with [cue.Filename](origin) in the context of the operation that uses it,
// so any validation error positions carry origin via [token.Pos.Filename]. A
// caller holding a string passes []byte(s).
//
// Returns an error positioned at origin if b does not parse (the returned
// [Source] is the zero value in that case). A payload that parses but does
// not evaluate, or violates the schema it meets, fails in the operation that
// applies it, positioned at origin.
func (k *Kernel) LoadSourceFromBytes(origin string, b []byte) (Source, error) {
	if err := parseSource(origin, b); err != nil {
		return Source{}, err
	}
	return Source{Origin: origin, Data: b}, nil
}

// LoadSourceFromFile reads a values file from disk and returns a [Source]
// whose Origin is the file's absolute path and whose Data is the file's
// bytes, after checking that they parse as CUE. Nothing is evaluated here:
// the operation that uses the source loads it through
// [cuelang.org/go/cue/load.Instances] at the file's directory (so imports
// resolve as they do for any CUE file), builds it in that operation's own
// context, and unwraps a top-level `values:` field that exists without error
// (OPM values files conventionally wrap their payload in one), so the value
// carried into validation is the inner object.
//
// Returns an error if the file cannot be read, or one positioned at the
// file's path if it does not parse.
func (k *Kernel) LoadSourceFromFile(path string) (Source, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Source{}, fmt.Errorf("resolving source path: %w", err)
	}
	if _, statErr := os.Stat(absPath); statErr != nil {
		if os.IsNotExist(statErr) {
			return Source{}, fmt.Errorf("values file %q not found", path)
		}
		return Source{}, fmt.Errorf("accessing values file %q: %w", path, statErr)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return Source{}, fmt.Errorf("reading values file %q: %w", path, err)
	}
	if err := parseSource(absPath, data); err != nil {
		return Source{}, err
	}
	return Source{Origin: absPath, Data: data}, nil
}

// parseSource checks that data parses as CUE, reporting a syntax error
// positioned at origin. It builds no value.
func parseSource(origin string, data []byte) error {
	if _, err := parser.ParseFile(origin, data); err != nil {
		return fmt.Errorf("parsing source %q: %w", origin, err)
	}
	return nil
}

// compileSource compiles s in ctx so that every position the value carries
// names s.Origin. A file-backed source (an absolute Origin naming an existing
// file) is loaded through cue/load at the file's directory with Data overlaid
// at Origin, so its imports resolve exactly as they would for the file on
// disk; any other source is compiled from Data with [cue.Filename](Origin).
// After either, a top-level `values:` field that exists without error is
// unwrapped: OPM values files conventionally wrap their payload in one, and
// the value carried into validation must be the inner object so it unifies
// against the module's #config directly.
//
// An empty Data is "no values supplied" and compiles to the zero value, which
// every consumer treats as absent. A compile failure is returned as the raw
// CUE error tree, positioned at Origin.
func compileSource(ctx *cue.Context, s Source) (cue.Value, error) {
	if len(s.Data) == 0 {
		return cue.Value{}, nil
	}
	var v cue.Value
	if isFileBacked(s.Origin) {
		cfg := &load.Config{
			Dir:     filepath.Dir(s.Origin),
			Overlay: map[string]load.Source{s.Origin: load.FromBytes(s.Data)},
		}
		instances := load.Instances([]string{filepath.Base(s.Origin)}, cfg)
		if len(instances) == 0 {
			return cue.Value{}, fmt.Errorf("no CUE instances found for %s", s.Origin)
		}
		if instances[0].Err != nil {
			return cue.Value{}, instances[0].Err
		}
		v = ctx.BuildInstance(instances[0])
	} else {
		v = ctx.CompileBytes(s.Data, cue.Filename(s.Origin))
	}
	if err := v.Err(); err != nil {
		return cue.Value{}, err
	}
	if values := v.LookupPath(schema.Values); values.Exists() && values.Err() == nil {
		v = values
	}
	return v, nil
}

// compileSources compiles every source in ctx, in stack order.
func compileSources(ctx *cue.Context, sources []Source) ([]cue.Value, error) {
	values := make([]cue.Value, 0, len(sources))
	for _, s := range sources {
		v, err := compileSource(ctx, s)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

// unifyValues unifies values in stack order and returns the merged value. A
// zero value contributes nothing, so a stack of absent values merges to the
// zero value — the "no values supplied" path.
func unifyValues(values []cue.Value) cue.Value {
	var merged cue.Value
	for _, v := range values {
		merged = merged.Unify(v)
	}
	return merged
}

// isFileBacked reports whether origin is an absolute path naming an existing
// file, which is what marks a source as loadable at its directory.
func isFileBacked(origin string) bool {
	if !filepath.IsAbs(origin) {
		return false
	}
	info, err := os.Stat(origin)
	return err == nil && !info.IsDir()
}
