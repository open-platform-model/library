// Package cache provides opt-in memoization for [materialize.Materialize].
//
// The kernel deliberately holds no materialize cache (Principle I — the
// kernel stays stateless, and invalidation policy differs per consumer: the
// operator keys on a CR generation, the CLI opts out and relies on CUE's
// on-disk module cache). Consumers that want memoization construct a
// [MaterializeCache] themselves, derive a key with [Key], and wrap their
// Materialize calls.
//
// Unlike schema.Cache (a single-value sync.Once memo of the core schema),
// this is a multi-entry keyed cache — a different shape, hence a separate
// type.
package cache

import (
	"container/list"
	"sync"

	"github.com/open-platform-model/library/opm/materialize"
)

// MaterializeCache memoizes materialized platforms by a caller-derived key
// (see [Key]). Implementations MUST be safe for concurrent use.
type MaterializeCache interface {
	// Get returns the cached platform for key and whether it was present.
	Get(key string) (*materialize.MaterializedPlatform, bool)
	// Put stores mp under key, evicting per the implementation's policy.
	Put(key string, mp *materialize.MaterializedPlatform)
}

// LRU is the reference [MaterializeCache]: a fixed-capacity, least-recently-
// used cache safe for concurrent use. A non-positive capacity disables
// caching (Put is a no-op, Get always misses).
type LRU struct {
	mu       sync.Mutex
	capacity int
	ll       *list.List               // front = most recently used
	items    map[string]*list.Element // key → *list.Element holding *lruEntry
}

type lruEntry struct {
	key string
	val *materialize.MaterializedPlatform
}

// NewLRU returns an [LRU] holding at most capacity entries. A non-positive
// capacity yields a cache that never stores anything.
func NewLRU(capacity int) *LRU {
	return &LRU{
		capacity: capacity,
		ll:       list.New(),
		items:    make(map[string]*list.Element),
	}
}

// Get implements [MaterializeCache].
func (c *LRU) Get(key string) (*materialize.MaterializedPlatform, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.ll.MoveToFront(el)
	return el.Value.(*lruEntry).val, true
}

// Put implements [MaterializeCache].
func (c *LRU) Put(key string, mp *materialize.MaterializedPlatform) {
	if c.capacity <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		el.Value.(*lruEntry).val = mp
		return
	}
	el := c.ll.PushFront(&lruEntry{key: key, val: mp})
	c.items[key] = el
	if c.ll.Len() > c.capacity {
		oldest := c.ll.Back()
		if oldest != nil {
			c.ll.Remove(oldest)
			delete(c.items, oldest.Value.(*lruEntry).key)
		}
	}
}

var _ MaterializeCache = (*LRU)(nil)
