---
title: "Embed the kernel"
description: "Load a module, validate values and render it from a Go program."
type: tutorial
sidebar:
  order: 10
---

<!-- Open with the end result, never "you will learn": a Go program, main.go, that fetches the published web_app module (opmodel.dev/modules/web_app@v1 at v1.0.4), checks a values file against the module's configuration, builds an instance named demo in namespace default, generates a platform that subscribes to the OPM catalog, renders the instance, and prints a Deployment and a Service as YAML. Nothing is applied to a cluster. One sentence on who this is for: Go developers building their own frontend, in the way the opm CLI and the operator embed the same kernel.
The sequence is New, AcquireModuleFromRegistry, LoadSourceFromFile, ValidateConfigDetailed, SynthesizeInstance, platformmodule Closure and Generate, AcquirePlatformFromDir, Render. There is no Materialize step: (*Kernel).Materialize was removed, and the render build now resolves the platform's catalogs itself. Enhancement 0018's design says the getting-started guide omits a mandatory Materialize step; that claim is stale, so do not add one. The getting-started guide is otherwise the closest source, but its module path example.com/modules/hello@v0 is not a real module and its "Go 1.22+" is out of date.
Check against: library/opm/kernel/doc.go, library/docs/getting-started.md (Removed entry points), library/opm/kernel/flow_synth_imported_test.go, library/opm/kernel/flow_integration_test.go -->

## Before you begin

<!-- Exact tools and versions, as links, with nothing explained: Go 1.25 or later (library/go.mod declares go 1.25.0); the library module github.com/open-platform-model/library at v1.0.0-alpha.33, the latest tag (Verify at writing); network access to ghcr.io and registry.cue.works, the two hosts in schema.PublicRegistry. No cue binary and no cluster are needed. Verify: web_app v1.0.4 and the catalog opm@v4 at 4.4.1 are published to GHCR and can be pulled anonymously. Check against: library/go.mod, library/.release-please-manifest.json, library/opm/schema/loader.go (PublicRegistry), modules/web_app/identity/identity.cue -->

## 1. Create a Go module

<!-- Commands: mkdir render-demo && cd render-demo, go mod init example.com/render-demo, go get github.com/open-platform-model/library@v1.0.0-alpha.33. Show the go get output: the "go: added" lines for the library and for cuelang.org/go v0.17.1. Check against: library/go.mod -->

## 2. Fetch the module

<!-- File edit: create main.go with k := kernel.New(kernel.WithRegistry(schema.PublicRegistry)) and mod, err := k.AcquireModuleFromRegistry(ctx, "opmodel.dev/modules/web_app@v1", "v1.0.4"), then print mod.Metadata.Name and mod.Metadata.Version. Command: go run . Output to show: web_app 1.0.4. At most one line on why: one Kernel serves the whole process and is safe to share between goroutines, and WithRegistry is the only registry setting every call uses. Link the concept page "Modules and instances". Check against: library/opm/kernel/kernel.go (New, WithRegistry), library/opm/kernel/acquire.go (AcquireModuleFromRegistry), library/opm/schema/metadata.go (ModuleMetadata) -->

## 3. Check your values

