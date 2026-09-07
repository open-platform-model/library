package module

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Source is the staged source tree an artifact was loaded or synthesized
// from: the module root the tree is keyed under, the package directory inside
// it, and (for in-memory trees) the overlay carrying the files as bytes.
//
// A Source is in one of two modes:
//
//   - Overlay mode (Overlay non-empty): the tree lives in memory, keyed under
//     the deterministic synthetic Root. This is how a module fetched from a
//     registry is staged (Kernel.AcquireModuleFromRegistry) and how a
//     synthesized instance is staged inside its module's tree
//     (Kernel.SynthesizeInstance), and how a module acquired from a
//     directory is staged (Kernel.AcquireModuleFromDir).
//   - On-disk mode (Overlay nil): the tree lives at Root on the real
//     filesystem. This is how an artifact acquired from a directory is
//     described (Kernel.AcquirePlatformFromDir, Kernel.AcquireInstanceFromDir).
//
// It exists so an artifact can be RE-USED as the input of a follow-on build:
// a module acquired from the registry becomes the main module of the synth
// build (so its already-tidied cue.mod/module.cue drives transitive dependency
// resolution), and an instance or platform carrying its tree can be imported
// as a package by a later render build. Carrying the staged source on the
// artifact avoids a second fetch or a second directory load.
//
// Source is carried by Module (registry path only), Instance (synthesis and
// directory acquire) and Platform (directory acquire; platform.Source is an
// alias of this type). It is nil for artifacts constructed from a bare value
// (e.g. a unit-test CompileString).
type Source struct {
	// Root is the absolute module root of the tree: the load.Config.ModuleRoot
	// a consumer builds against. In overlay mode it is the synthetic root every
	// Overlay key sits under; in on-disk mode it is a real directory.
	Root string

	// Pkg is the package directory relative to Root that holds the artifact's
	// CUE package. Empty means the root package (".").
	Pkg string

	// Overlay maps absolute paths under Root to their file contents as bytes,
	// cue.mod/module.cue included. Nil selects on-disk mode: the tree is read
	// from Root on the filesystem.
	//
	// Bytes, not cue/load's opaque source interface: every overlay the library
	// builds starts as bytes, and a consumer that materializes the tree — or
	// hands it to cue/load — should not have to recover them by reflection.
	// The kernel wraps them with load.FromBytes at the one place it calls
	// cue/load with an overlay.
	Overlay map[string][]byte
}

// WriteTo materializes an overlay-mode source under dir: every overlay entry
// is written at its path relative to Root — parent directories created as
// needed — so a build served from dir sees the tree it would see from Root.
// It returns the dir-relative paths it wrote, sorted, so a caller never has to
// iterate the overlay itself.
//
// The whole overlay is validated before anything is written, so a refusal
// leaves the directory untouched: a nil receiver, an on-disk source (Overlay
// nil — Root already IS the directory) and any entry whose path is not under
// Root are refused with a plain error.
//
// This is the library's one overlay writer. The kernel's render stage
// materializes an overlay-mode input through it, and a frontend that needs a
// fetched module's tree on disk (scaffolding from a published template) calls
// it instead of fetching the module a second time and walking it.
func (s *Source) WriteTo(dir string) ([]string, error) {
	if s == nil || s.Root == "" {
		return nil, errors.New("source carries no module root")
	}
	if s.Overlay == nil {
		return nil, errors.New("source carries no overlay to write; an on-disk source already lives at its Root")
	}

	root := filepath.Clean(s.Root)
	keyOf := make(map[string]string, len(s.Overlay))
	rels := make([]string, 0, len(s.Overlay))
	for key := range s.Overlay {
		rel, err := filepath.Rel(root, key)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("overlay entry %s is outside the source root %s", key, root)
		}
		keyOf[rel] = key
		rels = append(rels, rel)
	}
	sort.Strings(rels)

	for _, rel := range rels {
		out := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return nil, fmt.Errorf("creating %s: %w", filepath.Dir(out), err)
		}
		if err := os.WriteFile(out, s.Overlay[keyOf[rel]], 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", out, err)
		}
	}
	return rels, nil
}

// HasSource reports whether the module carries a staged registry source tree
// (non-nil Source with a populated overlay). Consumers that must build inside
// the module's own root — e.g. synth.Instance — gate on this and return a
// deterministic error when it is false, rather than silently fetching.
func (m *Module) HasSource() bool {
	return m != nil && m.Source != nil && m.Source.Root != "" && len(m.Source.Overlay) > 0
}
