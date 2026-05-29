package fingerprint

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
)

// Compute generates a deterministic fingerprint from a label map.
// Labels are sorted by key before hashing to ensure consistency.
func Compute(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s,", k, labels[k])
	}
	hash := md5.Sum([]byte(b.String()))
	return fmt.Sprintf("%x", hash)
}
