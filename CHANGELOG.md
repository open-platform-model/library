# Changelog

## [0.5.2](https://github.com/open-platform-model/library/compare/v0.5.1...v0.5.2) (2026-06-17)


### Miscellaneous Chores

* **docs:** cue closedness regression alpha2 ([#18](https://github.com/open-platform-model/library/issues/18)) ([5305aaa](https://github.com/open-platform-model/library/commit/5305aaa193dd2de488ab548a5a70febb89b9321e))

## [0.5.1](https://github.com/open-platform-model/library/compare/v0.5.0...v0.5.1) (2026-06-15)


### Bug Fixes

* **materialize:** source transforms from open composed map ([#16](https://github.com/open-platform-model/library/issues/16)) ([045414c](https://github.com/open-platform-model/library/commit/045414cd2b52ab86c638bf338315e95332d16d3b))

## [0.5.0](https://github.com/open-platform-model/library/compare/v0.4.0...v0.5.0) (2026-06-13)


### Features

* **loader:** add registry module loader for published modules ([#14](https://github.com/open-platform-model/library/issues/14)) ([c315f8d](https://github.com/open-platform-model/library/commit/c315f8d00dcdd1b48c4e68634d288c09a07397d0))

## [0.4.0](https://github.com/open-platform-model/library/compare/v0.3.1...v0.4.0) (2026-06-13)


### Features

* **security-audit:** add cue-kernel security audit skill ([e53a2b1](https://github.com/open-platform-model/library/commit/e53a2b17236f4722f0e05617f9ef94815c5005eb))

## [0.3.1](https://github.com/open-platform-model/library/compare/v0.3.0...v0.3.1) (2026-06-07)


### Documentation

* **changelog:** backfill v0.3.0 entries lost to squash merge ([8a0b522](https://github.com/open-platform-model/library/commit/8a0b522bdea010853c289eeeb802e1da4a50db32))

## [0.3.0](https://github.com/open-platform-model/library/compare/v0.2.1...v0.3.0) (2026-06-06)


### ⚠ BREAKING CHANGES

* **compile:** `compile.NewModule` now takes a `*cue.Context` as its first argument. `Kernel.Compile` is the only in-tree caller and downstream consumers go through `Kernel`, so they are unaffected. See MIGRATIONS.md.

### Features

* **helper:** add platform synthesis from typed inputs — `synth.Platform` + `Kernel.SynthesizePlatform`, the typed in-memory path to a `#Platform` (peer of release synthesis; stops before `Materialize`) ([4b973ec](https://github.com/open-platform-model/library/commit/4b973ec52894009eeb463141612dfbe91ba20b70))
* **kernel:** guarantee concurrent render against a shared platform — a once-materialized `*MaterializedPlatform` is safe to read concurrently from many per-goroutine Kernels' `Compile` calls ([4b973ec](https://github.com/open-platform-model/library/commit/4b973ec52894009eeb463141612dfbe91ba20b70))

### Code Refactoring

* **compile:** source `cue.Context` from the caller Kernel — renders build in the caller's context, consuming the materialized platform as read-only input (lands the v0.16-landable half of adr/002) ([4b973ec](https://github.com/open-platform-model/library/commit/4b973ec52894009eeb463141612dfbe91ba20b70))

### Documentation

* **openspec:** add platform-synthesis change proposal ([4b973ec](https://github.com/open-platform-model/library/commit/4b973ec52894009eeb463141612dfbe91ba20b70))
* **openspec:** sync concurrent-render-recontract deltas to specs ([4b973ec](https://github.com/open-platform-model/library/commit/4b973ec52894009eeb463141612dfbe91ba20b70))

### Miscellaneous Chores

* **openspec:** archive concurrent-render-recontract ([4b973ec](https://github.com/open-platform-model/library/commit/4b973ec52894009eeb463141612dfbe91ba20b70))
* **openspec:** archive remove-library-catalog ([4b973ec](https://github.com/open-platform-model/library/commit/4b973ec52894009eeb463141612dfbe91ba20b70))
* **spike:** add cross-context cue concurrency spike ([4b973ec](https://github.com/open-platform-model/library/commit/4b973ec52894009eeb463141612dfbe91ba20b70))

> Note: this 0.3.0 section was authored by hand. PR #8 squash-merged with a
> non-conventional title (`Feat/concurrent render recontract`), so
> release-please could not parse it; the entries above were reconstructed
> from the squash commit body (`4b973ec`). Hidden `build`/`ci`/`test` work
> (CUE v0.17 toolchain adoption, the hermetic kernel integration suite,
> `setup-cue` pin) shipped in the same merge — see MIGRATIONS.md.

## [0.2.1](https://github.com/open-platform-model/library/compare/v0.2.0...v0.2.1) (2026-05-29)


### Documentation

* **changelog:** backfill v0.2.0 entries lost to squash merge ([2f458f7](https://github.com/open-platform-model/library/commit/2f458f7e5d112a2ff85c5c2db20d0ec7144f3514))

## [0.2.0](https://github.com/open-platform-model/library/compare/v0.1.0...v0.2.0) (2026-05-29)


### ⚠ BREAKING CHANGES

* **compile:** Match/Plan/Compile take *materialize.MaterializedPlatform; callers must Materialize first. Compose/ComposePlatform removed. See MIGRATIONS.md.
* **schema:** replace embedded schema with OCI loader; opm/schema gains a Loader interface + OCILoader resolving opmodel.dev/core@v0 via CUE_REGISTRY. library/apis removed; SchemaValue/EmbeddedSchema removed; synth.ReleaseInput requires a SchemaCache field.
* **kernel:** drop api binding dispatch; centralise schema knowledge in opm/schema
* **api:** rename defaultNamespace annotation to kebab-case
* **loader:** replace LoadPlatformFile with LoadPlatformPackage
* **library:** all github.com/open-platform-model/library/pkg/* import paths are now github.com/open-platform-model/library/opm/*. Downstream consumers must rewrite imports accordingly.
* **loader:** unify module and release loading as packages
* **synth:** add release synthesis helper and Binding.SchemaValue
* **kernel:** remove deprecated ParseModuleRelease alias
* **compile:** MatchPlan.Ambiguous, CompileResult.Ambiguous, PlanResult.Ambiguous, *MultiFulfillerError, and the schema's _noMultiFulfiller / _invalid projections are removed. Catalog transformers that emit multiple resources must use list outputs instead of map-keyed-by-name.
* **apis:** implement 014 platform construct schemas
* **apis:** hoist core CUE module to apis/core/ root
* **kernel:** redesign config validation around CUE-native errors
* **kernel:** fold deprecated free functions into Kernel
* **kernel:** slim Match/Plan/Compile inputs (slice 11)
* **core:** rename Rendered to Compiled
* **helper:** add tiered values validation helper (slice 05)
* **kernel:** accept single cue.Value for values input (slice 04)
* **kernel:** rewrite match around Platform; retire Provider (slice 09)

### Features

* **api:** retire ModuleDebug; expose Paths.DebugValues (slice 03) ([8e8a5ad](https://github.com/open-platform-model/library/commit/8e8a5adc9752a5ae20d5a0e3c24f416c39db33b3))
* **apis:** add core v1alpha1 and v1alpha2 cue schemas ([967853d](https://github.com/open-platform-model/library/commit/967853d0ec5e37813afc9ce33dede6e4e601b64e))
* **apis:** implement 014 platform construct schemas ([14fb134](https://github.com/open-platform-model/library/commit/14fb1346eb14efeff331c796c503b12cf7df4140))
* **catalog:** repackage onto core@v0 with #Catalog + identity ([159d0a5](https://github.com/open-platform-model/library/commit/159d0a59733e1ce95d2f98c5fe336cb302f99aeb))
* **compile:** rewrite matcher onto MaterializedPlatform with unify rung ([071fb0d](https://github.com/open-platform-model/library/commit/071fb0d6131d99cc004b0a3c35e42308d5dc763f))
* **enhancements:** add lifecycle gates, metadata fields, and check task ([8fb8270](https://github.com/open-platform-model/library/commit/8fb82709751c236dbf0f593b9728695aef9b3af0))
* **enhancements:** tighten implementation.date to status=complete only ([ae11f73](https://github.com/open-platform-model/library/commit/ae11f736af11aa7b043c9d015c82be5168e467fc))
* **helper:** add Platform composition helper (slice 10) ([71e0448](https://github.com/open-platform-model/library/commit/71e0448caa9984ab489e76c6267e1723cebb2a45))
* **helper:** add tiered values validation helper (slice 05) ([44cae25](https://github.com/open-platform-model/library/commit/44cae25b75493f87a03e56c90b018da2b19ceb0e))
* **helper:** move loader under pkg/helper/loader/file (slice 07) ([42be960](https://github.com/open-platform-model/library/commit/42be9601b8df6410b1296e299fde349569a3afc7))
* **kernel:** accept single cue.Value for values input (slice 04) ([56c6255](https://github.com/open-platform-model/library/commit/56c6255dc6f0085bbaea73406fcae0afcc7d8be6))
* **kernel:** add phase methods and rename pkg/render to pkg/compile ([b3e2023](https://github.com/open-platform-model/library/commit/b3e20235b8b672a83ed47ba54c8318dea8c638d0))
* **kernel:** drop api binding dispatch; centralise schema knowledge in opm/schema ([4276ec4](https://github.com/open-platform-model/library/commit/4276ec4c2d18b6ecc3583f88a9baf01ce256be01))
* **kernel:** fold deprecated free functions into Kernel ([c6394ab](https://github.com/open-platform-model/library/commit/c6394ab90d260b6ccfae6ac2fd1febfd49059ed4))
* **kernel:** redesign config validation around CUE-native errors ([56a5ab2](https://github.com/open-platform-model/library/commit/56a5ab2598efded294892bd52efeec803e11f373))
* **kernel:** rewrite match around Platform; retire Provider (slice 09) ([7eba2b9](https://github.com/open-platform-model/library/commit/7eba2b98c580b434ce297f2672215061d3ca2ef0))
* **kernel:** slim Match/Plan/Compile inputs (slice 11) ([e93f8be](https://github.com/open-platform-model/library/commit/e93f8be3bffe3cc238b4c620f404e4b1d8615f95))
* **loader:** add shape-gate validation to package loaders ([218f61f](https://github.com/open-platform-model/library/commit/218f61f390de15bfd959d05455c3690837f039bd))
* **materialize:** realize platform subscriptions into MaterializedPlatform ([cd0182d](https://github.com/open-platform-model/library/commit/cd0182d2aeeec0cdc41ab16e48cc5bf88af13d2a))
* **modules:** add opm catalog module ([613efa9](https://github.com/open-platform-model/library/commit/613efa9495d43633862deec4f080fd92536f2ce2))
* **modules:** add opm-platform module ([6da1d3d](https://github.com/open-platform-model/library/commit/6da1d3d1fefd41c9467f26268f16528b0da35af9))
* **module:** unify artifact shape and land kernel scaffolding ([faf0b54](https://github.com/open-platform-model/library/commit/faf0b5488a141419517f324786d19d3f58ed7654))
* **platform:** add Platform artifact type and loader (slice 08) ([4af9f56](https://github.com/open-platform-model/library/commit/4af9f56ddb0abe2dea59dde712776943220df76a))
* **schema:** replace embedded schema with OCI loader ([4e83168](https://github.com/open-platform-model/library/commit/4e83168540863794537e6848431e7f6b2dcbcef9))
* **synth:** add release synthesis helper and Binding.SchemaValue ([1c3607f](https://github.com/open-platform-model/library/commit/1c3607f5634c29e66feb3bdec41aa8e024c59343))
* **transformers:** add gateway api route transformers ([bb53bb9](https://github.com/open-platform-model/library/commit/bb53bb92d5804188398652e8349845f57c1b5ab3))


### Bug Fixes

* **apis:** pin v1alpha2 #ApiVersion to literal string ([da2cacc](https://github.com/open-platform-model/library/commit/da2cacc4a78834bc4cee31d8e592d2c93b3d3560))
* **loader:** accept #Subscription platforms; restore flow integration tests ([91ba273](https://github.com/open-platform-model/library/commit/91ba273cf8343420d0e002d7c2979c7176ac8b26))


### Documentation

* **001:** mark implemented; record amendments and follow-up slices ([9a3b758](https://github.com/open-platform-model/library/commit/9a3b758b4b439d00038dd756f71d8ec02013f519))
* **002,005:** drop readme metadata tables; shorten titles ([97284e6](https://github.com/open-platform-model/library/commit/97284e6d2b8324e3f7c03019671f8a473ce16151))
* **002:** flag lifecycle-phase driving as OQ8 with trajectory sketch ([fe9f665](https://github.com/open-platform-model/library/commit/fe9f66565c2d645e225d3e9d3ee13ffcd804f48c))
* **003:** mark implemented; answer OQ3 transformer migration ([a0bee5e](https://github.com/open-platform-model/library/commit/a0bee5e0e03d6fec80715d10418fccc82484a6db))
* **004:** add in-tree experiments and link from README ([55523de](https://github.com/open-platform-model/library/commit/55523dee2bcc966524659955051bd8e75c5bbe03))
* **006:** add capability experiments; revise D5 per F1 ([8edf9db](https://github.com/open-platform-model/library/commit/8edf9db7b6e9ecdcf1032edc7ff0ff113a5e1012))
* **006:** drop stale 005 cross-ref; gofmt fillpath comments ([d398df6](https://github.com/open-platform-model/library/commit/d398df62d902b8c114eadbe8c6608fb3265bf657))
* add adr and enhancements templates from catalog ([ab2dd3d](https://github.com/open-platform-model/library/commit/ab2dd3d302af340e50318317e25c2269de63a717))
* add constitution and openspec config ([36d0be9](https://github.com/open-platform-model/library/commit/36d0be9141e0351d1afdbe3bb642f853fa5c33ab))
* **adr:** mark adr-001 accepted ([2767106](https://github.com/open-platform-model/library/commit/2767106ca7db771b2f1b388c0e25536c867b248c))
* **adr:** propose 001 module default namespace as annotation ([e437f82](https://github.com/open-platform-model/library/commit/e437f8290ff91ffa1d5b94acf3a540026613109a))
* **catalog:** mark phase-3 flow-test tasks complete ([a69a8cb](https://github.com/open-platform-model/library/commit/a69a8cbe83c423d3c6320a64dce23bfba4f52a06))
* **catalog:** mark phase-9 enhancement bookkeeping complete ([8fc7a9f](https://github.com/open-platform-model/library/commit/8fc7a9fcc2458a45ab4ce78ffb8c4baf9836cf23))
* **constitution:** list pkg/apiversion and pkg/api packages ([ae67e5c](https://github.com/open-platform-model/library/commit/ae67e5ce16824a1de11e3b09feed2cbaf3aad6d8))
* **enhancements:** add 007 platform registry subscription; supersede 003 ([5bf5d2f](https://github.com/open-platform-model/library/commit/5bf5d2f84d8cd3b85f2d8ccbf87657ab1e9fd41b))
* **enhancements:** add config.yaml metadata for 001-006 and graph ([38043c0](https://github.com/open-platform-model/library/commit/38043c0f18d515b704b896aaa12572ffb06f5543))
* **enhancements:** add optional experiments convention to template ([6bbd40a](https://github.com/open-platform-model/library/commit/6bbd40ad29f4018f460e4eda86e3b9531aadcea7))
* **enhancements:** drop prose duplicating config.yaml metadata ([2c8a106](https://github.com/open-platform-model/library/commit/2c8a106b7f46e7166ada2dc752781ff6b5e6ff8a))
* **enhancements:** migrate platform/claims/module-context from catalog ([1f702e4](https://github.com/open-platform-model/library/commit/1f702e46c3ca965bf6c2ca56f3ae7c6b4594d1bc))
* **enhancements:** slim 004 ctx.runtime; introduce 006 capabilities ([fa3cd32](https://github.com/open-platform-model/library/commit/fa3cd3239650199755e729ea292012fb7bde157e))
* **library:** split quick start into docs/getting-started.md ([feb29fa](https://github.com/open-platform-model/library/commit/feb29fafe6a66862fb30885136106468843e0123))
* **openspec:** archive add-kernel-struct slice ([60364be](https://github.com/open-platform-model/library/commit/60364be19a57188c4a0c8b6b2ee98756c242a2db))
* **openspec:** archive add-multi-apiversion-support slice ([ae7970c](https://github.com/open-platform-model/library/commit/ae7970c71aad295af1618f89b77033a04464add8))
* **openspec:** archive repackage-opm-catalog and sync catalog specs ([30df636](https://github.com/open-platform-model/library/commit/30df636257f1f0e8ffa2783c72ddb3e507ce6ad1))
* **openspec:** draft kernel-redesign slice proposals ([4272e96](https://github.com/open-platform-model/library/commit/4272e96d030269f9e325784d83aa26b1177057f3))
* **openspec:** draft retire-module-debug change ([549e862](https://github.com/open-platform-model/library/commit/549e8620dc6ea77c794133722ca386c766f942a7))
* **openspec:** propose remove-api-binding-dispatch change ([5545011](https://github.com/open-platform-model/library/commit/55450116cf9e5d87b43b047f35202d30b2c8c782))
* **openspec:** propose replace-embedded-schema-with-oci-loader change ([5877354](https://github.com/open-platform-model/library/commit/58773543d934020293bb6e1b70ed9070c364fb79))
* **openspec:** scope remove-api-binding-dispatch around post-0001 core schema ([d349bca](https://github.com/open-platform-model/library/commit/d349bca12c922d0ef059fc8cd7532f7191444174))


### Code Refactoring

* **api:** rename defaultNamespace annotation to kebab-case ([8154678](https://github.com/open-platform-model/library/commit/81546785820b07bcf400056948f3582d13f12c68))
* **apis:** hoist core CUE module to apis/core/ root ([ec7ca3c](https://github.com/open-platform-model/library/commit/ec7ca3cdc882d88ed8edfe599298249ae5ff8109))
* **apis:** rename #Transformer to #ComponentTransformer ([eb40056](https://github.com/open-platform-model/library/commit/eb40056294de9be6b5be2c5433d817f79b7514fa))
* **compile:** pair all transformers; kind-based output dispatch ([9ae1030](https://github.com/open-platform-model/library/commit/9ae1030e798595e67f176d409edc4d5ea3a09af1))
* **core:** rename Rendered to Compiled ([feccd1a](https://github.com/open-platform-model/library/commit/feccd1a684e073104227516bb4656e7025ef969e))
* **kernel:** remove deprecated ParseModuleRelease alias ([b3804e6](https://github.com/open-platform-model/library/commit/b3804e657af3b1f7b69946e6f08a446e758bc6f9))
* **library:** rename pkg/ to opm/ ([a195eb0](https://github.com/open-platform-model/library/commit/a195eb06251f594ddce569461b9841ee141cb150))
* **loader:** replace LoadPlatformFile with LoadPlatformPackage ([0aa8be3](https://github.com/open-platform-model/library/commit/0aa8be35b33952c7dc974069e4a651030e94e37b))
* **loader:** unify module and release loading as packages ([7c435f2](https://github.com/open-platform-model/library/commit/7c435f25d61d2765dfb5ee2932f3229f44402bc8))
* **modules/opm:** collocate schemas, flatten layout, drop area subpath from FQNs ([9bc22b4](https://github.com/open-platform-model/library/commit/9bc22b4210db116d682da5cfd2218a9300c81918))
* **modules/opm:** retire #defines aggregator; key transformers by metadata.fqn ([906848a](https://github.com/open-platform-model/library/commit/906848a57d1c0cee37dc22a84e11a69f0250faa3))
* **test:** drop legacy @v1 from opm_platform fixture module id ([bf47e6e](https://github.com/open-platform-model/library/commit/bf47e6edbf08d260749626df76f101647391c0b5))


### Miscellaneous Chores

* **agent:** update agent file ([f571047](https://github.com/open-platform-model/library/commit/f571047cfa37f2d81211562b2e74a882df9276ee))
* **apis:** add index generator script ([97bec14](https://github.com/open-platform-model/library/commit/97bec14160f5c44371ffd3f551c8f88dcaefab18))
* **apis:** drop unused v1alpha1 policy and primitives ([dc7453a](https://github.com/open-platform-model/library/commit/dc7453ad32cb5d3df9ca6e4bc72823a33111aa73))
* **apis:** remove deprecated v1alpha1 schemas and pkg/loader ([3a9a9bd](https://github.com/open-platform-model/library/commit/3a9a9bdfed465912cec79b4bf41a7353705916fa))
* **claude:** add project skills and slash commands ([37d4f92](https://github.com/open-platform-model/library/commit/37d4f929ec578e2aec6ebf4437b7ab81b95be389))
* ignore claude code local settings ([4efb7d5](https://github.com/open-platform-model/library/commit/4efb7d5eac43b2b2d345d27d36550fa0d435a342))
* **openspec:** Add change ([2e79a9b](https://github.com/open-platform-model/library/commit/2e79a9b8cb39be736a84a006528f5a5c45c2dc65))
* **openspec:** archive fold-deprecated-functions-into-kernel ([8c8d594](https://github.com/open-platform-model/library/commit/8c8d5942ee222c116b521d63d3da1669eb346d2e))
* **openspec:** archive introduce-tiered-validation ([84fbcb6](https://github.com/open-platform-model/library/commit/84fbcb6214a493a07c0449daec011a9ea1dada77))
* **openspec:** archive reorganize-helpers-under-helper ([2d087b9](https://github.com/open-platform-model/library/commit/2d087b926e4fb531f588226ff19b66a5ebe21c18))
* **openspec:** archive schema-testing capability changes ([7072027](https://github.com/open-platform-model/library/commit/7072027b4d3606241f81929d00a82edb079c95f6))
* **openspec:** archive slim-kernel-inputs ([72ab16c](https://github.com/open-platform-model/library/commit/72ab16cbdb4bef6726f16f1cfdb6d7af3185ad27))

## Changelog

All notable changes to this project are documented in this file.

Entries are generated by [release-please](https://github.com/googleapis/release-please) from [Conventional Commits](https://www.conventionalcommits.org/). The library follows [SemVer 2.0.0](https://semver.org/spec/v2.0.0.html) and distinguishes Go-module SemVer from OPM-schema versioning per the README.

For the pre-release API evolution and detailed breaking-change migration recipes, see [MIGRATIONS.md](./MIGRATIONS.md).
