// Tools module for task api:diff (.tasks/api-diff.sh). It pins
// golang.org/x/exp/cmd/apidiff; the go.sum next to this file is the
// checksum the build verifies (-mod=readonly), so nothing unpinned runs.
// It is a separate module so the library's own go.mod, which every
// embedder resolves, never requires x/exp or x/tools.
//
// Pin: the last x/exp commit (d6e0b57, 2026-08-20) before x/exp raised
// its go directive to 1.26.0. The go directive here must not exceed the
// library's go.mod, so CI's setup-go (go-version-file: go.mod) builds it
// without a toolchain switch. Bumped by hand, with go get -tool
// golang.org/x/exp/cmd/apidiff@<commit> under GOWORK=off; Dependabot does
// not watch this module.
module opm-library-apidiff-tools

go 1.25.0

tool golang.org/x/exp/cmd/apidiff

require (
	golang.org/x/exp v0.0.0-20260820122028-d6e0b57b1a69 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
)
