package catalog

// ProvidesFold exposes the deprecated fallback to the package's external
// tests, so the parity test can compare it with what Provides decodes.
var ProvidesFold = (*Catalog).providesFold
