package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_ParseToken_valid_token_returns_claims verifies that a token generated
// by GenerateToken can be parsed back with the same secret, and all claims match.
func Test_ParseToken_valid_token_returns_claims(t *testing.T) {
	secret := "test-secret-key-for-jwt-unit-tests"
	userID := uint(42)
	username := "alice"
	role := "admin"
	expireSeconds := 3600

	tokenStr, err := GenerateToken(userID, username, role, secret, expireSeconds)
	require.NoError(t, err, "GenerateToken should succeed")
	require.NotEmpty(t, tokenStr)

	claims, err := ParseToken(tokenStr, secret)
	require.NoError(t, err, "ParseToken should succeed for a valid token")
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, "sreagent", claims.Issuer)
}

// Test_ParseToken_expired_token_returns_error verifies that an expired token
// is rejected by ParseToken.
func Test_ParseToken_expired_token_returns_error(t *testing.T) {
	secret := "test-secret-key-for-jwt-unit-tests"

	// Generate a token that expires immediately (0 seconds).
	tokenStr, err := GenerateToken(1, "bob", "member", secret, 0)
	require.NoError(t, err)

	// Give the clock a moment to tick past the expiry.
	time.Sleep(100 * time.Millisecond)

	_, err = ParseToken(tokenStr, secret)
	assert.Error(t, err, "ParseToken should reject an expired token")
}

// Test_ParseToken_wrong_secret_returns_error verifies that a token signed
// with one secret cannot be parsed with a different secret.
func Test_ParseToken_wrong_secret_returns_error(t *testing.T) {
	secretA := "correct-secret-key-1234567890"
	secretB := "wrong-secret-key-0987654321"

	tokenStr, err := GenerateToken(1, "charlie", "viewer", secretA, 3600)
	require.NoError(t, err)

	_, err = ParseToken(tokenStr, secretB)
	assert.Error(t, err, "ParseToken should reject a token signed with a different secret")
}

// Test_ParseToken_algorithm_confusion_rejected verifies that the keyFunc
// rejects tokens that do not use HMAC signing. This guards against the
// "algorithm confusion" attack where an attacker crafts a token with
// alg=none or an RSA method.
func Test_ParseToken_algorithm_confusion_rejected(t *testing.T) {
	t.Run("none_algorithm", func(t *testing.T) {
		// Craft a token with "none" algorithm by manually building it.
		token := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
			UserID:   1,
			Username: "attacker",
			Role:     "admin",
		})
		tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		_, err = ParseToken(tokenStr, "any-secret")
		assert.Error(t, err, "ParseToken must reject alg=none tokens")
	})

	t.Run("rsa_algorithm_with_hmac_secret", func(t *testing.T) {
		// Generate an RSA key pair.
		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		// Sign a token with RS256.
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{
			UserID:   1,
			Username: "attacker",
			Role:     "admin",
		})
		tokenStr, err := token.SignedString(rsaKey)
		require.NoError(t, err)

		// Attempt to parse with HMAC secret — should fail because keyFunc
		// rejects non-HMAC signing methods.
		_, err = ParseToken(tokenStr, "hmac-secret-key")
		assert.Error(t, err, "ParseToken must reject RS256 tokens when expecting HMAC")
	})
}
