package inventory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
)

// ComputeDigest returns a deterministic SHA-256 digest of an inventory entry
// set, formatted as "sha256:<hex>". The digest is order-independent: entries are
// sorted by their identity tuple (Group, Kind, Namespace, Name, Component,
// Version) before a canonical JSON encoding is hashed. Two actors computing a
// digest over the same rendered inventory therefore obtain the same value,
// regardless of the order their entries were produced in.
func ComputeDigest(entries []InventoryEntry) string {
	if len(entries) == 0 {
		sum := sha256.Sum256(nil)
		return fmt.Sprintf("sha256:%x", sum)
	}

	sorted := make([]InventoryEntry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Group != sorted[j].Group {
			return sorted[i].Group < sorted[j].Group
		}
		if sorted[i].Kind != sorted[j].Kind {
			return sorted[i].Kind < sorted[j].Kind
		}
		if sorted[i].Namespace != sorted[j].Namespace {
			return sorted[i].Namespace < sorted[j].Namespace
		}
		if sorted[i].Name != sorted[j].Name {
			return sorted[i].Name < sorted[j].Name
		}
		if sorted[i].Component != sorted[j].Component {
			return sorted[i].Component < sorted[j].Component
		}
		return sorted[i].Version < sorted[j].Version
	})

	b, err := json.Marshal(sorted)
	if err != nil {
		// Unreachable: InventoryEntry is six plain string fields with no custom
		// json.Marshaler, so encoding cannot fail. A silent fallback to a
		// non-JSON encoding (e.g. fmt "%v") would emit a digest that no other
		// actor reproduces, silently breaking the cross-actor parity this digest
		// exists to guarantee. Treat a marshal failure as the invariant
		// violation it is rather than masking it (Constitution Principle I —
		// determinism; Fail Fast).
		panic("inventory: json.Marshal failed on []InventoryEntry (invariant violation): " + err.Error())
	}
	sum := sha256.Sum256(b)
	return fmt.Sprintf("sha256:%x", sum)
}
