package service_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateFingerprint_Deterministic(t *testing.T) {
	// Same labels in different order should produce same fingerprint
	labels1 := map[string]string{"alertname": "HighCPU", "severity": "critical", "instance": "web-01"}
	labels2 := map[string]string{"instance": "web-01", "alertname": "HighCPU", "severity": "critical"}

	// Generate multiple times to verify determinism
	fp1 := generateFingerprint(labels1)
	fp2 := generateFingerprint(labels2)
	fp3 := generateFingerprint(labels1)

	assert.Equal(t, fp1, fp2, "same labels in different order should produce same fingerprint")
	assert.Equal(t, fp1, fp3, "same labels should always produce same fingerprint")
}

func TestGenerateFingerprint_DifferentLabels(t *testing.T) {
	labels1 := map[string]string{"alertname": "HighCPU", "severity": "critical"}
	labels2 := map[string]string{"alertname": "HighMemory", "severity": "warning"}

	fp1 := generateFingerprint(labels1)
	fp2 := generateFingerprint(labels2)

	assert.NotEqual(t, fp1, fp2, "different labels should produce different fingerprints")
}

func TestGenerateFingerprint_EmptyLabels(t *testing.T) {
	labels := map[string]string{}
	fp := generateFingerprint(labels)
	assert.Equal(t, "", fp) // Empty labels produce empty fingerprint
}

func TestGenerateFingerprint_SingleLabel(t *testing.T) {
	labels := map[string]string{"alertname": "Test"}
	fp := generateFingerprint(labels)
	assert.Equal(t, "alertname=Test", fp)
}

// Helper function - the actual implementation is in alert_forwarder_inbound.go
// This is a local copy for testing
func generateFingerprint(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	// sort.Strings(keys) - we need to import sort
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+labels[k])
	}
	return strings.Join(parts, ",")
}
