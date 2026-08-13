package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/platform"
	"github.com/open-platform-model/library/opm/schema"
)

// Key derives a stable cache key from a platform's #registry subtree — the
// input that fully determines materialization (Q2: registry-only). The key is
// invariant to field ordering and to enable defaulting (an explicit
// `enable: true` hashes the same as the default), so two
// semantically-identical registries map to the same key.
//
// The whole-spec alternative (hashing metadata too) was rejected: metadata
// does not affect the materialized transformer index, so keying on it would
// cause spurious cache misses on cosmetic edits.
func Key(p *platform.Platform) (string, error) {
	if p == nil {
		return "", fmt.Errorf("cache.Key: nil *platform.Platform")
	}
	reg := p.Package.LookupPath(schema.Registry)
	if !reg.Exists() {
		return "", fmt.Errorf("cache.Key: platform has no #registry")
	}

	it, err := reg.Fields()
	if err != nil {
		return "", fmt.Errorf("cache.Key: reading #registry: %w", err)
	}

	norm := map[string]normSub{}
	for it.Next() {
		sub := it.Value()
		ns := normSub{Enable: true}
		if en := sub.LookupPath(cue.ParsePath("enable")); en.Exists() {
			if b, e := en.Bool(); e == nil {
				ns.Enable = b
			}
		}
		if ver := sub.LookupPath(cue.ParsePath("version")); ver.Exists() {
			if s, e := ver.String(); e == nil {
				ns.Version = s
			}
		}
		norm[it.Selector().Unquoted()] = ns
	}

	// json.Marshal sorts map keys, so the byte form is canonical.
	b, err := json.Marshal(norm)
	if err != nil {
		return "", fmt.Errorf("cache.Key: encoding #registry: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// normSub is the canonical, JSON-stable projection of one #Subscription used
// for key derivation. Version is core v2's scalar build selector. The v1-era
// filter fields (range/allow/deny) were dropped with the subscription
// collapse (0010 D14): a v2 subscription could never populate them, so v2
// keys are byte-identical before and after the trim, and v1 platforms stopped
// loading at the core retarget.
type normSub struct {
	Enable  bool   `json:"enable"`
	Version string `json:"version,omitempty"`
}
