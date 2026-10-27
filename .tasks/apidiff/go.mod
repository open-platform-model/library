// Tools module for task api:diff (.tasks/api-diff.sh). It pins
// golang.org/x/exp/cmd/apidiff; the go.sum next to this file is the
// checksum the build verifies (-mod=readonly), so nothing unpinned runs.
// It is a separate module so the library's own go.mod, which every
// embedder resolves, never requires x/exp or x/tools.
//
// Pin: x/exp commit f0251eb (2026-10-14). The go directive here must not exceed the
// library's go.mod, so CI's setup-go (go-version-file: go.mod) builds it
// without a toolchain switch. Bumped by hand, with go get -tool
// golang.org/x/exp/cmd/apidiff@<commit> under GOWORK=off; Dependabot does
// not watch this module.
module opm-library-apidiff-tools

go 1.26.0

tool golang.org/x/exp/cmd/apidiff

require (
	golang.org/x/exp v0.0.0-20261014143641-f0251eb4aa3e // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
)
