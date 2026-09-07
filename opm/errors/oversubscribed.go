package errors

import (
	"fmt"
	"strings"
)

// OverSubscribedContract is one row of the single-provider guard: a contract
// key declared `fulfilment: "provider"` on a required demand of transformers
// from more than one of the platform's enabled registry entries (enhancement
// 0010 D32 as corrected by D37; enforced inside the render build since
// library-render-cutover). A platform must carry exactly one provider for
// such a key; two is a misconfigured platform, not an arbitration.
//
// It is data, not an error; [OverSubscribedContractsError] is the gate cause.
type OverSubscribedContract struct {
	// Key is the provider-fulfilled contract key (a resource or trait FQN).
	Key string

	// Catalogs are the registry keys whose transformers require Key: the
	// catalog module paths (path@major) core binds each entry's embedded
	// catalog identity to. Sorted, so the refusal is deterministic.
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

func (e *OverSubscribedContractsError) Error() string {
	msg := fmt.Sprintf("%d over-subscribed provider contract(s):", len(e.Contracts))
	for _, c := range e.Contracts {
		msg += "\n  " + c.describe()
	}
	return msg
}
