package kernel

import (
	"fmt"
	"path/filepath"

	"cuelang.org/go/cue"

	loaderfile "github.com/open-platform-model/library/pkg/helper/loader/file"
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
// Name defaults to the basename. The function delegates to the existing
// [loaderfile.LoadValuesFile] helper.
func (k *Kernel) LoadSourceFromFile(path string) (Source, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Source{}, fmt.Errorf("resolving source path: %w", err)
	}
	v, err := loaderfile.LoadValuesFile(k.cueCtx, absPath)
	if err != nil {
		return Source{}, err
	}
	return Source{
		Value:  v,
		Name:   filepath.Base(absPath),
		Origin: absPath,
	}, nil
}
