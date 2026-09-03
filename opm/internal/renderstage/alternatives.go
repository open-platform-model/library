package renderstage

import (
	"sort"
	"strings"

	"github.com/open-platform-model/library/opm/compat"
)

// Alternatives returns every contract key in universe other than missing that
// shares its base (the FQN before the final "@"), in contract-key order (the
// kube-aware apiVersion ladder, 0010 D34/D4): "a transformer exists for a
// different version of this primitive".
func Alternatives(universe []string, missing string) []string {
	base := fqnBase(missing)
	var alts []string
	for _, key := range universe {
		if key != missing && fqnBase(key) == base {
			alts = append(alts, key)
		}
	}
	sort.Slice(alts, func(i, j int) bool {
		if c := compat.CompareAPIVersions(fqnVersion(alts[i]), fqnVersion(alts[j])); c != 0 {
			return c < 0
		}
		return alts[i] < alts[j]
	})
	return alts
}

func fqnBase(fqn string) string {
	if i := strings.LastIndex(fqn, "@"); i >= 0 {
		return fqn[:i]
	}
	return fqn
}

func fqnVersion(fqn string) string {
	if i := strings.LastIndex(fqn, "@"); i >= 0 {
		return fqn[i+1:]
	}
	return ""
}
