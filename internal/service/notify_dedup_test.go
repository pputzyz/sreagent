package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildNotifyDedupKey(t *testing.T) {
	key := BuildNotifyDedupKey(42, 7, "abc123", "firing")
	assert.Equal(t, "42:7:abc123:firing", key)
}

func TestBuildNotifyDedupKey_DifferentStatus(t *testing.T) {
	firing := BuildNotifyDedupKey(42, 7, "abc123", "firing")
	resolved := BuildNotifyDedupKey(42, 7, "abc123", "resolved")
	assert.NotEqual(t, firing, resolved)
}
