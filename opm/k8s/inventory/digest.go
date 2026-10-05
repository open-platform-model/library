package inventory

import (
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"slices"
)

// inventoryTag is the first line of the bytes the inventory digest hashes. A
// change to the encoding is a new tag, and a new tag changes every value a
// frontend has stored.
const inventoryTag = "opm-inventory-v1\n"

// Digest returns the inventory digest of entries, "sha256:" followed by 64
// lowercase hex digits. It hashes this encoding, written field by field so
// that no serialiser, struct tag or Go release can move it (0012:D7):
//
//	"opm-inventory-v1\n"
//	for each entry, sorted by Group, Kind, Namespace, Name, Component, Version:
//	    for each of Group, Kind, Namespace, Name, Version, Component:
//	        the field's byte length as an 8-byte big-endian unsigned integer
//	        the field's bytes
//
// The digest depends only on the entries' field values, not on their input
// order. Entries are not de-duplicated. A nil or empty inventory hashes the
// tag line alone. Digest never changes its input.
func Digest(entries []Entry) string {
	sorted := slices.Clone(entries)
	slices.SortFunc(sorted, compareEntries)

	h := sha256.New()
	h.Write([]byte(inventoryTag))
	for _, e := range sorted {
		writeField(h, e.Group)
		writeField(h, e.Kind)
		writeField(h, e.Namespace)
		writeField(h, e.Name)
		writeField(h, e.Version)
		writeField(h, e.Component)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// compareEntries orders entries by Group, Kind, Namespace, Name, Component and
// Version, compared as byte strings. It uses every field, so it is a total
// order on distinct entries.
func compareEntries(a, b Entry) int {
	return cmp.Or(
		cmp.Compare(a.Group, b.Group),
		cmp.Compare(a.Kind, b.Kind),
		cmp.Compare(a.Namespace, b.Namespace),
		cmp.Compare(a.Name, b.Name),
		cmp.Compare(a.Component, b.Component),
		cmp.Compare(a.Version, b.Version),
	)
}

// writeField writes s's length as 8 big-endian bytes, then s.
func writeField(h hash.Hash, s string) {
	h.Write(binary.BigEndian.AppendUint64(nil, uint64(len(s))))
	h.Write([]byte(s))
}
