package v1alpha2_test

import (
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemaCase exercises a single CUE fixture under apis/core/v1alpha2/testdata/.
//
// Authoring rules:
//   - fixture: filename relative to apis/core/v1alpha2/testdata/.
//   - inputPath (optional, default "input"): CUE path of the value-under-test
//     within the fixture. Override to drive multiple cases per fixture; the
//     paired `expect` field for positive cases must live at "<inputPath>_expect".
//   - expectError != "": negative case. The value at inputPath must fail
//     Validate(cue.Concrete(true)) (or BuildInstance must fail); the regex
//     matches against err.Error().
//   - expectError == "" + assertField == "": positive case asserted via
//     `<inputPath> & <inputPath>_expect` evaluated under cue.Concrete(true).
//   - assertField != "": positive case asserted by decoding the value at
//     `assertField` into a target shaped like assertValue and comparing equal.
//
// See apis/core/v1alpha2/testdata/README.md for the fixture authoring contract.
type schemaCase struct {
	name        string
	fixture     string
	inputPath   string
	assertField string
	assertValue any
	expectError string
}

var schemaCases = []schemaCase{
	// ── Seed cases (add-cue-schema-test-harness) ─────────────────────────
	{
		name:    "platform_matchers_projects_single_candidate",
		fixture: "platform_matchers_fixture.cue",
	},
	{
		name:        "fqn_collision_across_modules_bottoms",
		fixture:     "fqn_collision_fixture.cue",
		expectError: `conflicting values`,
	},

	// ── Tier 1 (add-cue-schema-test-coverage) ────────────────────────────
	{
		name:    "distinct_predicates_share_resource_fqn_as_separate_candidates",
		fixture: "predicate_distinct_labels_fixture.cue",
	},
	{
		name:    "disabled_module_suppresses_projections",
		fixture: "enabled_false_suppresses_fixture.cue",
	},
	{
		name:    "transformer_context_merges_three_scopes",
		fixture: "transformer_context_labels_fixture.cue",
	},
	{
		name:        "module_fqn_matches_format_string",
		fixture:     "module_uuid_fixture.cue",
		assertField: "input.metadata.fqn",
		assertValue: "example.com/demo/demo:0.1.0",
	},
	{
		// Pinned UUIDv5(OPMNamespace, "example.com/demo/demo:0.1.0").
		// To regenerate (only if OPMNamespace itself changes):
		//   cue eval -t test ./testdata/module_uuid_fixture.cue \
		//     --expression input.metadata.uuid
		name:        "module_uuid_matches_pinned_v5_hash",
		fixture:     "module_uuid_fixture.cue",
		assertField: "input.metadata.uuid",
		assertValue: "174b36e1-e5ea-5ba4-8de2-8b30c882e669",
	},

	// ── Tier 2 (add-cue-schema-test-coverage) ────────────────────────────
	{
		name:    "trait_matchers_projects_single_candidate",
		fixture: "trait_matchers_fixture.cue",
	},
	{
		name:        "type_regex_rejects_uppercase_name",
		fixture:     "type_regex_fixture.cue",
		inputPath:   "bad_name_uppercase",
		expectError: `invalid value`,
	},
	{
		name:        "type_regex_rejects_leading_hyphen_name",
		fixture:     "type_regex_fixture.cue",
		inputPath:   "bad_name_leading_hyphen",
		expectError: `invalid value`,
	},
	{
		name:        "type_regex_rejects_fqn_without_version",
		fixture:     "type_regex_fixture.cue",
		inputPath:   "bad_fqn_no_version",
		expectError: `invalid value`,
	},
	{
		name:        "type_regex_rejects_module_fqn_without_semver",
		fixture:     "type_regex_fixture.cue",
		inputPath:   "bad_module_fqn_no_semver",
		expectError: `invalid value`,
	},
	{
		name:        "type_regex_rejects_version_without_v_prefix",
		fixture:     "type_regex_fixture.cue",
		inputPath:   "bad_version_no_v_prefix",
		expectError: `invalid value`,
	},
}

// schemaModuleRoot resolves the on-disk path to the apis/core CUE module
// (the directory holding cue.mod/module.cue). Mirrors embed_test.go:22-28 so
// the harness works regardless of the directory `go test` is invoked from.
func schemaModuleRoot(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(here), "..", "..", ".."))
	return filepath.Join(repoRoot, "apis", "core")
}

