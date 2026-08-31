// Package identity is the single source of this module's path and version
// (core #IdentityPackage, enhancements 0010 D38 / 0011 D12). It sits at the
// bottom of the module's import graph: no intra-module imports, no core
// import; validation is external (a publishing tool unifies this package
// against core's #IdentityPackage). The library's own tests never load it;
// it exists so the fixture is also vet-able by the opm CLI.
package identity

// ModulePath is the module's complete CUE module path, major suffix included,
// byte-identical to cue.mod's `module:` field.
ModulePath: "testing.opmodel.dev/modules/web_app@v1"

// Version is the module's bare SemVer; its major must agree with ModulePath's.
// Hand-managed: this fixture is not published, so a bump is an explicit edit
// here, kept in step with metadata.version in module.cue.
Version: "1.0.0"
