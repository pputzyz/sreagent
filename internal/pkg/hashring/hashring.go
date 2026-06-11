// Package hashring implements a consistent hash ring for distributing
// alert rules across multiple engine instances. It is modelled after
// Nightingale's DatasourceHashRing but implemented without external
// dependencies so it can be used as a lightweight drop-in.
//
// Design choices:
//   - Virtual nodes (replicas) per physical node for better distribution.
//   - CRC32 as the hash function (fast, good distribution, stdlib).
//   - Sorted ring with binary-search lookup (O(log N)).
//   - Thread-safe via sync.RWMutex.
package hashring

import (
	"errors"
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

// ErrEmptyRing is returned when the ring has no nodes.
var ErrEmptyRing = errors.New("hashring: empty ring")

// DefaultReplicas is the default number of virtual nodes per physical node.
// Nightingale uses 500; we match that for equivalent distribution quality.
const DefaultReplicas = 500

// Ring is a consistent hash ring.
type Ring struct {
	mu       sync.RWMutex
	replicas int
	keys     []uint32            // sorted virtual-node positions
	hashMap  map[uint32]string   // virtual-node position -> physical node name
	nodes    map[string]struct{} // set of physical nodes
}

// New creates a Ring with the given number of virtual-node replicas.
// If replicas <= 0, DefaultReplicas is used.
func New(replicas int) *Ring {
	if replicas <= 0 {
		replicas = DefaultReplicas
	}
	return &Ring{
		replicas: replicas,
		hashMap:  make(map[uint32]string),
		nodes:    make(map[string]struct{}),
	}
}

// Add inserts a physical node into the ring.
// Duplicate adds are idempotent.
func (r *Ring) Add(node string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[node]; exists {
		return
	}
	r.nodes[node] = struct{}{}

	for i := 0; i < r.replicas; i++ {
		key := r.hashKey(node, i)
		r.keys = append(r.keys, key)
		r.hashMap[key] = node
	}
	sort.Slice(r.keys, func(i, j int) bool { return r.keys[i] < r.keys[j] })
}

// Remove deletes a physical node from the ring.
func (r *Ring) Remove(node string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[node]; !exists {
		return
	}
	delete(r.nodes, node)

	newKeys := r.keys[:0]
	for _, key := range r.keys {
		if r.hashMap[key] == node {
			delete(r.hashMap, key)
		} else {
			newKeys = append(newKeys, key)
		}
	}
	r.keys = newKeys
}

// GetNode returns the physical node responsible for the given key.
// Returns ErrEmptyRing if the ring has no nodes.
func (r *Ring) GetNode(key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.keys) == 0 {
		return "", ErrEmptyRing
	}

	h := hashString(key)
	idx := sort.Search(len(r.keys), func(i int) bool {
		return r.keys[i] >= h
	})
	if idx >= len(r.keys) {
		idx = 0
	}
	return r.hashMap[r.keys[idx]], nil
}

// IsHit returns true if the given node is responsible for the given key.
// Returns false if the ring is empty.
func (r *Ring) IsHit(key string, node string) bool {
	n, err := r.GetNode(key)
	if err != nil {
		return false
	}
	return n == node
}

// Members returns the list of physical nodes (unsorted).
func (r *Ring) Members() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	members := make([]string, 0, len(r.nodes))
	for n := range r.nodes {
		members = append(members, n)
	}
	return members
}

// Size returns the number of physical nodes in the ring.
func (r *Ring) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.nodes)
}

// hashKey generates a hash for the i-th virtual node of a physical node.
func (r *Ring) hashKey(node string, i int) uint32 {
	return hashString(fmt.Sprintf("%s#%d", node, i))
}

// hashString hashes a string to uint32 using CRC32.
func hashString(s string) uint32 {
	return crc32.ChecksumIEEE([]byte(s))
}

// RuleRingKey generates a standard ring key for an alert rule.
// It uses the rule ID as a string.
func RuleRingKey(ruleID uint) string {
	return strconv.FormatUint(uint64(ruleID), 10)
}