func TestSchemaFixtures(t *testing.T) {
	moduleRoot := schemaModuleRoot(t)

	for _, tc := range schemaCases {
		t.Run(tc.name, func(t *testing.T) {
			fixtureRel := filepath.Join("v1alpha2", "testdata", tc.fixture)
			inputPath := tc.inputPath
			if inputPath == "" {
				inputPath = "input"
			}
			expectPath := inputPath + "_expect"
			if inputPath == "input" {
				expectPath = "expect"
			}

			insts := load.Instances(
				[]string{"./" + filepath.ToSlash(fixtureRel)},
				&load.Config{
					Dir:  moduleRoot,
					Tags: []string{"test"},
				},
			)
			require.Len(t, insts, 1, "load.Instances must return exactly one instance for %q", tc.fixture)
			require.NoErrorf(t, insts[0].Err, "load error for fixture %q (path: %s)", tc.fixture, fixtureRel)

			ctx := cuecontext.New()
			built := ctx.BuildInstance(insts[0])

			// Negative cases: failures may surface at BuildInstance time (CUE
			// eagerly evaluates closed-struct conflicts and `0 & N` gates
			// during build) or at Validate(Concrete) time. Treat both as the
			// expected error surface and match against the regex.
			if tc.expectError != "" {
				re := regexp.MustCompile(tc.expectError)
				if buildErr := built.Err(); buildErr != nil {
					assert.Regexpf(t, re, buildErr.Error(),
						"fixture %q build error did not match expectError regex\nfull error: %s",
						tc.fixture, buildErr.Error())
					return
				}
				input := built.LookupPath(cue.ParsePath(inputPath))
				require.Truef(t, input.Exists(), "fixture %q must declare top-level %q field", tc.fixture, inputPath)
				err := input.Validate(cue.Concrete(true))
				require.Errorf(t, err, "fixture %q expected validation failure at %q but Validate succeeded", tc.fixture, inputPath)
				assert.Regexpf(t, re, err.Error(),
					"fixture %q error at %q did not match expectError regex\nfull error: %s", tc.fixture, inputPath, err.Error())
				return
			}

			require.NoErrorf(t, built.Err(), "build error for fixture %q", tc.fixture)
			input := built.LookupPath(cue.ParsePath(inputPath))
			require.Truef(t, input.Exists(), "fixture %q must declare top-level %q field", tc.fixture, inputPath)

			if tc.assertField != "" {
				field := built.LookupPath(cue.ParsePath(tc.assertField))
				require.Truef(t, field.Exists(), "fixture %q has no value at assertField %q", tc.fixture, tc.assertField)
				require.NoErrorf(t, field.Validate(cue.Concrete(true)),
					"fixture %q value at %q is not concrete", tc.fixture, tc.assertField)
				gotPtr := newDecodeTarget(tc.assertValue)
				require.NoErrorf(t, field.Decode(gotPtr),
					"fixture %q decode of %q into %T failed", tc.fixture, tc.assertField, tc.assertValue)
				assert.Equalf(t, tc.assertValue, derefDecodeTarget(gotPtr),
					"fixture %q value at %q did not match expected", tc.fixture, tc.assertField)
				return
			}

			// Default positive contract: `<inputPath> & <expectPath>` must be concrete.
			expect := built.LookupPath(cue.ParsePath(expectPath))
			require.Truef(t, expect.Exists(),
				"fixture %q (positive case without assertField) must declare top-level %q field", tc.fixture, expectPath)
			unified := input.Unify(expect)
			require.NoErrorf(t, unified.Err(), "fixture %q: %s & %s produced an error value", tc.fixture, inputPath, expectPath)
			require.NoErrorf(t, unified.Validate(cue.Concrete(true)),
				"fixture %q: %s & %s is not concrete (likely a divergence between input's computed projection and expect)", tc.fixture, inputPath, expectPath)
		})
	}
}

// newDecodeTarget allocates a *T pointer suitable for cue.Value.Decode given a
// sample value of type T. Decode requires a settable target; reflect lets us
// build one without forcing schemaCase authors to spell out pointer types.
func newDecodeTarget(sample any) any {
	return reflect.New(reflect.TypeOf(sample)).Interface()
}

func derefDecodeTarget(ptr any) any {
	return reflect.ValueOf(ptr).Elem().Interface()
}
