package errors

import (
	"fmt"
	"strings"
)

// OverSubscribedContract is one row of the single-provider guard: a contract
// key declared `fulfilment: "provider"` on a required demand of transformers
// from two or more of the platform's enabled registry entries (path plus
// major: two majors of one catalog are two entries), whether or not an
// enabled catalog defines the key (0010:D32/D37). The render build enforces
// the guard. The count is core's #Platform.#contracts.providedBy, which the render glue
// reads and never recomputes, so the rows are exactly the keys the platform's
// contract inventory reports over-subscribed. A platform must carry exactly
// one provider for such a key; two is a misconfigured platform, not an
// arbitration. The refusal text is unchanged by where the count comes from.
//
// It is data, not an error; [OverSubscribedContractsError] is the gate cause.
type OverSubscribedContract struct {
	// Key is the provider-fulfilled contract key (a resource or trait FQN).
	Key string

	// Catalogs are the registry keys whose transformers require Key: the
	// catalog module paths (path@major) core binds each entry's embedded
	// catalog identity to, exactly #contracts.providedBy[Key]. Sorted, so
	// the refusal is deterministic.
	Catalogs []string
}

// describe renders one over-subscription row as a line of the aggregate's
// message.
func (c OverSubscribedContract) describe() string {
	quoted := make([]string, 0, len(c.Catalogs))
	for _, cat := range c.Catalogs {
		quoted = append(quoted, fmt.Sprintf("%q", cat))
	}
	return fmt.Sprintf(
		"contract %q declares fulfilment \"provider\" but is supplied by transformers from %d catalogs (%s): a platform must carry exactly one provider for it",
		c.Key, len(c.Catalogs), strings.Join(quoted, ", "))
}

// OverSubscribedContractsError aggregates every over-subscription row into
// the one typed cause the fail-closed gate joins. It carries the
// diagnostics' rows unchanged and wraps nothing.
type OverSubscribedContractsError struct {
	// Contracts is the over-subscription set, key-sorted.
	Contracts []OverSubscribedContract
}

// Error returns a count line, then one indented line per over-subscribed
// contract naming its key and the catalogs whose transformers supply it.
func (e *OverSubscribedContractsError) Error() string {
	msg := fmt.Sprintf("%d over-subscribed provider contract(s):", len(e.Contracts))
	for _, c := range e.Contracts {
		msg += "\n  " + c.describe()
	}
	return msg
}
