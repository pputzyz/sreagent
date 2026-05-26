package hashring

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Ring_Empty_GetNode_ReturnsError(t *testing.T) {
	r := New(100)
	_, err := r.GetNode("any-key")
	assert.ErrorIs(t, err, ErrEmptyRing)
}

func Test_Ring_Add_SingleNode(t *testing.T) {
	r := New(100)
	r.Add("node-1")

	node, err := r.GetNode("some-key")
	require.NoError(t, err)
	assert.Equal(t, "node-1", node)
}

func Test_Ring_Add_DuplicateIsIdempotent(t *testing.T) {
	r := New(100)
	r.Add("node-1")
	r.Add("node-1")

	assert.Equal(t, 1, r.Size())
}

func Test_Ring_GetNode_DistributesAcrossNodes(t *testing.T) {
	r := New(500)
	r.Add("node-a")
	r.Add("node-b")
	r.Add("node-c")

	hits := map[string]int{}
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("rule-%d", i)
		node, err := r.GetNode(key)
		require.NoError(t, err)
		hits[node]++
	}

	// Each node should get at least 10% of keys (rough sanity check)
	for _, count := range hits {
		assert.Greater(t, count, 500, "node received too few keys — distribution may be broken")
	}
	assert.Equal(t, 3, len(hits), "all 3 nodes should receive keys")
}

func Test_Ring_Remove_Node(t *testing.T) {
	r := New(200)
	r.Add("node-a")
	r.Add("node-b")
	r.Remove("node-a")

	assert.Equal(t, 1, r.Size())

	node, err := r.GetNode("any-key")
	require.NoError(t, err)
	assert.Equal(t, "node-b", node)
}

func Test_Ring_Remove_Nonexistent_IsNoop(t *testing.T) {
	r := New(100)
	r.Add("node-a")
	r.Remove("ghost")
	assert.Equal(t, 1, r.Size())
}

func Test_Ring_IsHit(t *testing.T) {
	r := New(200)
	r.Add("node-a")
	r.Add("node-b")

	key := "test-rule-42"
	node, err := r.GetNode(key)
	require.NoError(t, err)

	assert.True(t, r.IsHit(key, node))
	assert.False(t, r.IsHit(key, "other-node"))
}

func Test_Ring_IsHit_EmptyRing(t *testing.T) {
	r := New(100)
	assert.False(t, r.IsHit("key", "node"))
}

func Test_Ring_Members(t *testing.T) {
	r := New(100)
	r.Add("node-a")
	r.Add("node-b")
	r.Add("node-c")

	members := r.Members()
	assert.Len(t, members, 3)
}

func Test_Ring_Consistency(t *testing.T) {
	// Adding a new node should only remap ~1/N of keys
	r := New(500)
	r.Add("node-a")
	r.Add("node-b")

	// Record initial mapping
	initial := make(map[string]string)
	for i := 0; i < 5000; i++ {
		key := fmt.Sprintf("rule-%d", i)
		node, _ := r.GetNode(key)
		initial[key] = node
	}

	// Add a third node
	r.Add("node-c")

	remapped := 0
	for i := 0; i < 5000; i++ {
		key := fmt.Sprintf("rule-%d", i)
		node, _ := r.GetNode(key)
		if initial[key] != node {
			remapped++
		}
	}

	// With 3 nodes, ideally ~1/3 of keys should be remapped.
	// Accept up to 50% to account for hash variance.
	assert.Less(t, remapped, 2500, "too many keys remapped after adding a node")
	assert.Greater(t, remapped, 500, "too few keys remapped — ring may not be working")
}

func Test_DefaultReplicas(t *testing.T) {
	r := New(0)
	assert.Equal(t, DefaultReplicas, r.replicas)

	r2 := New(-1)
	assert.Equal(t, DefaultReplicas, r2.replicas)
}
