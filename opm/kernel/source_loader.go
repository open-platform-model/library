package kernel

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
)

// LoadSourceFromBytes compiles b into a [Source] with [cue.Filename](origin)
// baked into the resulting [cue.Value], so that any subsequent validation
// error positions carry origin via [token.Pos.Filename].
//
// Returns an error if compilation fails (the returned [Source] is the zero
// value in that case).
func (k *Kernel) LoadSourceFromBytes(origin, name string, b []byte) (Source, error) {
	v := k.cueCtx.CompileBytes(b, cue.Filename(origin))
	if err := v.Err(); err != nil {
		return Source{}, fmt.Errorf("compiling source %q: %w", origin, err)
	}
	return Source{Value: v, Name: name, Origin: origin}, nil
}

// LoadSourceFromString is the [string]-input mirror of
// [Kernel.LoadSourceFromBytes].
func (k *Kernel) LoadSourceFromString(origin, name, s string) (Source, error) {
	v := k.cueCtx.CompileString(s, cue.Filename(origin))
	if err := v.Err(); err != nil {
		return Source{}, fmt.Errorf("compiling source %q: %w", origin, err)
	}
	return Source{Value: v, Name: name, Origin: origin}, nil
}

// LoadSourceFromFile reads a values file from disk, compiles it via
// [cuelang.org/go/cue/load.Instances] (which populates [cue.Filename]
// automatically with the file's absolute path), and returns a [Source]
// whose Origin matches the absolute path so per-position diagnostics
// remain consistent with the file the user wrote.
//
// Auto-unwrap: if the loaded value has a top-level `values:` field that
// exists and reports no error, that field is returned as the Source.Value.
// Otherwise the whole evaluated file value is returned.
//
// Name defaults to the basename.
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

	cfg := &load.Config{
		Dir: filepath.Dir(absPath),
	}
	instances := load.Instances([]string{filepath.Base(absPath)}, cfg)
	if len(instances) == 0 {
		return Source{}, fmt.Errorf("no CUE instances found for %s", absPath)
	}
	if instances[0].Err != nil {
		return Source{}, fmt.Errorf("loading values file: %w", instances[0].Err)
	}

	v := k.cueCtx.BuildInstance(instances[0])
	if err := v.Err(); err != nil {
		return Source{}, fmt.Errorf("building values file: %w", err)
	}

	// Why: OPM values files conventionally wrap their payload in a top-level
	// `values:` field; the Source.Value carried into validation must be the
	// inner object so it unifies against the module's #config directly.
	if valuesField := v.LookupPath(cue.ParsePath("values")); valuesField.Exists() && valuesField.Err() == nil {
		v = valuesField
	}

	return Source{
		Value:  v,
		Name:   filepath.Base(absPath),
		Origin: absPath,
	}, nil
}
