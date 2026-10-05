package platformmodule

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteTo places the generated files under dir, creating parent directories
// as needed. A file name that would resolve outside dir is refused before
// anything is written. The helper owns no directory lifecycle: whether dir
// is a fresh generation directory, a staging directory later swapped into
// place, or a cache entry is the frontend's policy.
func (f Files) WriteTo(dir string) error {
	if dir == "" {
		return fmt.Errorf("target directory is required")
	}
	paths := make(map[string]string, len(f))
	for name := range f {
		rel := filepath.Clean(filepath.FromSlash(name))
		// IsLocal refuses an absolute or escaping name; "." is local but names
		// the module directory itself, which is not a file to write.
		if !filepath.IsLocal(rel) || rel == "." {
			return fmt.Errorf("refusing to write %q outside the module directory", name)
		}
		paths[name] = filepath.Join(dir, rel)
	}
	for name, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, f[name], 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}
	return nil
}
