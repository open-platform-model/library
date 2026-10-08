package kernel_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/open-platform-model/library/opm/errors"
	"github.com/open-platform-model/library/opm/internal/registrytest"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

// The colliding-majors fixtures (testdata/render): maj 0.1.0 and maj 1.4.0
// list the same container, expose and backup keys; maj 1.4.0's bridge
// transformer requires maj 0.1.0's container.
const (
	renderMajPath = renderPrefix + "/maj"
	renderMajV0   = renderMajPath + "@v0"
	renderMajV1   = renderMajPath + "@v1"
)

// majCollisionRows is the collision set every colliding-majors platform
// carries, key-sorted.
func majCollisionRows() []oerrors.ContractCollision {
	both := []string{renderMajV0, renderMajV1}
	return []oerrors.ContractCollision{
		{Key: renderMajPath + "/resources/container@v1", Catalogs: both},
		{Key: renderMajPath + "/traits/backup@v1", Catalogs: both},
		{Key: renderMajPath + "/traits/expose@v1", Catalogs: both},
	}
}

// logCompiled records every compiled object's kind, name and built-by
// annotation, so a render that should have refused shows what it produced.
func logCompiled(t *testing.T, res *kernel.RenderResult) {
	t.Helper()
	if res == nil {
		return
	}
	for _, c := range res.Compiled {
		kind, _ := c.Value.LookupPath(cue.ParsePath("kind")).String()
		name, _ := c.Value.LookupPath(cue.ParsePath("metadata.name")).String()
		by, _ := c.Value.LookupPath(cue.ParsePath(`metadata.annotations."built-by"`)).String()
		t.Logf("rendered %s %s built-by=%q (transformer %s)", kind, name, by, c.Transformer)
	}
}

// joinedCauses returns the gate's joined causes in order.
func joinedCauses(t *testing.T, re *kernel.RenderError) []error {
	t.Helper()
	j, ok := re.Err.(interface{ Unwrap() []error })
	require.True(t, ok, "the gate joins its causes: %T", re.Err)
	return j.Unwrap()
}

// renderCollide renders an instance fixture against a colliding platform and
// asserts the refusal every such render shares: a RenderError whose first
// joined cause is the typed collision cause carrying the three rows, the
// same rows on the diagnostics, no result, routable false, and the render
// module's own gate agreeing.
func renderCollide(t *testing.T, platformDir, instanceDir string, skip bool) *kernel.RenderError {
	t.Helper()
	k := newRenderKernel(t)
	plat := acquireRenderPlatform(t, k, platformDir)
	inst := acquireRenderInstance(t, k, instanceDir)

	built, res, err := k.RenderForTest(context.Background(), kernel.RenderInput{
		Instance: inst, Platform: plat, RuntimeName: "rt", SkipUnprovided: skip,
	})
	logCompiled(t, res)
	require.Error(t, err, "a colliding platform is refused")
	var re *kernel.RenderError
	require.True(t, errors.As(err, &re), "the refusal is a RenderError, got: %v", err)
	assert.Nil(t, res, "no compiled object is returned")
	assertGateAgrees(t, built, true)

	causes := joinedCauses(t, re)
	first, ok := causes[0].(*oerrors.ContractCollisionsError)
	require.True(t, ok, "the collision cause is the first joined cause, got %T", causes[0])
	assert.Equal(t, majCollisionRows(), first.Contracts, "the cause carries the three rows, key-sorted")
	assert.Equal(t, majCollisionRows(), re.Diagnostics.Collisions, "the diagnostics carry the same rows")
	assert.False(t, re.Diagnostics.Routable)
	var nr *oerrors.NotRoutableError
	assert.False(t, errors.As(err, &nr), "the collision rows explain routable false")
	assert.Contains(t, err.Error(), "3 colliding contract(s)")
	return re
}

