// Package labelmatch provides a unified label matching engine used by
// AlertChannel, NotifyRule, BizGroup, DispatchPolicy, InhibitionRule,
// and MuteRule routing.
package labelmatch

import (
	"regexp"
	"strings"
	"sync"
)

var (
	regexCache   sync.Map // map[string]*regexp.Regexp
	regexCacheMu sync.Mutex
)

// CompileRegex returns a compiled regex from cache or compiles and caches it.
// Exported for use by dispatch/noise_reducer/mute_rule services.
func CompileRegex(pattern string) (*regexp.Regexp, error) {
	return getOrCompileRegex(pattern)
}

// getOrCompileRegex returns a compiled regex from cache or compiles and caches it.
func getOrCompileRegex(pattern string) (*regexp.Regexp, error) {
	if re, ok := regexCache.Load(pattern); ok {
		if r, ok := re.(*regexp.Regexp); ok {
			return r, nil
		}
	}
	regexCacheMu.Lock()
	defer regexCacheMu.Unlock()
	// Double-check after acquiring lock
	if re, ok := regexCache.Load(pattern); ok {
		if r, ok := re.(*regexp.Regexp); ok {
			return r, nil
		}
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	regexCache.Store(pattern, re)
	return re, nil
}

// Match returns true if all entries in pattern match the corresponding values in target.
// Pattern values support four operator prefixes:
//   - (none): exact equality
//   - "!=": not equal
//   - "=~": regex match
//   - "!~": negated regex match
//
// An empty pattern map always returns true (wildcard).
func Match(target, pattern map[string]string) bool {
	for k, p := range pattern {
		tv := target[k]
		switch {
		case strings.HasPrefix(p, "!~"):
			re, err := getOrCompileRegex(p[2:])
			if err != nil || re.MatchString(tv) {
				return false
			}
		case strings.HasPrefix(p, "=~"):
			re, err := getOrCompileRegex(p[2:])
			if err != nil || !re.MatchString(tv) {
				return false
			}
		case strings.HasPrefix(p, "!="):
			if tv == p[2:] {
				return false
			}
		default:
			if tv != p {
				return false
			}
		}
	}
	return true
}

// MatchWithSourceID is like Match but also checks a datasource_id dimension.
// If patternDSID is nil, it acts as a wildcard (matches any datasource).
// If patternDSID is non-nil, it must equal targetDSID for a match.
// Label matching proceeds only if the datasource dimension matches.
func MatchWithSourceID(target map[string]string, targetDSID *uint, pattern map[string]string, patternDSID *uint) bool {
	// Datasource dimension check
	if patternDSID != nil {
		if targetDSID == nil || *patternDSID != *targetDSID {
			return false
		}
	}
	return Match(target, pattern)
}
