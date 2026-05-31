// Package crosscontext is the keystone spike for
// openspec/changes/spike-concurrent-render-v0170.
//
// It answers the one fact the whole concurrent-render design rests on, with
// NO library / registry / fixtures — just raw CUE:
//
//   - Is combining values from DIFFERENT *cue.Context legal? (Unify/FillPath)
//   - Is doing so CONCURRENTLY race-clean, when many goroutines each own a
//     context and all combine against ONE shared value built in another
//     context? (the "per-goroutine Kernel Compiles against a shared
//     *MaterializedPlatform" pattern, reduced to its CUE core)
//
// Run on the control (v0.16.1) — expected to fail/race — and on the candidate
// (v0.17.0-alpha.1) — expected clean. Swap the require in go.mod between runs.
package crosscontext

import (
	"fmt"
	"sync"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

// TestCrossContextUnify: can a value from ctxA combine with a value from ctxB?
// On v0.16 this is documented as undefined/illegal; v0.17 deprecates
// Value.Context() precisely because cross-context combination is supported.
func TestCrossContextUnify(t *testing.T) {
	ctxA := cuecontext.New()
	ctxB := cuecontext.New()

	a := ctxA.CompileString(`{x: int, shared: "from-A"}`)
	if err := a.Err(); err != nil {
		t.Fatalf("compile a: %v", err)
	}
	b := ctxB.CompileString(`{x: 42}`)
	if err := b.Err(); err != nil {
		t.Fatalf("compile b: %v", err)
	}

	u := a.Unify(b) // cross-context combine
	if err := u.Validate(); err != nil {
		t.Fatalf("cross-context Unify validate: %v", err)
	}
	x, err := u.LookupPath(cue.ParsePath("x")).Int64()
	if err != nil || x != 42 {
		t.Fatalf("cross-context Unify result x=%d err=%v", x, err)
	}
}

// TestConcurrentCrossContextFill reduces the production pattern to CUE:
// ONE shared value built in ctx0 (the "materialized platform"), and N
// goroutines each owning their own context (a "per-goroutine Kernel") that
// FillPath a per-goroutine value INTO a value derived from the shared one,
// concurrently, under -race.
func TestConcurrentCrossContextFill(t *testing.T) {
	ctx0 := cuecontext.New()
	// A "transformer output" shape with an open slot — stands in for a
	// #composedTransformers entry on a shared materialized platform.
	shared := ctx0.CompileString(`{
		output: { kind: "Demo", spec: { name: string } }
	}`)
	if err := shared.Err(); err != nil {
		t.Fatalf("compile shared: %v", err)
	}
	sharedOut := shared.LookupPath(cue.ParsePath("output"))

	const goroutines = 32
	const iters = 200

	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctxN := cuecontext.New() // one context per goroutine, like one Kernel per goroutine
			want := fmt.Sprintf("c-%d", id)
			comp := ctxN.CompileString(fmt.Sprintf(`{ spec: name: %q }`, want))
			if err := comp.Err(); err != nil {
				errs <- fmt.Errorf("g%d compile comp: %w", id, err)
				return
			}
			compSpec := comp.LookupPath(cue.ParsePath("spec"))
			for j := 0; j < iters; j++ {
				// cross-context FillPath: per-goroutine value into shared-derived value
				out := sharedOut.FillPath(cue.ParsePath("spec"), compSpec)
				if err := out.Validate(cue.Concrete(true)); err != nil {
					errs <- fmt.Errorf("g%d iter%d validate: %w", id, j, err)
					return
				}
				got, err := out.LookupPath(cue.ParsePath("spec.name")).String()
				if err != nil || got != want {
					errs <- fmt.Errorf("g%d iter%d got=%q want=%q err=%v", id, j, got, want, err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

// The benchmarks answer "safe ≠ parallel": does concurrent cross-context
// FillPath actually scale with cores, or does CUE serialize evaluation
// internally? Run: go test -bench=Fill -benchmem -run='^$' -cpu=1,2,4,8
// Compare BenchmarkFillConcurrent ns/op across -cpu: ~linear drop = real
// parallelism; flat = internal serialization (no win over a mutex).

func sharedOutput(tb testing.TB) cue.Value {
	ctx0 := cuecontext.New()
	shared := ctx0.CompileString(`{ output: { kind: "Demo", spec: { name: string } } }`)
	if err := shared.Err(); err != nil {
		tb.Fatalf("compile shared: %v", err)
	}
	return shared.LookupPath(cue.ParsePath("output"))
}

// BenchmarkFillSerial: one context, one goroutine, repeated cross-context Fill.
func BenchmarkFillSerial(b *testing.B) {
	sharedOut := sharedOutput(b)
	ctxN := cuecontext.New()
	comp := ctxN.CompileString(`{ spec: name: "c" }`)
	spec := comp.LookupPath(cue.ParsePath("spec"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := sharedOut.FillPath(cue.ParsePath("spec"), spec)
		if _, err := out.LookupPath(cue.ParsePath("spec.name")).String(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFillConcurrent: GOMAXPROCS goroutines, each its own context (one
// Kernel per goroutine), all cross-context Fill against ONE shared value.
func BenchmarkFillConcurrent(b *testing.B) {
	sharedOut := sharedOutput(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctxN := cuecontext.New()
		comp := ctxN.CompileString(`{ spec: name: "c" }`)
		spec := comp.LookupPath(cue.ParsePath("spec"))
		for pb.Next() {
			out := sharedOut.FillPath(cue.ParsePath("spec"), spec)
			if _, err := out.LookupPath(cue.ParsePath("spec.name")).String(); err != nil {
				b.Fatal(err)
			}
		}
	})
}
