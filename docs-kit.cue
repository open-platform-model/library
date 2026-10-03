// The library's docs bundle (docs-kit docs/contracts.md C6, C15, C20): the Go
// API reference opm-docs generates from the exported packages under opm/,
// plus the authored pages under docs/site/. docs.yml checks it on every pull
// request and publishes edge from main; release.yml's publish-docs job
// publishes each release. Preview it with `task docs:bundle`.
bundles: library: {
	placement: {kind: "docs", root: "/docs/", owns: ["reference/go-api/"]}
	version: {from: "tag", prefix: "v"}
	sources: [{
		kind:   "go-api"
		module: "./"
		// The module's root package cannot be documented (C20): pages are
		// named by the package directory below root, so helper/objectset is
		// the page helper-objectset. opm/internal/ is never documented.
		root: "./opm"
		packages: ["./opm/..."]
		section:     "reference/go-api/"
		title:       "Go API"
		description: "Every exported package of the OPM library, from its doc comments."
	}, {
		// The authored pages ship in the same bundle (docs-kit DESIGN
		// decision 20). The library commits no generated pages, so nothing
		// is excluded.
		kind: "markdown", dir: "docs/site"
	}]
}
