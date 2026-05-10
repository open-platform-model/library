@if(test)

// Negative bundle: each top-level `bad_<reason>` field violates exactly one
// type-regex from apis/core/v1alpha2/types.cue. The harness drives one
// schemaCase per field via the `inputPath` override (see
// pkg/api/v1alpha2/schema_fixture_test.go). All cases share the same
// fixture file because the regex assertions are tiny and live next to
// each other for readability.
//
// Each field unifies a single primitive type with a known-bad value; CUE
// bottoms with "does not match" or similar regex-related diagnostic.
package fixtures

import (
	core "opmodel.dev/core/v1alpha2@v1"
)

// #NameType: lowercase alphanumeric + hyphens, RFC 1123 DNS label
//   (apis/core/v1alpha2/types.cue:17)
bad_name_uppercase: core.#NameType & "BadName"
bad_name_leading_hyphen: core.#NameType & "-leading-hyphen"

// #FQNType: <path>/<name>@v<N>  (apis/core/v1alpha2/types.cue:42)
bad_fqn_no_version: core.#FQNType & "example.com/foo/bar"

// #ModuleFQNType: <path>/<name>:<semver>  (apis/core/v1alpha2/types.cue:29)
bad_module_fqn_no_semver: core.#ModuleFQNType & "example.com/foo/bar:not-semver"

// #MajorVersionType: ^v[0-9]+$  (apis/core/v1alpha2/types.cue:25)
bad_version_no_v_prefix: core.#MajorVersionType & "1"
