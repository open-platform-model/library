// Harness validating 006 D13's FillPath claim AND demonstrating the revised
// kernel-driven writeback mechanism (exp 02 finding F1).
//
// The end-user-authored artifact is `cases/jellyfin/release-unbound.cue` — it
// has neither #platform set nor any matched #consumes values. The kernel's
// job at apply time is to:
//  1. Compute matched #consumes by walking the module's declared FQNs
//     against the platform's #provides map.
//  2. FillPath every matched entry into the module's #consumes.
//  3. FillPath the chosen #platform onto the release.
//  4. Evaluate the release.
//
// This program performs steps 1–4 against the unbound fixture, prints the
// resolved component value, and exits non-zero if the value does not match
// the bound-release reference.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	// Resolve the experiment root from this file's location so the program
	// works no matter where it is invoked from.
	_, thisFile, _, _ := runtime.Caller(0)
	expRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))

	ctx := cuecontext.New()

	// Load the jellyfin case package — module.cue + platform-prod.cue +
	// release-unbound.cue. We deliberately do NOT include release-bound.cue,
	// since the Go harness is reproducing the bound state externally.
	cfg := &load.Config{
		Dir: expRoot,
	}
	instances := load.Instances([]string{
		"./cases/jellyfin/module.cue",
		"./cases/jellyfin/platform-prod.cue",
		"./cases/jellyfin/release-unbound.cue",
	}, cfg)
	if len(instances) != 1 {
		return fmt.Errorf("expected 1 instance, got %d", len(instances))
	}
	inst := instances[0]
	if inst.Err != nil {
		return fmt.Errorf("load: %w", inst.Err)
	}

	root := ctx.BuildInstance(inst)
	if err := root.Err(); err != nil {
		return fmt.Errorf("build: %w", err)
	}

	// Read the unbound release, the platform, and the module's declared
	// #consumes.required map.
	release := root.LookupPath(cue.ParsePath("ReleaseUnbound"))
	platform := root.LookupPath(cue.ParsePath("ProdPlatform"))
	module := root.LookupPath(cue.ParsePath("JellyfinModule"))

	consumedRequired := module.LookupPath(cue.ParsePath("#consumes.required"))
	if err := consumedRequired.Err(); err != nil {
		return fmt.Errorf("locate module.#consumes.required: %w", err)
	}

	provides := platform.LookupPath(cue.ParsePath("#provides"))
	if err := provides.Err(); err != nil {
		return fmt.Errorf("locate platform.#provides: %w", err)
	}

	// Step 1+2 (the kernel match + writeback). For every FQN the module
	// declares as required, look up the platform's #provides[fqn]. If found,
	// FillPath the provider value into the module's #consumes.required[fqn]
	// at the release's #module path.
	iter, err := consumedRequired.Fields(cue.Optional(true), cue.Definitions(true))
	if err != nil {
		return fmt.Errorf("iterate consumed required: %w", err)
	}
	bound := release
	for iter.Next() {
		fqn := iter.Selector().Unquoted()
		providerValue := provides.LookupPath(cue.MakePath(cue.Str(fqn)))
		if providerValue.Err() != nil {
			// No provider — leave the entry incomplete; cue vet -c will
			// report it as the actionable diagnostic.
			fmt.Fprintf(os.Stderr, "[kernel] %s: no provider, leaving unresolved\n", fqn)
			continue
		}
		fmt.Fprintf(os.Stderr, "[kernel] %s: filling matched provider value\n", fqn)
		writeback := cue.MakePath(
			cue.Def("#module"),
			cue.Def("#consumes"),
			cue.Str("required"),
			cue.Str(fqn),
		)
		bound = bound.FillPath(writeback, providerValue)
	}

	// Step 3 — fill the kernel-populated #platform field on the release.
	bound = bound.FillPath(cue.MakePath(cue.Def("#platform")), platform)

	if err := bound.Err(); err != nil {
		return fmt.Errorf("bound release has errors: %w", err)
	}

	// Step 4 — evaluate. Read the component value the experiment cares about.
	resolvedValue := bound.LookupPath(cue.ParsePath("components.app.env.AppHost.value"))
	if err := resolvedValue.Err(); err != nil {
		return fmt.Errorf("lookup components.app.env.AppHost.value: %w", err)
	}

	str, err := resolvedValue.String()
	if err != nil {
		return fmt.Errorf("expected resolved value to be a concrete string, got: %w", err)
	}

	const want = "jellyfin.apps.example.com"
	fmt.Printf("components.app.env.AppHost.value = %q\n", str)
	if str != want {
		return fmt.Errorf("mismatch: got %q, want %q", str, want)
	}
	fmt.Println("OK — FillPath kernel-style writeback resolved the read-surface interpolation.")
	return nil
}
