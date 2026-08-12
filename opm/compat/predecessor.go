package compat

import "github.com/Masterminds/semver/v3"

// HighestStable returns the highest published stable (non-pre-release)
// version. published is the registry's `v`-prefixed, SemVer-ascending list.
// Pre-release tags (e.g. v0.6.0-dev.*) are skipped so selection lands on the
// latest *released* build. If no stable version exists, the highest overall is
// returned so a pre-release-only path still resolves. Unparseable entries are
// skipped.
//
// This is 0011 D9's predecessor selection: the publish gate compares each
// gated primitive against the last published build carrying it, and this
// picks which build that is. Selection is pure — enumerating and fetching the
// candidate are the caller's. Moved verbatim from opm/materialize's
// since-deleted filterVersions path (0010 D14): subscription resolution now
// reads the authored version! scalar and performs no selection, so the
// publish gate is this function's only consumer.
func HighestStable(published []string) string {
	for i := len(published) - 1; i >= 0; i-- {
		sv, err := semver.NewVersion(published[i])
		if err != nil {
			continue
		}
		if sv.Prerelease() == "" {
			return published[i]
		}
	}
	return published[len(published)-1]
}
