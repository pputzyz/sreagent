package labelmatch

import (
	"testing"
)

func Test_Match_ExactEquality(t *testing.T) {
	target := map[string]string{"env": "prod", "team": "infra"}
	pattern := map[string]string{"env": "prod"}
	if !Match(target, pattern) {
		t.Error("expected match for exact equality")
	}
}

func Test_Match_ExactMismatch(t *testing.T) {
	target := map[string]string{"env": "staging"}
	pattern := map[string]string{"env": "prod"}
	if Match(target, pattern) {
		t.Error("expected no match for mismatched value")
	}
}

func Test_Match_NotEqual(t *testing.T) {
	target := map[string]string{"env": "prod"}
	pattern := map[string]string{"env": "!=staging"}
	if !Match(target, pattern) {
		t.Error("expected match: prod != staging")
	}

	pattern2 := map[string]string{"env": "!=prod"}
	if Match(target, pattern2) {
		t.Error("expected no match: prod == prod")
	}
}

func Test_Match_RegexMatch(t *testing.T) {
	target := map[string]string{"instance": "web-123"}
	pattern := map[string]string{"instance": "=~web-\\d+"}
	if !Match(target, pattern) {
		t.Error("expected regex match")
	}
}

func Test_Match_RegexNoMatch(t *testing.T) {
	target := map[string]string{"instance": "db-123"}
	pattern := map[string]string{"instance": "=~web-\\d+"}
	if Match(target, pattern) {
		t.Error("expected no regex match")
	}
}

func Test_Match_NegatedRegex(t *testing.T) {
	target := map[string]string{"instance": "db-123"}
	pattern := map[string]string{"instance": "!~web-\\d+"}
	if !Match(target, pattern) {
		t.Error("expected negated regex match")
	}
}

func Test_Match_EmptyPattern(t *testing.T) {
	target := map[string]string{"env": "prod"}
	pattern := map[string]string{}
	if !Match(target, pattern) {
		t.Error("empty pattern should always match (wildcard)")
	}
}

func Test_Match_MultipleConditions(t *testing.T) {
	target := map[string]string{"env": "prod", "team": "infra", "region": "us-east-1"}
	pattern := map[string]string{"env": "prod", "team": "infra"}
	if !Match(target, pattern) {
		t.Error("expected match when all conditions satisfied")
	}
}

func Test_Match_MultipleConditions_OneFails(t *testing.T) {
	target := map[string]string{"env": "prod", "team": "platform"}
	pattern := map[string]string{"env": "prod", "team": "infra"}
	if Match(target, pattern) {
		t.Error("expected no match when one condition fails")
	}
}

func Test_Match_MissingKey(t *testing.T) {
	target := map[string]string{"env": "prod"}
	pattern := map[string]string{"missing_key": "value"}
	if Match(target, pattern) {
		t.Error("expected no match when key is missing from target")
	}
}

// --- MatchWithSourceID tests ---

func Test_MatchWithSourceID_NilPatternDSID_Wildcard(t *testing.T) {
	target := map[string]string{"env": "prod"}
	var patternDSID *uint // nil = wildcard
	if !MatchWithSourceID(target, uintPtr(1), target, patternDSID) {
		t.Error("nil patternDSID should match any datasource")
	}
}

func Test_MatchWithSourceID_SameDSID(t *testing.T) {
	target := map[string]string{"env": "prod"}
	patternDSID := uintPtr(5)
	if !MatchWithSourceID(target, uintPtr(5), target, patternDSID) {
		t.Error("expected match when datasource IDs are equal")
	}
}

func Test_MatchWithSourceID_DifferentDSID(t *testing.T) {
	target := map[string]string{"env": "prod"}
	patternDSID := uintPtr(5)
	if MatchWithSourceID(target, uintPtr(3), target, patternDSID) {
		t.Error("expected no match when datasource IDs differ")
	}
}

func Test_MatchWithSourceID_NilTargetDSID_NonNilPattern(t *testing.T) {
	target := map[string]string{"env": "prod"}
	patternDSID := uintPtr(5)
	if MatchWithSourceID(target, nil, target, patternDSID) {
		t.Error("expected no match when target DSID is nil but pattern requires specific DSID")
	}
}

func Test_MatchWithSourceID_LabelsAndDSID(t *testing.T) {
	target := map[string]string{"env": "prod"}
	pattern := map[string]string{"env": "=~prod|staging"}
	patternDSID := uintPtr(2)
	// Both label and DSID match
	if !MatchWithSourceID(target, uintPtr(2), pattern, patternDSID) {
		t.Error("expected match when both labels and DSID match")
	}
	// Labels match but DSID differs
	if MatchWithSourceID(target, uintPtr(99), pattern, patternDSID) {
		t.Error("expected no match when DSID differs even if labels match")
	}
}

func uintPtr(v uint) *uint { return &v }

// --- CompileRegex tests ---

func Test_CompileRegex_CacheHit(t *testing.T) {
	re1, err := CompileRegex(`test-\d+`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	re2, err := CompileRegex(`test-\d+`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if re1 != re2 {
		t.Error("expected same cached regex instance")
	}
}

func Test_CompileRegex_InvalidPattern(t *testing.T) {
	_, err := CompileRegex(`[invalid`)
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}
