package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/config"
	"github.com/sreagent/sreagent/internal/middleware"
	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"
	"github.com/sreagent/sreagent/internal/service"
	"github.com/sreagent/sreagent/internal/testutil"
)

// testJWTSecret is a fixed secret for unit tests (not involving DB).
const testJWTSecret = "test-secret-key-for-unit-tests-32bytes!!"

// setupAuthService creates an AuthService wired to a real test database.
func setupAuthService(t *testing.T) (*service.AuthService, *gorm.DB) {
	db := testutil.TestDB(t)
	testutil.CleanupDB(t, db)

	userRepo := repository.NewUserRepository(db)
	jwtCfg := &config.JWTConfig{
		Secret: testJWTSecret,
		Expire: 3600,
		Issuer: "sreagent-test",
	}
	svc := service.NewAuthService(userRepo, jwtCfg, nil, testutil.TestLogger())
	return svc, db
}

// newAuthServiceNoDB creates an AuthService without a database connection.
// Used for tests that only exercise CheckLoginRateLimit / RefreshToken
// with synthetic tokens and mock fail stores.
func newAuthServiceNoDB() *service.AuthService {
	jwtCfg := &config.JWTConfig{
		Secret: testJWTSecret,
		Expire: 3600,
		Issuer: "sreagent-test",
	}
	return service.NewAuthService(nil, jwtCfg, nil, testutil.TestLogger())
}

// seedUserWithPassword creates a user with a real bcrypt-hashed password.
func seedUserWithPassword(t *testing.T, db *gorm.DB, username, password string, role model.Role, isActive bool) *model.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost) // MinCost for speed
	require.NoError(t, err)
	user := &model.User{
		Username: username,
		Password: string(hash),
		Role:     role,
		IsActive: true, // GORM default:true overrides false, so always create as active
	}
	require.NoError(t, db.Create(user).Error)
	// If we need inactive, update after creation
	if !isActive {
		require.NoError(t, db.Model(user).Update("is_active", false).Error)
		user.IsActive = false
	}
	return user
}

