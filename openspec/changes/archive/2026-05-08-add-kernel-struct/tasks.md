## 1. Package Skeleton

- [x] 1.1 Create `opm/kernel/` directory and `kernel.go` with package doc comment that states the goroutine-safety contract and shows a one-Kernel-per-goroutine example
- [x] 1.2 Define the unexported `Kernel` struct fields: `cueCtx *cue.Context`, `logger *slog.Logger`, `tracer trace.Tracer`, `clock Clock`
- [x] 1.3 Define the `Option` type as `type Option func(*Kernel)`
- [x] 1.4 Define the `Clock` interface as `type Clock interface { Now() time.Time }`; provide an unexported `systemClock` default

## 2. Constructor and Accessors

- [x] 2.1 Implement `func New(opts ...Option) *Kernel`; default to `cuecontext.New()`, no-op `slog.Logger`, no-op `trace.Tracer`, `systemClock`
- [x] 2.2 Implement `WithLogger(*slog.Logger) Option`
- [x] 2.3 Implement `WithTracer(trace.Tracer) Option`
- [x] 2.4 Implement `WithClock(Clock) Option`
- [x] 2.5 Implement `(k *Kernel) CueContext() *cue.Context` with godoc marking it as advanced

## 3. Wrapper Methods

- [x] 3.1 Add `(k *Kernel) LoadModulePackage(ctx context.Context, dirPath string) (cue.Value, apiversion.Version, error)` delegating to `loader.LoadModulePackage(k.cueCtx, dirPath)` (mirrors the underlying function's full triple)
- [x] 3.2 Add `(k *Kernel) LoadReleaseFile(ctx context.Context, filePath string, opts loader.LoadOptions) (cue.Value, string, apiversion.Version, error)` delegating to `loader.LoadReleaseFile` (mirrors the underlying function's full quad)
- [x] 3.3 Add `(k *Kernel) LoadValuesFile(ctx context.Context, path string) (cue.Value, error)` delegating to `loader.LoadValuesFile`
- [x] 3.4 Add `(k *Kernel) LoadProvider(providerName string, providers map[string]cue.Value) (*provider.Provider, error)` delegating to `loader.LoadProvider`
- [x] 3.5 Add `(k *Kernel) ParseModuleRelease(ctx context.Context, spec cue.Value, mod module.Module, values []cue.Value) (*module.Release, error)` delegating to `module.ParseModuleRelease`
- [x] 3.6 Add `(k *Kernel) NewRenderModule(p *provider.Provider, runtimeName string) *render.Module` delegating to `render.NewModule`
- [x] 3.7 Add `(k *Kernel) ProcessModuleRelease(ctx context.Context, rel *module.Release, p *provider.Provider, runtimeName string) (*render.ModuleResult, error)` delegating to `render.ProcessModuleRelease`
- [x] 3.8 Add `(k *Kernel) ValidateConfig(schema cue.Value, values []cue.Value, contextLabel, name string) (cue.Value, *oerrors.ConfigError)` delegating to `validate.Config`

## 4. Deprecation Annotations

- [x] 4.1 Add `// Deprecated: use Kernel.LoadModulePackage` doc comment to `loader.LoadModulePackage`
- [x] 4.2 Add equivalent `// Deprecated:` annotations to `loader.LoadReleaseFile`, `loader.LoadValuesFile`, `loader.LoadProvider`, `module.ParseModuleRelease`, `render.NewModule`, `render.ProcessModuleRelease`, `validate.Config`
- [x] 4.3 Verify each deprecation annotation references the new method name

## 5. Tests

- [x] 5.1 Add `opm/kernel/kernel_test.go` covering: default construction, construction with each option, identity of `k.CueContext()` across calls
- [x] 5.2 Add a parity test for each wrapper method confirming results identical to the underlying free function
- [x] 5.3 Add a goroutine-safety regression test that constructs N kernels (one per goroutine), runs each through a basic Load + Process cycle, and asserts no race detector complaints
- [x] 5.4 Confirm existing test suites in `opm/loader/`, `opm/module/`, `opm/render/`, `opm/validate/` continue to pass without modification

## 6. Documentation

- [x] 6.1 Update `library/README.md` Quick Start section to show the Kernel construction form alongside the existing free-function form
- [x] 6.2 Add a `opm/kernel/doc.go` (or top-of-file package doc) covering: purpose, goroutine-safety, one-per-goroutine pattern, advanced `CueContext()` accessor

## 7. Validation

- [x] 7.1 Run `task fmt` and confirm no diff
- [x] 7.2 Run `task vet` and confirm clean
- [x] 7.3 Run `task lint` and confirm clean
- [x] 7.4 Run `task test` and confirm all packages pass
- [x] 7.5 Run `task check` end-to-end
