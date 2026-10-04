package kernel

import (
	"context"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/internal/loader"
	"github.com/open-platform-model/library/opm/module"
)

// RenderForTest is Render with the render module's built value exposed, so a
// test can assert that the module's own `gate` field agrees with the kernel's
// decoded refusal. The value is zero on a refusal that never reached the
// build. Test-only seam; not part of the kernel's public surface.
func (k *Kernel) RenderForTest(ctx context.Context, in RenderInput) (cue.Value, *RenderResult, error) {
	return k.render(ctx, in)
}

// DirSourceForTest is the directory description every overlay-mode directory
// verb builds from, exposed so a test can show the package is built from the
// bytes it read and not from a later read of the directory. Test-only seam;
// not part of the kernel's public surface.
func DirSourceForTest(dirPath string) (*module.Source, error) {
	return dirSource("DirSourceForTest", dirPath, loader.ModuleSpec, true)
}
