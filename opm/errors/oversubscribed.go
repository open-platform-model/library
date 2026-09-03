package errors

import (
	"fmt"
	"strings"
)

// OverSubscribedContractError is the render refusal for a contract key
// declared `fulfilment: "provider"` on a required demand of transformers from
// more than one of the platform's enabled registry entries (the
// single-provider guard, enhancement 0010 D32 as corrected by D37; enforced
// inside the render build since library-render-cutover). A platform must
// carry exactly one provider for such a key; two is a misconfigured
// platform, not an arbitration.
//
// It is carried on the kernel's render diagnostics as a row and joined into
// the fail-closed gate error, reachable via errors.As.
type OverSubscribedContractError struct {
	// Key is the provider-fulfilled contract key (a resource or trait FQN).
	Key string

	// Catalogs are the registry keys whose transformers require Key: the
	// catalog module paths (path@major) core binds each entry's embedded
	// catalog identity to. Sorted, so the refusal is deterministic.
	Catalogs []string
}

func (e OverSubscribedContractError) Error() string {
	quoted := make([]string, 0, len(e.Catalogs))
	for _, c := range e.Catalogs {
		quoted = append(quoted, fmt.Sprintf("%q", c))
	}
	return fmt.Sprintf(
		"contract %q declares fulfilment \"provider\" but is supplied by transformers from %d catalogs (%s): a platform must carry exactly one provider for it",
		e.Key, len(e.Catalogs), strings.Join(quoted, ", "))
}
