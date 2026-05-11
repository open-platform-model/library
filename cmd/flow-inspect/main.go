// Command flow-inspect dumps each stage of the plan/match/compile pipeline
// as pretty-printed CUE so a reader can inspect what the kernel sees.
//
// Default stages (run all):
//
//  1. module    — the loaded web_app #Module
//  2. platform  — opm_platform plus its computed views
//  3. release   — the constructed #ModuleRelease passed to the matcher
//  4. plan      — the Go MatchPlan and per-pair Compiled outputs
//
// Usage:
//
//	go run ./cmd/flow-inspect                  # all stages
//	go run ./cmd/flow-inspect -stages module   # one stage
//	go run ./cmd/flow-inspect -stages plan,release
//
// Imports are resolved through the local OPM registry; the command exits
// non-zero with a clear message when localhost:5000 is unreachable.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/format"

	"github.com/open-platform-model/library/pkg/api"
	_ "github.com/open-platform-model/library/pkg/api/v1alpha2"
	loaderfile "github.com/open-platform-model/library/pkg/helper/loader/file"
	"github.com/open-platform-model/library/pkg/kernel"
)

const localRegistry = "testing.opmodel.dev=localhost:5000+insecure,opmodel.dev=localhost:5000+insecure,registry.cue.works"

type stage string

const (
	stageModule   stage = "module"
	stagePlatform stage = "platform"
	stageRelease  stage = "release"
	stagePlan     stage = "plan"
)

var allStages = []stage{stageModule, stagePlatform, stageRelease, stagePlan}

func main() {
	stagesFlag := flag.String("stages", "", "comma-separated list of stages to dump (module,platform,release,plan); empty = all")
	libRoot := flag.String("library", "", "absolute path to the library/ directory; defaults to current working directory")
	flag.Parse()

	root, err := resolveLibraryRoot(*libRoot)
	if err != nil {
		fatalf("resolving library root: %v", err)
	}

	want, err := parseStages(*stagesFlag)
	if err != nil {
		fatalf("%v", err)
	}

	if err := checkRegistry(); err != nil {
		fatalf("local CUE registry not reachable: %v\n\nStart it via `task -d ../opm registry:start` (or set OPM_FLOW_INSPECT_FORCE=1 to bypass this probe).", err)
	}

	if err := os.Setenv("CUE_REGISTRY", localRegistry); err != nil {
		fatalf("setting CUE_REGISTRY: %v", err)
	}

	if err := run(root, want); err != nil {
		fatalf("%v", err)
	}
}

