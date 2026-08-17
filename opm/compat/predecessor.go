package compat

import "github.com/Masterminds/semver/v3"

// HighestStable returns the highest published stable (non-pre-release)
// version. published is the registry's `v`-prefixed, SemVer-ascending list.
// Pre-release tags (e.g. v0.6.0-dev.*) are skipped so selection lands on the
// latest *released* build. If no stable version exists, the highest overall is
// returned so a pre-release-only path still resolves. Unparseable entries are
// skipped.
//
// This is the FLOAT selector — "give me the latest released build" — and it
// is deliberately NOT the compatibility gate's predecessor selection. An
// earlier revision of this comment claimed it was; 0011 D23 (amending D9)
// corrected that: the publish gate's predecessor is found by D9's literal
// rule — scan published versions strictly below the effective version, same
// major, prereleases included, newest first — implemented gate-side in the
// CLI, because a stable-preferring selector coincides with that rule only on
// a prerelease-only history and would miss breaks that prerelease pinners
// (0010 D14-blessed) can see. Selection here is pure — enumerating and
// fetching the candidate are the caller's. Moved verbatim from
// opm/materialize's since-deleted filterVersions path (0010 D14). Its first
// true caller is template resolution's version selection
// (cli-template-modules), which is why it stays.
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