// single-build-render spec, "A colliding bridge platform is refused instead
// of rendering twice": without the refusal the maj 0.1.0 component renders
// twice, as two Deployments named probe-maj0-web (built by 0.1.0 and by
// bridge-1.4.0).
func TestRender_CollidingBridgePlatformRefused(t *testing.T) {
	re := renderCollide(t, "platform_collide", "instance_maj0", false)
	assert.Len(t, joinedCauses(t, re), 1, "a collision alone: no other cause")
}

// single-build-render spec, "A collision refuses under the skip switch".
func TestRender_CollisionRefusesUnderSkipUnprovided(t *testing.T) {
	re := renderCollide(t, "platform_collide", "instance_maj0", true)
	assert.Len(t, joinedCauses(t, re), 1)
}

// single-build-render spec, "A collision and an over-subscription are both
// named, collision first": both bprov majors require the colliding backup
// trait.
func TestRender_CollisionThenOverSubscription(t *testing.T) {
	re := renderCollide(t, "platform_collide_oversubscribed", "instance_maj0", false)

	causes := joinedCauses(t, re)
	require.Len(t, causes, 2, "collision, then over-subscription")
	over, ok := causes[1].(*oerrors.OverSubscribedContractsError)
	require.True(t, ok, "the second cause is the over-subscription, got %T", causes[1])
	assert.Equal(t, []oerrors.OverSubscribedContract{{
		Key:      renderMajPath + "/traits/backup@v1",
		Catalogs: []string{renderPrefix + "/bprov@v0", renderPrefix + "/bprov@v1"},
	}}, over.Contracts)
}

// single-build-render spec, "A colliding contract row names its entries":
// the load-bearing backup trait on a platform where it collides and nothing
// requires it. The row carries no defining catalog, names the colliding
// entries and never says that no enabled catalog defines the contract.
func TestRender_CollidingUnresolvedRowNamesItsEntries(t *testing.T) {
	re := renderCollide(t, "platform_collide", "instance_bk0", false)

	causes := joinedCauses(t, re)
	require.Len(t, causes, 2, "collision, then the unresolved backup demand")
	unresolved, ok := causes[1].(*oerrors.UnresolvedDemandsError)
	require.True(t, ok, "got %T", causes[1])
	require.Len(t, unresolved.Demands, 1)
	row := unresolved.Demands[0]
	assert.Equal(t, "web", row.Component)
	assert.Equal(t, renderMajPath+"/traits/backup@v1", row.FQN)
	assert.Empty(t, row.DefinedBy, "a colliding key has no single defining catalog")
	assert.Equal(t, []string{renderMajV0, renderMajV1}, row.Colliding)
	assert.Equal(t, re.Diagnostics.Unresolved, unresolved.Demands)
	assert.Contains(t, unresolved.Error(), "defined by more than one enabled registry entry")
	assert.NotContains(t, unresolved.Error(), "no enabled catalog defines")
}

