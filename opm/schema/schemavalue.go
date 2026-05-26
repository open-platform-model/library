package schema

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	core "github.com/open-platform-model/library/apis/core"
)

// EmbeddedSchema returns the embedded OPM CUE schema filesystem. Used for
// offline / deterministic schema validation without a CUE_REGISTRY round-trip.
func EmbeddedSchema() fs.FS { return core.Schema }

// Package-level cache for SchemaValue. The cache is implicitly keyed on the
// first *cue.Context passed in; the documented invariant ("one Kernel per
// process, one *cue.Context per process") makes that safe in practice. A
// second context after the first call has implementation-defined behavior.
var (
	schemaOnce sync.Once
	schemaVal  cue.Value
	schemaErr  error
)

// SchemaValue loads the embedded OPM schema package and returns it as a fully
// built cue.Value. The returned value's definitions (#ModuleRelease, #Module,
// etc.) are reachable via LookupPath; callers unify against them to produce
// schema-derived artifact values without a CUE_REGISTRY round-trip.
//
// Caching: the first call builds the package and caches the result. Repeated
// calls return the same cue.Value (or the same error) without re-running
// cue/load.Instances. The cache is safe for concurrent first-call invocation
// via sync.Once. Errors are cached too — the load is never retried.
//
// Fatal-on-error guidance: the embedded filesystem is fixed at build time, so
// a load failure indicates a programming error (broken embed pattern, missing
// manifest, malformed package). Callers MAY treat the returned error as
// fatal — there is no recovery short of rebuilding the binary.
func SchemaValue(ctx *cue.Context) (cue.Value, error) {
	schemaOnce.Do(func() {
		schemaVal, schemaErr = loadSchemaValue(ctx, core.Schema, "apis/core", ".")
	})
	return schemaVal, schemaErr
}

// loadSchemaValue builds a CUE instance from an embed.FS-shaped read-only
// filesystem. fsys MUST contain a cue.mod/module.cue at its root and a CUE
// package at <pkgDir> (use "." when the package lives at the FS root).
// virtualRoot is the absolute path used to key the load.Config.Overlay map;
// CUE walks parents from <virtualRoot>/<pkgDir> to locate cue.mod, so the
// overlay paths MUST agree.
//
// The helper is package-private so an internal test may drive it with a
// deliberately broken filesystem.
func loadSchemaValue(ctx *cue.Context, fsys fs.FS, virtualRoot, pkgDir string) (cue.Value, error) {
	if ctx == nil {
		return cue.Value{}, fmt.Errorf("schema SchemaValue: nil *cue.Context")
	}
	if fsys == nil {
		return cue.Value{}, fmt.Errorf("schema SchemaValue: nil embed filesystem")
	}

	// Build an overlay mapping virtualRoot-joined paths to file contents.
	// Using a synthetic absolute root keeps the load entirely off-disk — no
	// CUE_REGISTRY, no filesystem read.
	root := filepath.Join(string(filepath.Separator), "embed", virtualRoot)
	overlay := map[string]load.Source{}
	walkErr := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return readErr
		}
		overlay[filepath.Join(root, filepath.FromSlash(p))] = load.FromBytes(data)
		return nil
	})
	if walkErr != nil {
		return cue.Value{}, fmt.Errorf("schema SchemaValue: walking embed: %w", walkErr)
	}

	pkgDirAbs := filepath.Join(root, filepath.FromSlash(pkgDir))
	cfg := &load.Config{
		Dir:     pkgDirAbs,
		Overlay: overlay,
	}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return cue.Value{}, fmt.Errorf("schema SchemaValue: load.Instances returned no instances")
	}
	if instances[0].Err != nil {
		return cue.Value{}, fmt.Errorf("schema SchemaValue: loading schema: %w", instances[0].Err)
	}

	// BuildInstance returns a usable cue.Value even when deep-path
	// evaluation errors exist on unset required fields (e.g. the schema's
	// own self-references at #Module.metadata.modulePath). We deliberately
	// do NOT call val.Err() here: the well-formedness gate is load-level
	// (instances[0].Err above), and definition existence is what consumers
	// actually rely on. Treat val.Err() at root as informational; downstream
	// callers that unify against an unset path see the real diagnostic.
	val := ctx.BuildInstance(instances[0])
	return val, nil
}
