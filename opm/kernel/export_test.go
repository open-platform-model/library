package kernel

import (
	"context"

	"cuelang.org/go/cue"
)

// RenderForTest is Render with the render module's built value exposed, so a
// test can assert that the module's own `gate` field agrees with the kernel's
// decoded refusal. The value is zero on a refusal that never reached the
// build. Test-only seam; not part of the kernel's public surface.
func (k *Kernel) RenderForTest(ctx context.Context, in RenderInput) (cue.Value, *RenderResult, error) {
	return k.render(ctx, in)
}
