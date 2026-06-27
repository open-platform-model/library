package materialize

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// subscriptionFilter is the Go projection of a #SubscriptionFilter. Range is
// a SemVer constraint expression; Allow / Deny are bare-SemVer (#VersionType)
// version lists. A nil *subscriptionFilter (no filter authored) selects the
// highest published stable (non-pre-release) version; pre-releases require
// explicit opt-in via Allow or a pre-release-bearing Range.
type subscriptionFilter struct {
	Range string
	Allow []string
	Deny  []string
}

// isEmpty reports whether the filter constrains nothing (no range, no allow,
// no deny) — equivalent to having no filter at all.
func (f *subscriptionFilter) isEmpty() bool {
	return f == nil || (f.Range == "" && len(f.Allow) == 0 && len(f.Deny) == 0)
}

// filterVersions narrows the published version list to the survivor set per
// D10: `range` restricts the candidate set, `allow` force-includes specific
// published versions (even outside range), `deny` force-excludes versions.
//
// published is the registry's `v`-prefixed, SemVer-ascending list (the form
// [enumerateVersions] returns). Allow/Deny carry the bare-SemVer form;
// comparison normalizes the `v`-prefix via Masterminds/semver (D4), which
// tolerates a leading `v`. The result preserves the published ascending order
// and keeps the `v`-prefixed strings (the form [pullCatalog] consumes).
//
// With no filter (or an all-empty filter) the highest published *stable*
// version is selected — pre-releases are excluded so a leaked dev tag does not
// silently become "latest" (spec: "Enabled subscription with no filter"). A
// path that has published only pre-releases falls back to the highest of them.
// Pre-instances are otherwise reachable only by explicit opt-in: a filter.allow
// naming the exact version, or a filter.range whose constraint carries a
// pre-release identifier.
func filterVersions(published []string, f *subscriptionFilter) ([]string, error) {
	if len(published) == 0 {
		return nil, nil
	}
	if f.isEmpty() {
		return []string{highestStable(published)}, nil
	}

	selected := make(map[string]bool, len(published))

	// 1. range restricts the candidate set. Absent range with allow/deny
	// present bases the set on all published versions, then allow/deny adjust.
	if f.Range != "" {
		constraint, err := semver.NewConstraint(f.Range)
		if err != nil {
			return nil, fmt.Errorf("parsing filter.range %q: %w", f.Range, err)
		}
		for _, v := range published {
			sv, err := semver.NewVersion(v)
			if err != nil {
				continue
			}
			if constraint.Check(sv) {
				selected[v] = true
			}
		}
	} else {
		for _, v := range published {
			selected[v] = true
		}
	}

	// 2. allow force-includes published versions regardless of range.
	for _, a := range f.Allow {
		matches, err := matchingPublished(published, a, "filter.allow")
		if err != nil {
			return nil, err
		}
		for _, v := range matches {
			selected[v] = true
		}
	}

	// 3. deny force-excludes.
	for _, d := range f.Deny {
		matches, err := matchingPublished(published, d, "filter.deny")
		if err != nil {
			return nil, err
		}
		for _, v := range matches {
			delete(selected, v)
		}
	}

	out := make([]string, 0, len(selected))
	for _, v := range published {
		if selected[v] {
			out = append(out, v)
		}
	}
	return out, nil
}

// highestStable returns the highest published stable (non-pre-release) version.
// published is the registry's `v`-prefixed, SemVer-ascending list. Pre-release
// tags (e.g. v0.6.0-dev.*) are skipped so an unfiltered subscription resolves to
// the latest *released* catalog — matching the drift check's stable-only
// semantics. If no stable version exists, the highest overall is returned so a
// pre-release-only catalog still materializes.
func highestStable(published []string) string {
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

// matchingPublished returns the published versions SemVer-equal to want
// (normalizing the `v`-prefix on both sides). field names the filter field
// for error context. A version that does not parse is treated as
// non-matching; a malformed want is an error.
func matchingPublished(published []string, want, field string) ([]string, error) {
	wv, err := semver.NewVersion(want)
	if err != nil {
		return nil, fmt.Errorf("parsing %s %q: %w", field, want, err)
	}
	var out []string
	for _, v := range published {
		pv, err := semver.NewVersion(v)
		if err != nil {
			continue
		}
		if pv.Equal(wv) {
			out = append(out, v)
		}
	}
	return out, nil
}
