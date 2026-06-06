// Package synth builds OPM artifact CUE values from in-memory typed inputs by
// unifying caller-supplied identity (name, namespace, module reference,
// values, subscriptions, labels, annotations) against the schema definition
// resolved through the caller-supplied *schema.Cache.
//
// It covers two of the three OPM artifacts:
//
//   - synth.Release builds a #ModuleRelease from a *module.Module plus typed
//     identity and values.
//   - synth.Platform builds a #Platform from typed identity and subscription
//     inputs.
//
// Synth is a peer of opm/helper/loader, not a subpackage of it. The loader
// tree (opm/helper/loader/file, opm/helper/loader/bytes) reads existing
// artifact bytes from a source — filesystem or byte buffer. Synthesis is
// creation from typed inputs: there is no file to parse, no bytes to decode.
// Co-locating synth under loader would conflate verbs and force every reader
// to ignore the package doc when interpreting the package path. Keeping them
// as peers makes each verb legible at a glance.
//
// Recommended entry points: (*kernel.Kernel).SynthesizeRelease and
// (*kernel.Kernel).SynthesizePlatform. SynthesizeRelease chains synth.Release
// into Kernel.ProcessModuleRelease so a caller building a release from typed
// inputs gets a fully validated, concrete *module.Release in one call.
// SynthesizePlatform chains synth.Platform into platform.NewPlatformFromValue
// and returns a typed pre-materialize *platform.Platform; it does NOT call
// Materialize (registry I/O stays an explicit, separate, caller-driven step).
// Call synth.Release / synth.Platform directly only when there is no *Kernel
// on hand (rare; mostly unit-testing in tight loops).
//
// Schema source of truth: the synth helpers never reimplement derivations the
// CUE schema already owns (release UUID stamping, components fan-out from
// #components, auto-secrets injection, standard label stamping, the
// #Subscription enable default). Every derived field flows through unification
// with the artifact definition obtained from in.SchemaCache.Get(ctx); the
// schema package itself is resolved by the Cache's underlying Loader
// (typically schema.OCILoader against CUE_REGISTRY).
//
// Boundary scope: synth helpers live under opm/helper/ because they are
// opinionated frontend conveniences. Frontends MAY bypass them and unify
// against the schema themselves; nothing in synth is part of the kernel
// contract.
package synth