<!-- File edit: create values.cue holding replicas: 2 (web_app's #config gives every other field a default). In main.go: vals, err := k.LoadSourceFromFile("values.cue"), then _, err = k.ValidateConfigDetailed(mod.ConfigSchema(), []kernel.Source{vals}); on error print it with cueerrors.Print(os.Stderr, err, nil) from cuelang.org/go/cue/errors. Command: go run . Output to show: the step 2 line plus "values OK" (the program's own message). Optionally show that replicas: 0 fails with a position in values.cue (Verify: the exact error text before quoting it). Check against: library/opm/kernel/validate.go, library/opm/kernel/source_loader.go, modules/web_app/module.cue (#config) -->

## 4. Build an instance

<!-- In main.go: inst, err := k.SynthesizeInstance(ctx, kernel.InstanceInput{Module: mod, Name: "demo", Namespace: "default", Values: []kernel.Source{vals}}), then print inst.Metadata.Name, inst.Metadata.Namespace and inst.Metadata.UUID. Command: go run . Output to show: demo default <uuid>. At most one line on why: the render imports the instance as a CUE package, so it must come from SynthesizeInstance or AcquireInstanceFromDir, and synthesis checks the values again inside that build. Check against: library/opm/kernel/synth.go (InstanceInput, SynthesizeInstance), library/opm/schema/metadata.go (InstanceMetadata) -->

## 5. Generate a platform

<!-- In main.go: src, err := platformmodule.NewRegistry(platformmodule.RegistryConfig{Registry: schema.PublicRegistry, ClientType: "render-demo", Env: os.Environ()}); entries := []platformmodule.Entry{{Path: "opmodel.dev/catalogs/opm@v4", Version: "4.4.1", Enable: true}}; deps, err := platformmodule.Closure(ctx, src, platformmodule.Roots(entries)); files, err := platformmodule.Generate(platformmodule.Input{Name: "demo", Type: "kubernetes", ModulePath: "opmodel.dev/platforms/demo@v0", Entries: entries, Deps: deps}); files.WriteTo("platform"); then plat, err := k.AcquirePlatformFromDir(ctx, "platform") and print plat.Metadata.Name and plat.Metadata.Type. Command: go run . then cat platform/platform.cue. Output to show: demo kubernetes, and the generated file with its "Generated by opm/helper/platformmodule" header, core.#Platform, and the #registry entry for opmodel.dev/catalogs/opm@v4 carrying enable, version and #catalog. At most one line on why: a platform is a CUE module that imports its catalogs; the opmodel.dev/platforms/ path is reserved and never published. Pick a catalog version at or above the one the module pins (web_app pins 4.4.0), or the render reports version skew. Link the concept page "Platforms and catalogs". Check against: library/opm/helper/platformmodule/doc.go, library/opm/helper/platformmodule/generate.go (renderPlatformFile), library/opm/helper/platformmodule/build_test.go, library/opm/kernel/acquire.go (AcquirePlatformFromDir), modules/web_app/cue.mod/module.cue -->

## 6. Render the instance

<!-- In main.go: res, err := k.Render(ctx, kernel.RenderInput{Instance: inst, Platform: plat, RuntimeName: "render-demo"}), then for each c in res.Compiled print c.Component and c.Transformer. Command: go run . Output to show: two lines, web with .../transformers/deployment-transformer@4.4.1 and web with .../transformers/service-transformer@4.4.1 (Verify: the order, which is the build's pair order). At most one line on why: when a component cannot be rendered, Render returns a *kernel.RenderError whose Diagnostics names the cause, and errors.As reaches the typed error; link the diagnostics entries "No matching transformer" and "Unresolved demands". Check against: library/opm/kernel/render.go (RenderInput, Compiled, RenderError), library/opm/kernel/flow_integration_test.go (pairs subtest) -->

## 7. Print the objects as YAML

<!-- In main.go: for each c in res.Compiled, out, err := yaml.Encode(c.Value) from cuelang.org/go/encoding/yaml, printed with a --- separator between objects. Command: go run . Output to show: an excerpt of the Deployment (name demo-web, namespace default, replicas: 2, label app.kubernetes.io/managed-by: render-demo) and of the Service. Name the instance-prefixed name rule in one line with a link to the concept page "Identity and names". Verify: the rendered names and labels by running the program before quoting them. Check against: library/opm/kernel/render.go (Compiled.Value), core/src/transformer.cue (#TransformerContext controllerLabels), catalog_opm/docs/name-constraints.md -->

## What you built

<!-- Two or three sentences on the result: one Go program and one Kernel took a published module and a values file to Kubernetes objects in memory. Each Compiled value carries the instance, component and transformer that produced it; applying the objects, and wrapping them in platform-specific types, is the embedding program's job. -->

## Next steps

<!-- Three links at most. Concept: "How matching works". Diagnostics how-to: "Unresolved demands". Reference: the Go API documentation for github.com/open-platform-model/library/opm/kernel (Verify: where it is published, for example pkg.go.dev, before linking). -->