func run(libraryRoot string, want map[stage]bool) error {
	k := kernel.New()
	ctx := context.Background()

	// ── Load module ─────────────────────────────────────────────────
	moduleDir := filepath.Join(libraryRoot, "testdata", "modules", "web_app")
	modVal, modVer, err := k.LoadModulePackage(ctx, moduleDir)
	if err != nil {
		return fmt.Errorf("loading module from %s: %w", moduleDir, err)
	}
	mod, err := k.NewModuleFromValue(modVal)
	if err != nil {
		return fmt.Errorf("constructing *module.Module: %w", err)
	}

	if want[stageModule] {
		header("Stage 1: Loaded Module — web_app")
		mustPrintCUE(modVal)
	}

	// ── Load platform ───────────────────────────────────────────────
	platformDir := filepath.Join(libraryRoot, "modules", "opm_platform")
	platVal, _, err := k.LoadPlatformFile(ctx, platformDir, loaderfile.LoadOptions{Registry: localRegistry})
	if err != nil {
		return fmt.Errorf("loading platform from %s: %w", platformDir, err)
	}
	plat, err := k.NewPlatformFromValue(platVal)
	if err != nil {
		return fmt.Errorf("constructing *platform.Platform: %w", err)
	}

	binding, err := api.Lookup(modVer)
	if err != nil {
		return fmt.Errorf("resolving binding: %w", err)
	}
	paths := binding.Paths()

	if want[stagePlatform] {
		header("Stage 2: Loaded Platform — opm-platform")

		subHeader("metadata + type")
		mustPrintConcreteCUE(platVal.LookupPath(cue.ParsePath("metadata")))
		fmt.Printf("\n  type: %q\n", plat.Metadata.Type)

		// Registry entries: print one summary line per registered Module.
		subHeader("#registry (registered Modules)")
		printRegistrySummary(platVal.LookupPath(paths.Registry))

		// Computed views are too large to dump fully (they recursively pull in
		// every primitive's schema). Print FQN keys instead so the reader sees
		// the "what is registered" shape without the noise.
		subHeader("#knownResources (FQNs)")
		printFQNKeys(platVal.LookupPath(paths.KnownResources))

		subHeader("#knownTraits (FQNs)")
		printFQNKeys(platVal.LookupPath(paths.KnownTraits))

		subHeader("#composedTransformers (FQNs)")
		printFQNKeys(platVal.LookupPath(paths.ComposedTransformers))

		subHeader("#matchers.resources (FQN → [transformer FQNs])")
		printMatcherIndex(platVal.LookupPath(paths.MatchersResources), paths)

		subHeader("#matchers.traits (FQN → [transformer FQNs])")
		printMatcherIndex(platVal.LookupPath(paths.MatchersTraits), paths)
	}

	// ── Build the release ───────────────────────────────────────────
	debugValues := modVal.LookupPath(paths.DebugValues)
	if !debugValues.Exists() {
		return fmt.Errorf("web_app fixture must define debugValues")
	}

	releaseSkeleton := k.CueContext().CompileString(`
apiVersion: "opmodel.dev/v1alpha2"
kind:       "ModuleRelease"
metadata: {
	name:      "web-app-demo"
	namespace: "default"
	uuid:      "11111111-2222-5333-8444-555555555555"
}
`, cue.Filename("release.cue"))
	if err := releaseSkeleton.Err(); err != nil {
		return fmt.Errorf("building release skeleton: %w", err)
	}

	unifiedModule := modVal.FillPath(paths.Config, debugValues)
	if err := unifiedModule.Err(); err != nil {
		return fmt.Errorf("filling values into module #config: %w", err)
	}
	moduleComponents := unifiedModule.LookupPath(cue.ParsePath("#components"))
	if !moduleComponents.Exists() {
		return fmt.Errorf("module exposes no #components after values are unified")
	}

	releaseSpec := releaseSkeleton.
		FillPath(paths.Module, modVal).
		FillPath(paths.Values, debugValues).
		FillPath(paths.Components, moduleComponents)
	if err := releaseSpec.Err(); err != nil {
		return fmt.Errorf("building release spec: %w", err)
	}

	rel, err := k.ProcessModuleRelease(ctx, releaseSpec, *mod, debugValues)
	if err != nil {
		return fmt.Errorf("processing module release: %w", err)
	}

	if want[stageRelease] {
		header("Stage 3: Constructed ModuleRelease — web-app-demo")
		subHeader("metadata")
		mustPrintConcreteCUE(rel.Package.LookupPath(cue.ParsePath("metadata")))
		subHeader("values (resolved from debugValues)")
		mustPrintConcreteCUE(rel.Package.LookupPath(paths.Values))
		subHeader("components (passed to matcher)")
		mustPrintConcreteCUE(rel.Package.LookupPath(paths.Components))
	}

	// ── Match + Compile ─────────────────────────────────────────────
	if !want[stagePlan] {
		return nil
	}

	header("Stage 4: Plan, Match, and Compile outputs")

	plan, err := k.Match(ctx, kernel.MatchInput{ModuleRelease: rel, Platform: plat})
	if err != nil {
		return fmt.Errorf("match: %w", err)
	}

	subHeader("MatchPlan.MatchedPairs()")
	for _, p := range plan.MatchedPairs() {
		fmt.Printf("  %s  →  %s\n", p.ComponentName, p.TransformerFQN)
	}
	if len(plan.MatchedPairs()) == 0 {
		fmt.Println("  (no matched pairs)")
	}

	subHeader("MatchPlan.Unmatched / UnhandledTraits")
	fmt.Printf("  Unmatched:        %v\n", plan.Unmatched)
	fmt.Printf("  UnhandledTraits:  %v\n", plan.UnhandledTraits)
	if w := plan.Warnings(); len(w) > 0 {
		fmt.Println("  Warnings:")
		for _, msg := range w {
			fmt.Printf("    - %s\n", msg)
		}
	}

	out, err := k.Compile(ctx, kernel.CompileInput{
		ModuleRelease: rel,
		Platform:      plat,
		RuntimeName:   "flow-inspect",
	})
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	subHeader(fmt.Sprintf("CompileResult.Compiled (%d items)", len(out.Compiled)))
	for i, c := range out.Compiled {
		fmt.Printf("\n--- Compiled[%d]: component=%s transformer=%s ---\n", i, c.Component, c.Transformer)
		mustPrintConcreteCUE(c.Value)
	}

	if len(out.Components) > 0 {
		subHeader("CompileResult.Components (per-component summary)")
		for _, s := range out.Components {
			fmt.Printf("  %s  labels=%v  resources=%v  traits=%v\n", s.Name, s.Labels, s.ResourceFQNs, s.TraitFQNs)
		}
	}

	if len(out.Warnings) > 0 {
		subHeader("CompileResult.Warnings")
		for _, msg := range out.Warnings {
			fmt.Printf("  - %s\n", msg)
		}
	}

	return nil
}

