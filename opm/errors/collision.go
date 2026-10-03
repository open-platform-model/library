package errors

import (
	"fmt"
	"strings"
)

// ContractCollision is one row of the collision report: a contract key that
// the catalogs of two or more enabled registry entries list in their
// contract maps (two majors of one catalog sharing a key, say). Core folds
// only keys with exactly one enabled definer, so a colliding key is absent
// from the platform's definedBy, requiredBy and comparability report, and
// the platform is not routable. The render glue reads the rows off core's
// #Platform.#contracts.collisions and collidingEntries and never computes
// them, so they are exactly the platform inventory's CollidingEntries.
//
// It is data, not an error; [ContractCollisionsError] is the gate cause.
type ContractCollision struct {
	// Key is the colliding contract key (a resource, trait or blueprint
	// FQN).
	Key string `json:"key"`

	// Catalogs are the registry keys (path@major) of the enabled entries
	// whose catalogs list Key, exactly #contracts.collidingEntries[Key].
	// Sorted, so the refusal is deterministic.
	Catalogs []string `json:"catalogs"`
}

// describe renders one collision row as a line of the aggregate's message.
func (c ContractCollision) describe() string {
	quoted := make([]string, 0, len(c.Catalogs))
	for _, cat := range c.Catalogs {
		quoted = append(quoted, fmt.Sprintf("%q", cat))
	}
	return fmt.Sprintf(
		"contract %q is defined by %d enabled registry entries (%s); a contract key must have exactly one enabled definer until side-by-side catalog majors are supported, so disable all but one of these entries",
		c.Key, len(c.Catalogs), strings.Join(quoted, ", "))
}

// ContractCollisionsError aggregates every collision row into the one typed
// cause the fail-closed gate joins, first among the causes: every other row
// is read against an inventory the collision distorts. The refusal is
// platform-wide, whatever the instance and whatever the caller's skip
// switch says. It carries the diagnostics' rows unchanged and wraps
// nothing.
type ContractCollisionsError struct {
	// Contracts is the collision set, key-sorted.
	Contracts []ContractCollision
}

// Error returns a count line, then one indented line per colliding contract
// key naming the entries that define it.
func (e *ContractCollisionsError) Error() string {
	msg := fmt.Sprintf("%d colliding contract(s):", len(e.Contracts))
	for _, c := range e.Contracts {
		msg += "\n  " + c.describe()
	}
	return msg
}

// NotRoutableError is the gate's catch-all: the platform's contract
// inventory reads routable false, and the render decoded neither an
// over-subscription row nor a collision row to explain it. It fires on no
// platform any core release produces today; it keeps the kernel's decoded
// refusal in agreement with the render module's own gate (which reads
// routable) should a future core add a term to routable that no decoded row
// reports. It carries no rows and wraps nothing.
type NotRoutableError struct{}

// Error returns a fixed message: the platform is not routable and no
// over-subscribed or colliding contract explains why.
func (e *NotRoutableError) Error() string {
	return "platform is not routable: its contract inventory reads routable false and reports no over-subscribed or colliding contract to explain it"
}