// generateTestToken creates a JWT token with custom claims for testing.
func generateTestToken(t *testing.T, userID uint, username, role, secret string, issuedAt time.Time) string {
	t.Helper()
	claims := middleware.Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			Issuer:    "sreagent-test",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

// ---------------------------------------------------------------------------
// Login tests (require SREAGENT_TEST_DSN)
// ---------------------------------------------------------------------------

// Test_Login_Success verifies that a valid username/password returns a token.
func Test_Login_Success(t *testing.T) {
	svc, db := setupAuthService(t)
	seedUserWithPassword(t, db, "admin1", "correct-password", model.RoleAdmin, true)

	token, expire, err := svc.Login(context.Background(), "admin1", "correct-password")
	require.NoError(t, err)
	assert.NotEmpty(t, token, "token should not be empty")
	assert.Equal(t, 3600, expire, "expire should match config")

	// Verify the token is parseable
	claims, err := middleware.ParseToken(token, svc.GetJWTSecret())
	require.NoError(t, err)
	assert.Equal(t, "admin1", claims.Username)
	assert.Equal(t, "admin", claims.Role)
}

// Test_Login_InvalidCredentials verifies that wrong password returns ErrInvalidCreds.
func Test_Login_InvalidCredentials(t *testing.T) {
	svc, db := setupAuthService(t)
	seedUserWithPassword(t, db, "user1", "real-password", model.RoleMember, true)

	_, _, err := svc.Login(context.Background(), "user1", "wrong-password")
	assert.Error(t, err, "should fail with wrong password")
	assert.Contains(t, err.Error(), "invalid credentials")
}

// Test_Login_NonexistentUser verifies that a non-existent username returns ErrInvalidCreds.
func Test_Login_NonexistentUser(t *testing.T) {
	svc, _ := setupAuthService(t)

	_, _, err := svc.Login(context.Background(), "ghost", "any-password")
	assert.Error(t, err, "should fail for non-existent user")
	assert.Contains(t, err.Error(), "invalid credentials")
}

// Test_Login_DisabledAccount verifies that a disabled user cannot log in.
func Test_Login_DisabledAccount(t *testing.T) {
	svc, db := setupAuthService(t)
	seedUserWithPassword(t, db, "disabled1", "correct-password", model.RoleMember, false)

	_, _, err := svc.Login(context.Background(), "disabled1", "correct-password")
	assert.Error(t, err, "should fail for disabled account")
	assert.Contains(t, err.Error(), "disabled")
}

// ---------------------------------------------------------------------------
// CheckLoginRateLimit tests (no DB required)
// ---------------------------------------------------------------------------

// mockFailStore implements LoginFailStore for testing rate limiting.
type mockFailStore struct {
	count int64
	err   error
}

func (m *mockFailStore) GetLoginFailCount(_ context.Context, _ string) (int64, error) {
	return m.count, m.err
}

func (m *mockFailStore) IncrLoginFail(_ context.Context, _ string, _ time.Duration) error { return nil }

func (m *mockFailStore) ClearLoginFail(_ context.Context, _ string) error { return nil }

// Test_CheckLoginRateLimit_Locked verifies that exceeding the failure limit
// returns an error.
func Test_CheckLoginRateLimit_Locked(t *testing.T) {
	svc := newAuthServiceNoDB()

	// Inject a mock fail store that reports 5 failures (the limit)
	svc.SetFailStore(&mockFailStore{count: service.LoginFailMax})

	err := svc.CheckLoginRateLimit(context.Background(), "user1")
	assert.Error(t, err, "should be locked when fail count >= LoginFailMax")
	assert.Contains(t, err.Error(), "temporarily locked")
}

// Test_CheckLoginRateLimit_NotLocked verifies that under the limit, no error.
func Test_CheckLoginRateLimit_NotLocked(t *testing.T) {
	svc := newAuthServiceNoDB()

	svc.SetFailStore(&mockFailStore{count: 3})

	err := svc.CheckLoginRateLimit(context.Background(), "user1")
	assert.NoError(t, err, "should not be locked when fail count < LoginFailMax")
}

// Test_CheckLoginRateLimit_NoStore verifies that when no fail store is set,
// rate limiting is skipped (graceful degradation).
func Test_CheckLoginRateLimit_NoStore(t *testing.T) {
	svc := newAuthServiceNoDB()
	// No SetFailStore call — failStore is nil

	err := svc.CheckLoginRateLimit(context.Background(), "user1")
	assert.NoError(t, err, "should pass when fail store is nil")
}

// Test_CheckLoginRateLimit_RedisError_Degrades verifies that a Redis error
// in the fail store is treated as "not locked" (graceful degradation).
func Test_CheckLoginRateLimit_RedisError_Degrades(t *testing.T) {
	svc := newAuthServiceNoDB()

	svc.SetFailStore(&mockFailStore{count: 0, err: assert.AnError})

	err := svc.CheckLoginRateLimit(context.Background(), "user1")
	assert.NoError(t, err, "should degrade gracefully on Redis error")
}

// ---------------------------------------------------------------------------
// RefreshToken tests (require SREAGENT_TEST_DSN for DB-dependent tests)
// ---------------------------------------------------------------------------

// Test_RefreshToken_Valid verifies that a recently-issued token can be refreshed.
func Test_RefreshToken_Valid(t *testing.T) {
	svc, db := setupAuthService(t)
	user := seedUserWithPassword(t, db, "refresh-user", "pass", model.RoleMember, true)

	// Generate a token issued just now (within the 7-day grace window)
	tokenStr := generateTestToken(t, user.ID, user.Username, string(user.Role), svc.GetJWTSecret(), time.Now())

	newToken, expire, err := svc.RefreshToken(context.Background(), tokenStr)
	require.NoError(t, err)
	assert.NotEmpty(t, newToken)
	assert.Equal(t, 3600, expire)

	// The new token should be valid
	claims, err := middleware.ParseToken(newToken, svc.GetJWTSecret())
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
}

// Test_RefreshToken_Expired verifies that a token issued more than 7 days ago
// is rejected for refresh.
func Test_RefreshToken_Expired(t *testing.T) {
	svc, db := setupAuthService(t)
	user := seedUserWithPassword(t, db, "old-user", "pass", model.RoleMember, true)

	// Generate a token issued 10 days ago (beyond the 7-day grace window)
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	tokenStr := generateTestToken(t, user.ID, user.Username, string(user.Role), svc.GetJWTSecret(), oldTime)

	_, _, err := svc.RefreshToken(context.Background(), tokenStr)
	assert.Error(t, err, "should reject token older than 7 days")
	assert.Contains(t, err.Error(), "too old to refresh")
}

// Test_RefreshToken_DisabledUser verifies that refresh fails if the user
// was disabled after the token was issued.
func Test_RefreshToken_DisabledUser(t *testing.T) {
	svc, db := setupAuthService(t)
	user := seedUserWithPassword(t, db, "will-disable", "pass", model.RoleMember, true)

	tokenStr := generateTestToken(t, user.ID, user.Username, string(user.Role), svc.GetJWTSecret(), time.Now())

	// Disable the user after token issuance
	require.NoError(t, db.Model(user).Update("is_active", false).Error)

	_, _, err := svc.RefreshToken(context.Background(), tokenStr)
	assert.Error(t, err, "should reject refresh for disabled user")
	assert.Contains(t, err.Error(), "disabled")
}

// Test_RefreshToken_InvalidSignature verifies that a token signed with
// a different secret is rejected (no DB needed).
func Test_RefreshToken_InvalidSignature(t *testing.T) {
	svc := newAuthServiceNoDB()

	// Sign with a different secret
	tokenStr := generateTestToken(t, 1, "user", "member", "wrong-secret-key-1234567890123456", time.Now())

	_, _, err := svc.RefreshToken(context.Background(), tokenStr)
	assert.Error(t, err, "should reject token with invalid signature")
}

// ---------------------------------------------------------------------------
// Token Revocation regression tests (the fix: RefreshToken now checks
// TokenRevocationChecker and TokenBlacklistChecker, matching JWTAuth middleware)
// ---------------------------------------------------------------------------

// mockRevocationChecker implements middleware.TokenRevocationChecker.
type mockRevocationChecker struct {
	revokedAt map[uint]time.Time
}

func (m *mockRevocationChecker) GetUserTokenRevokedAt(userID uint) time.Time {
	if t, ok := m.revokedAt[userID]; ok {
		return t
	}
	return time.Time{}
}

// mockBlacklistChecker implements middleware.TokenBlacklistChecker.
type mockBlacklistChecker struct {
	blacklisted map[string]bool
}

func (m *mockBlacklistChecker) IsTokenBlacklisted(_ context.Context, tokenID string) (bool, error) {
	if m.blacklisted != nil {
		return m.blacklisted[tokenID], nil
	}
	return false, nil
}

// Test_RefreshToken_RevokedTokenRejected verifies that a token whose user
// was revoked AFTER the token was issued is rejected for refresh.
// This is the regression test for the token revocation bypass fix.
func Test_RefreshToken_RevokedTokenRejected(t *testing.T) {
	svc, db := setupAuthService(t)
	user := seedUserWithPassword(t, db, "revoked-user", "pass", model.RoleMember, true)

	// Token issued 5 minutes ago (within the 30-min grace window)
	tokenStr := generateTestToken(t, user.ID, user.Username, string(user.Role), svc.GetJWTSecret(), time.Now().Add(-5*time.Minute))

	// Set up revocation checker: user was revoked 2 minutes ago (AFTER token issuance)
	origRevocation := middleware.TokenRevocationChecker
	middleware.TokenRevocationChecker = &mockRevocationChecker{
		revokedAt: map[uint]time.Time{
			user.ID: time.Now().Add(-2 * time.Minute),
		},
	}
	defer func() { middleware.TokenRevocationChecker = origRevocation }()

	_, _, err := svc.RefreshToken(context.Background(), tokenStr)
	assert.Error(t, err, "should reject refresh for revoked token")
	assert.Contains(t, err.Error(), "revoked", "error should mention revocation")
}

// Test_RefreshToken_BlacklistedTokenRejected verifies that a blacklisted
// token is rejected for refresh.
func Test_RefreshToken_BlacklistedTokenRejected(t *testing.T) {
	svc, db := setupAuthService(t)
	user := seedUserWithPassword(t, db, "blacklist-user", "pass", model.RoleMember, true)

	tokenStr := generateTestToken(t, user.ID, user.Username, string(user.Role), svc.GetJWTSecret(), time.Now())

	// Compute the token hash the same way auth.go does
	// We need to pre-compute it. Since we can't import the exact logic,
	// we'll set up the blacklist to match the token string hash.
	// auth.go does: hash := sha256.Sum256([]byte(tokenString)); tokenID := hex.EncodeToString(hash[:16])
	// For the test, we just need to ensure the checker is consulted.

	origBlacklist := middleware.TokenBlacklistChecker
	middleware.TokenBlacklistChecker = &mockBlacklistChecker{
		blacklisted: map[string]bool{
			// We'll use a wildcard approach: any token is blacklisted
			"": true,
		},
	}
	// Override to match all tokens
	middleware.TokenBlacklistChecker = &alwaysBlacklistedChecker{}
	defer func() { middleware.TokenBlacklistChecker = origBlacklist }()

	_, _, err := svc.RefreshToken(context.Background(), tokenStr)
	assert.Error(t, err, "should reject refresh for blacklisted token")
	assert.Contains(t, err.Error(), "revoked", "error should mention revocation")
}

// alwaysBlacklistedChecker returns true for all tokens.
type alwaysBlacklistedChecker struct{}

func (a *alwaysBlacklistedChecker) IsTokenBlacklisted(_ context.Context, _ string) (bool, error) {
	return true, nil
}

// Test_RefreshToken_RevocationBeforeTokenIssued verifies that a revocation
// timestamp BEFORE the token's issuance time does NOT block refresh.
func Test_RefreshToken_RevocationBeforeTokenIssued(t *testing.T) {
	svc, db := setupAuthService(t)
	user := seedUserWithPassword(t, db, "old-revocation", "pass", model.RoleMember, true)

	// Token issued 5 minutes ago (within grace window)
	tokenStr := generateTestToken(t, user.ID, user.Username, string(user.Role), svc.GetJWTSecret(), time.Now().Add(-5*time.Minute))

	// Revocation was 10 minutes ago (BEFORE token issuance) — should NOT block
	origRevocation := middleware.TokenRevocationChecker
	middleware.TokenRevocationChecker = &mockRevocationChecker{
		revokedAt: map[uint]time.Time{
			user.ID: time.Now().Add(-10 * time.Minute),
		},
	}
	defer func() { middleware.TokenRevocationChecker = origRevocation }()

	newToken, _, err := svc.RefreshToken(context.Background(), tokenStr)
	assert.NoError(t, err, "should allow refresh when revocation is before token issuance")
	assert.NotEmpty(t, newToken)
}