// header prints a top-level stage banner with consistent spacing.
func header(title string) {
	fmt.Printf("\n%s\n%s\n%s\n\n", strings.Repeat("=", 78), title, strings.Repeat("=", 78))
}

// subHeader prints a section banner inside a stage.
func subHeader(title string) {
	fmt.Printf("\n--- %s ---\n", title)
}

// mustPrintCUE formats a cue.Value as pretty CUE source preserving the
// definitions and disjunctions exactly as authored. Use this for inputs
// (the loaded module) where the reader wants to see the schema shape.
func mustPrintCUE(v cue.Value) {
	printCUE(v, cue.Definitions(true), cue.Optional(true), cue.Attributes(false), cue.Docs(false))
}

// mustPrintConcreteCUE formats a cue.Value as fully-evaluated CUE source —
// constraints collapsed to their concrete values, defaults resolved. Use
// this for downstream stages (release, compiled outputs) where the reader
// wants the post-evaluation data, not the constraint tree.
//
// cue.Docs(false) is forced — Final() preserves the import-side comments
// from the upstream schemas (k8s.io, opm/schemas), which drown the
// post-evaluation data in pages of doc strings. The constraint tree itself
// is what we want, not the docstrings the schema authors attached to it.
func mustPrintConcreteCUE(v cue.Value) {
	printCUE(v, cue.Final(), cue.Concrete(true), cue.Attributes(false), cue.Docs(false))
}

// printCUE is the shared formatter; both wrappers above feed it different
// Syntax options. Bottoms / non-existent values surface as a one-line
// marker so the reader sees there was nothing to print rather than silent
// emptiness.
func printCUE(v cue.Value, opts ...cue.Option) {
	if !v.Exists() {
		fmt.Println("  <field does not exist>")
		return
	}
	if err := v.Err(); err != nil {
		fmt.Printf("  <error: %v>\n", err)
		return
	}
	node := v.Syntax(opts...)
	if node == nil {
		fmt.Println("  <empty>")
		return
	}
	stripComments(node)
	out, err := format.Node(node, format.Simplify())
	if err != nil {
		fmt.Printf("  <format error: %v>\n", err)
		return
	}
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		fmt.Printf("  %s\n", line)
	}
}

// stripComments walks the CUE AST and detaches every comment group.
// cue.Docs(false) on Syntax() does not propagate through cue.Final() —
// the upstream schemas (k8s.io, opm/schemas) bring along multi-line
// docstrings that drown the post-evaluation data we actually want to
// inspect. astutil.Apply lets us clear the SetComments slot on every
// node before format.Node serialises it.
func stripComments(node ast.Node) {
	astutil.Apply(node, func(c astutil.Cursor) bool {
		if cn, ok := c.Node().(interface {
			SetComments([]*ast.CommentGroup)
		}); ok {
			cn.SetComments(nil)
		}
		return true
	}, nil)
}

