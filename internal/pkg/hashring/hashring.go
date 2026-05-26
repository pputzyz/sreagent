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
	keys     []uint32        // sorted virtual-node positions
	hashMap  map[uint32]string // virtual-node position -> physical node name
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

// RingManager manages per-key hash rings (similar to Nightingale's
// DatasourceHashRingType). In SREAgent the "key" is typically a
// datasource ID, so different datasources can have independent rings.
type RingManager struct {
	mu      sync.RWMutex
	rings   map[string]*Ring
	replicas int
}

// NewRingManager creates a ring manager with the given replica count.
func NewRingManager(replicas int) *RingManager {
	if replicas <= 0 {
		replicas = DefaultReplicas
	}
	return &RingManager{
		rings:    make(map[string]*Ring),
		replicas: replicas,
	}
}

// GetRing returns the ring for the given key, creating it if absent.
func (m *RingManager) GetRing(key string) *Ring {
	m.mu.RLock()
	r, ok := m.rings[key]
	m.mu.RUnlock()
	if ok {
		return r
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// double-check after acquiring write lock
	if r, ok = m.rings[key]; ok {
		return r
	}
	r = New(m.replicas)
	m.rings[key] = r
	return r
}

// SetRing replaces the ring for the given key.
func (m *RingManager) SetRing(key string, r *Ring) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rings[key] = r
}

// DeleteRing removes the ring for the given key.
func (m *RingManager) DeleteRing(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rings, key)
}

// RebuildRing rebuilds the ring for the given key with a new set of nodes.
func (m *RingManager) RebuildRing(key string, nodes []string) {
	r := New(m.replicas)
	for _, n := range nodes {
		r.Add(n)
	}
	m.SetRing(key, r)
}

// IsHit checks whether the given node is responsible for the given
// primary key within the ring identified by ringKey.
func (m *RingManager) IsHit(ringKey string, pk string, node string) bool {
	ring := m.GetRing(ringKey)
	if ring == nil {
		return false
	}
	return ring.IsHit(pk, node)
}

// RuleRingKey generates a standard ring key for an alert rule.
// It uses the rule ID as a string.
func RuleRingKey(ruleID uint) string {
	return strconv.FormatUint(uint64(ruleID), 10)
}

// DatasourceRingKey generates a standard ring key for a datasource.
func DatasourceRingKey(dsID uint) string {
	return strconv.FormatUint(uint64(dsID), 10)
}
