package kernel

import (
	"context"

	"github.com/open-platform-model/library/opm/materialize"
	"github.com/open-platform-model/library/opm/platform"
)

// Materialize realizes a #Platform's path-keyed catalog subscriptions into a
// sealed [*materialize.MaterializedPlatform], delegating to opm/materialize
// with the kernel's configured registry ([WithRegistry]) and owned
// [*cue.Context].
//
// It performs registry I/O (version enumeration + OCI pulls) and is explicit
// and caller-driven: the kernel holds no materialize cache (Principle I).
// Long-running consumers that want memoization store the result keyed on an
// invalidation signal they own; short-lived ones rely on CUE's on-disk
// module cache.
//
// The phase methods (Match, Compile) accept only the materialized form:
// callers MUST Materialize a *platform.Platform before invoking either
// phase.
func (k *Kernel) Materialize(ctx context.Context, p *platform.Platform) (*materialize.MaterializedPlatform, error) {
	return materialize.Materialize(ctx, k, k.registry, p)
}