// printFQNKeys lists field keys of a value that maps FQNs to dense subtrees
// (transformers, matcher buckets). Dumping the full subtree drowns the
// inspection output; the keys alone are usually what a reader wants.
func printFQNKeys(v cue.Value) {
	if !v.Exists() {
		fmt.Println("  <field does not exist>")
		return
	}
	iter, err := v.Fields()
	if err != nil {
		fmt.Printf("  <iter error: %v>\n", err)
		return
	}
	any := false
	for iter.Next() {
		any = true
		fmt.Printf("  - %s\n", iter.Selector().Unquoted())
	}
	if !any {
		fmt.Println("  (empty)")
	}
}

// printRegistrySummary prints one line per registered Module (Id, FQN,
// enabled). Avoids dumping the whole #Module value (which is the entire
// catalog) for every entry.
func printRegistrySummary(registry cue.Value) {
	if !registry.Exists() {
		fmt.Println("  <field does not exist>")
		return
	}
	iter, err := registry.Fields()
	if err != nil {
		fmt.Printf("  <iter error: %v>\n", err)
		return
	}
	any := false
	for iter.Next() {
		any = true
		id := iter.Selector().Unquoted()
		entry := iter.Value()
		fqn, _ := entry.LookupPath(cue.ParsePath("#module.metadata.fqn")).String()
		enabled := true
		if e := entry.LookupPath(cue.ParsePath("enabled")); e.Exists() {
			if v, err := e.Bool(); err == nil {
				enabled = v
			}
		}
		fmt.Printf("  - %s  (fqn=%s, enabled=%v)\n", id, fqn, enabled)
	}
	if !any {
		fmt.Println("  (empty)")
	}
}

// printMatcherIndex prints each FQN's bucket as the FQNs of the candidate
// transformers, so the reader sees which transformers compete for each
// primitive without the dense per-transformer subtree.
func printMatcherIndex(idx cue.Value, paths api.Paths) {
	if !idx.Exists() {
		fmt.Println("  <field does not exist>")
		return
	}
	iter, err := idx.Fields()
	if err != nil {
		fmt.Printf("  <iter error: %v>\n", err)
		return
	}
	any := false
	for iter.Next() {
		any = true
		fqn := iter.Selector().Unquoted()
		bucket := iter.Value()
		var tfFQNs []string
		listIter, err := bucket.List()
		if err == nil {
			for listIter.Next() {
				if s, err := listIter.Value().LookupPath(paths.MetadataFQN).String(); err == nil {
					tfFQNs = append(tfFQNs, s)
				}
			}
		}
		fmt.Printf("  - %s\n", fqn)
		for _, tf := range tfFQNs {
			fmt.Printf("      → %s\n", tf)
		}
	}
	if !any {
		fmt.Println("  (empty)")
	}
}

// resolveLibraryRoot returns an absolute path to the library/ directory.
// Default: the current working directory, on the assumption the user runs
// `go run ./cmd/flow-inspect` from library/.
func resolveLibraryRoot(override string) (string, error) {
	if override != "" {
		abs, err := filepath.Abs(override)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return cwd, nil
}

// parseStages turns the comma-separated -stages flag into a set; empty input
// expands to every stage so the default is "show me everything".
func parseStages(raw string) (map[stage]bool, error) {
	want := map[stage]bool{}
	if raw == "" {
		for _, s := range allStages {
			want[s] = true
		}
		return want, nil
	}
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		s := stage(name)
		switch s {
		case stageModule, stagePlatform, stageRelease, stagePlan:
			want[s] = true
		default:
			return nil, fmt.Errorf("unknown stage %q (valid: module, platform, release, plan)", name)
		}
	}
	return want, nil
}

// checkRegistry probes localhost:5000 with a short deadline. The probe is the
// same one the integration test uses; sharing the contract means a passing
// `task cue:test:flow` implies `flow-inspect` will also succeed.
func checkRegistry() error {
	if os.Getenv("OPM_FLOW_INSPECT_FORCE") == "1" {
		return nil
	}
	conn, err := net.DialTimeout("tcp", "localhost:5000", 300*time.Millisecond)
	if err != nil {
		return err
	}
	return conn.Close()
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "flow-inspect: "+format+"\n", args...)
	os.Exit(1)
}