// single-build-render spec, "A platform pinning a core without the report
// renders unchanged": the healthy platform re-pinned to core 2.0.0-alpha.12,
// whose #contracts carries no collision report. The guarded glue row reads
// empty and the render succeeds as before.
func TestRender_OlderCoreWithoutCollisionReportRendersUnchanged(t *testing.T) {
	k := newRenderKernel(t)
	dir := t.TempDir()
	copyTree(t, renderFixtureDir(t, "platform"), dir)
	modFile := filepath.Join(dir, "cue.mod", "module.cue")
	data, err := os.ReadFile(modFile)
	require.NoError(t, err)
	pin := `v: "` + registrytest.DefaultCoreVersion + `"`
	require.Contains(t, string(data), pin)
	require.NoError(t, os.WriteFile(modFile, []byte(strings.Replace(string(data), pin, `v: "v2.0.0-alpha.12"`, 1)), 0o644))
	plat, err := k.AcquirePlatformFromDir(context.Background(), dir)
	require.NoError(t, err)
	require.False(t, plat.Package.LookupPath(schema.ContractsCollisions).Exists(),
		"precondition: core 2.0.0-alpha.12 carries no collision report")

	inst := acquireRenderInstance(t, k, "instance")
	built, res, err := k.RenderForTest(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	require.NoError(t, err)
	assertGateAgrees(t, built, false)
	assert.Empty(t, res.Diagnostics.Collisions, "no collision row")
	assert.True(t, res.Diagnostics.Routable)
	assert.Equal(t, []string{
		"config :: configmap-transformer@0.1.0",
		"web :: deployment-transformer@0.1.0",
		"web :: service-transformer@0.1.0",
	}, renderPairSet(res.Diagnostics.Pairs), "the same pairs as on the default core")
}

// A hand-built platform (NewPlatformFromValue plus a Source, no
// AcquirePlatformFromDir) over platform_collide re-pinned to core
// 2.0.0-alpha.12, a core without the collision report: the one value outside
// the absent-means-empty proof, since no loader checked it. Measured: the
// loader refuses the directory (the definedBy fold conflicts), the hand-built
// value still carries metadata and #contracts.providedBy so it passes
// construction (which records the contracts refusal and does not return it)
// and the render's core floor, and the render build then fails
// on the same definedBy conflict when the diagnostics are decoded. It is a
// plain error, never a render: the collision is not detected as a collision,
// but it is not rendered either.
func TestRender_HandBuiltOlderCoreCollidingPlatformNeverRenders(t *testing.T) {
	k := newRenderKernel(t)
	dir := t.TempDir()
	copyTree(t, renderFixtureDir(t, "platform_collide"), dir)
	modFile := filepath.Join(dir, "cue.mod", "module.cue")
	data, err := os.ReadFile(modFile)
	require.NoError(t, err)
	pin := `v: "` + registrytest.DefaultCoreVersion + `"`
	require.Contains(t, string(data), pin)
	require.NoError(t, os.WriteFile(modFile, []byte(strings.Replace(string(data), pin, `v: "v2.0.0-alpha.12"`, 1)), 0o644))

	_, acqErr := k.AcquirePlatformFromDir(context.Background(), dir)
	require.Error(t, acqErr, "the loader refuses the old-core colliding platform")
	assert.Contains(t, acqErr.Error(), "conflicting values")

	// The same value built by hand, bypassing the loader's error check.
	pkg := buildPlatformValue(t, dir)
	plat, err := platform.NewPlatformFromValue(pkg)
	require.NoError(t, err, "construction records the contracts refusal and does not return it")
	require.True(t, plat.Package.LookupPath(schema.ContractsProvidedBy).Exists(), "the value carries providedBy")
	require.NoError(t, plat.CoreFloor(), "the value passes the core floor")
	_, contractsErr := plat.Contracts()
	require.Error(t, contractsErr, "the recorded inventory refusal is returned later")
	assert.Contains(t, contractsErr.Error(), "conflicting values")
	require.False(t, plat.Package.LookupPath(schema.ContractsCollisions).Exists(), "and carries no collision report")
	plat.Source = &module.Source{Root: dir}

	inst := acquireRenderInstance(t, k, "instance_maj0")
	_, res, rerr := k.RenderForTest(context.Background(), kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "rt"})
	logCompiled(t, res)
	require.Error(t, rerr, "a hand-built old-core colliding platform never renders")
	assert.Nil(t, res)
	assert.Contains(t, rerr.Error(), "conflicting values", "the build fails on the definedBy conflict")
}

// buildPlatformValue builds the platform package at dir the way a caller
// holding its own value would, with no loader gate: the value is returned
// whether or not it evaluated, under the test's registry mapping.
func buildPlatformValue(t *testing.T, dir string) cue.Value {
	t.Helper()
	insts := load.Instances([]string{"."}, &load.Config{Dir: dir, Env: os.Environ()})
	require.Len(t, insts, 1)
	require.NoError(t, insts[0].Err, "loading the platform package")
	return cuecontext.New().BuildInstance(insts[0])
}
