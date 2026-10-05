package inventory

// StaleSet returns, in previous order, every previous entry that no current
// entry is the [SameObject] as: the objects the current render no longer
// produces. Component and version never count, so a component rename or an
// API version change leaves nothing stale (0012:D7). Duplicate previous
// entries are each returned. StaleSet returns a non-nil empty slice when
// nothing is stale, including when previous is empty, and never changes its
// inputs.
func StaleSet(previous, current []Entry) []Entry {
	kept := make(map[identity]struct{}, len(current))
	for _, c := range current {
		kept[identityOf(c)] = struct{}{}
	}
	stale := make([]Entry, 0)
	for _, p := range previous {
		if _, ok := kept[identityOf(p)]; !ok {
			stale = append(stale, p)
		}
	}
	return stale
}
