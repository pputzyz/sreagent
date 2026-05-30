package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sreagent/sreagent/internal/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockRevocationChecker implements TokenRevocationChecker for tests.
type mockRevocationChecker struct {
	revokedAt time.Time
}

func (m *mockRevocationChecker) GetUserTokenRevokedAt(userID uint) time.Time {
	return m.revokedAt
}

// setupJWTTestContext creates a gin test context with the given Authorization header.
func setupJWTTestContext(t *testing.T, authHeader string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/test", nil)
	if authHeader != "" {
		c.Request.Header.Set("Authorization", authHeader)
	}
	return c, w
}

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

// ---------------------------------------------------------------------------
// JWTAuth middleware tests
// ---------------------------------------------------------------------------

// Test_JWTAuth_valid_token_sets_context verifies that a valid JWT token
// passes through the middleware and sets user info in the gin context.
func Test_JWTAuth_valid_token_sets_context(t *testing.T) {
	secret := "test-jwt-auth-secret-key-32bytes!!"
	cfg := &config.JWTConfig{Secret: secret, Expire: 3600, Issuer: "test"}

	tokenStr, err := GenerateToken(42, "alice", "admin", secret, 3600)
	require.NoError(t, err)

	c, w := setupJWTTestContext(t, "Bearer "+tokenStr)

	handler := JWTAuth(cfg)
	handler(c)

	assert.False(t, c.IsAborted(), "valid token should not abort")
	assert.Equal(t, http.StatusOK, w.Code)

	uid, exists := c.Get(ContextKeyUserID)
	assert.True(t, exists)
	assert.Equal(t, uint(42), uid)

	username, exists := c.Get(ContextKeyUsername)
	assert.True(t, exists)
	assert.Equal(t, "alice", username)

	role, exists := c.Get(ContextKeyRole)
	assert.True(t, exists)
	assert.Equal(t, "admin", role)
}

// Test_JWTAuth_missing_header_returns_401 verifies that a request without
// an Authorization header is rejected with 401.
func Test_JWTAuth_missing_header_returns_401(t *testing.T) {
	cfg := &config.JWTConfig{Secret: "secret", Expire: 3600}

	c, w := setupJWTTestContext(t, "")

	handler := JWTAuth(cfg)
	handler(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing authorization header")
}

// Test_JWTAuth_invalid_format_returns_401 verifies that a malformed
// Authorization header is rejected.
func Test_JWTAuth_invalid_format_returns_401(t *testing.T) {
	cfg := &config.JWTConfig{Secret: "secret", Expire: 3600}

	tests := []struct {
		name   string
		header string
	}{
		{"no_bearer_prefix", "Token abc123"},
		{"no_space", "Bearerabc123"},
		{"bearer_only", "Bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := setupJWTTestContext(t, tt.header)
			handler := JWTAuth(cfg)
			handler(c)

			assert.True(t, c.IsAborted())
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// Test_JWTAuth_expired_token_returns_401 verifies that an expired token
// is rejected by the middleware.
func Test_JWTAuth_expired_token_returns_401(t *testing.T) {
	secret := "test-jwt-auth-secret-key-32bytes!!"
	cfg := &config.JWTConfig{Secret: secret, Expire: 3600}

	tokenStr, err := GenerateToken(1, "bob", "member", secret, 0)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	c, w := setupJWTTestContext(t, "Bearer "+tokenStr)
	handler := JWTAuth(cfg)
	handler(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired token")
}

// Test_JWTAuth_revoked_token_returns_401 verifies that a token issued
// before the user's revocation time is rejected.
func Test_JWTAuth_revoked_token_returns_401(t *testing.T) {
	secret := "test-jwt-auth-secret-key-32bytes!!"
	cfg := &config.JWTConfig{Secret: secret, Expire: 3600}

	tokenStr, err := GenerateToken(42, "alice", "admin", secret, 3600)
	require.NoError(t, err)

	originalChecker := TokenRevocationChecker
	defer func() { TokenRevocationChecker = originalChecker }()

	TokenRevocationChecker = &mockRevocationChecker{
		revokedAt: time.Now().Add(1 * time.Second),
	}

	c, w := setupJWTTestContext(t, "Bearer "+tokenStr)
	handler := JWTAuth(cfg)
	handler(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "token has been revoked")
}

// Test_JWTAuth_non_revoked_token_passes verifies that a token issued
// after the revocation time is accepted.
func Test_JWTAuth_non_revoked_token_passes(t *testing.T) {
	secret := "test-jwt-auth-secret-key-32bytes!!"
	cfg := &config.JWTConfig{Secret: secret, Expire: 3600}

	originalChecker := TokenRevocationChecker
	defer func() { TokenRevocationChecker = originalChecker }()

	TokenRevocationChecker = &mockRevocationChecker{
		revokedAt: time.Now().Add(-1 * time.Hour),
	}

	tokenStr, err := GenerateToken(42, "alice", "admin", secret, 3600)
	require.NoError(t, err)

	c, w := setupJWTTestContext(t, "Bearer "+tokenStr)
	handler := JWTAuth(cfg)
	handler(c)

	assert.False(t, c.IsAborted(), "token issued after revocation should pass")
	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_JWTAuth_nil_revocation_checker_skips_check verifies that when
// TokenRevocationChecker is nil, revocation checks are skipped entirely.
func Test_JWTAuth_nil_revocation_checker_skips_check(t *testing.T) {
	secret := "test-jwt-auth-secret-key-32bytes!!"
	cfg := &config.JWTConfig{Secret: secret, Expire: 3600}

	originalChecker := TokenRevocationChecker
	defer func() { TokenRevocationChecker = originalChecker }()
	TokenRevocationChecker = nil

	tokenStr, err := GenerateToken(42, "alice", "admin", secret, 3600)
	require.NoError(t, err)

	c, _ := setupJWTTestContext(t, "Bearer "+tokenStr)
	handler := JWTAuth(cfg)
	handler(c)

	assert.False(t, c.IsAborted(), "nil checker should skip revocation and allow")
}
