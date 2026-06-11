package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	mrand "math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/middleware"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	sredis "github.com/sreagent/sreagent/internal/pkg/redis"
	"github.com/sreagent/sreagent/internal/service"
)

type AuthHandler struct {
	svc     *service.AuthService
	userSvc *service.UserService
	ldapSvc *service.LDAPService // optional — nil when LDAP is not configured
	redis   *sredis.Client       // optional — nil when Redis is not configured
}

// SetUserService wires the user service for /me endpoints.
func (h *AuthHandler) SetUserService(svc *service.UserService) {
	h.userSvc = svc
}

// SetRedis injects the Redis client for captcha support.
func (h *AuthHandler) SetRedis(rc *sredis.Client) {
	h.redis = rc
}

// SetLDAPService injects the LDAP service for LDAP login fallback.
func (h *AuthHandler) SetLDAPService(svc *service.LDAPService) {
	h.ldapSvc = svc
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type LoginRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	CaptchaID  string `json:"captcha_id"`
	CaptchaVal string `json:"captcha"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	ctx := c.Request.Context()

	// --- #2: Mandatory captcha after 5 failed login attempts ---
	// Skip captcha in testing mode (SREAGENT_TESTING=true)
	isTesting := os.Getenv("SREAGENT_TESTING") == "true"
	if !isTesting && h.redis != nil {
		failCount, fcErr := h.redis.GetLoginFailCount(ctx, req.Username)
		if fcErr == nil && failCount >= 5 && req.CaptchaID == "" {
			Error(c, apperr.WithMessage(apperr.ErrForbidden, "captcha required after multiple failed login attempts"))
			return
		}
	}

	// --- Rate limit check (Redis-based, per username) ---
	if err := h.svc.CheckLoginRateLimit(ctx, req.Username); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrForbidden, err.Error()))
		return
	}

	// --- Captcha verification (mandatory when fail count >= 5) ---
	if req.CaptchaID != "" && h.redis != nil {
		expected, err := h.redis.GetCaptcha(ctx, req.CaptchaID)
		if err != nil {
			h.svc.RecordLoginFail(ctx, req.Username)
			Error(c, apperr.WithMessage(apperr.ErrInvalidCreds, "invalid or expired captcha"))
			return
		}
		// #17: Case-sensitive comparison for captcha
		if expected == "" || expected != req.CaptchaVal {
			h.svc.RecordLoginFail(ctx, req.Username)
			Error(c, apperr.WithMessage(apperr.ErrInvalidCreds, "incorrect captcha"))
			return
		}
		// #2: Delete captcha after successful verification (defense-in-depth;
		// GetCaptcha already uses GetDel internally)
		_ = h.redis.Del(ctx, sredis.CaptchaKey(req.CaptchaID))
	}

	// --- Actual login ---
	token, expiresIn, err := h.svc.Login(ctx, req.Username, req.Password)
	if err != nil {
		// Try LDAP login if local auth fails and LDAP is configured
		if h.ldapSvc != nil && h.ldapSvc.Enabled() {
			ldapToken, ldapExpires, ldapErr := h.ldapSvc.AuthenticateAndLogin(ctx, req.Username, req.Password, h.svc.GetJWTSecret(), h.svc.GetJWTExpire(ctx))
			if ldapErr == nil {
				// LDAP login successful
				h.svc.ClearLoginFailures(ctx, req.Username)
				Success(c, LoginResponse{
					Token:     ldapToken,
					ExpiresIn: ldapExpires,
				})
				return
			}
			// Both failed — log LDAP error and return local auth error
			if l, exists := c.Get("logger"); exists {
				if logger, ok := l.(*zap.Logger); ok {
					logger.Debug("LDAP login fallback failed",
						zap.String("username", req.Username),
						zap.Error(ldapErr),
					)
				}
			}
		}
		h.svc.RecordLoginFail(ctx, req.Username)
		Error(c, err)
		return
	}

	// --- Success: clear failure counter ---
	h.svc.ClearLoginFailures(ctx, req.Username)

	Success(c, LoginResponse{
		Token:     token,
		ExpiresIn: expiresIn,
	})
}

// Captcha generates a simple math captcha and returns the captcha ID + SVG image.
// GET /api/v1/auth/captcha
func (h *AuthHandler) Captcha(c *gin.Context) {
	if h.redis == nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, "captcha service unavailable"))
		return
	}

	expr, answer := generateMathCaptcha()
	captchaID := uuid.New().String()

	if err := h.redis.SetCaptcha(c.Request.Context(), captchaID, answer, 5*time.Minute); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrRedis, "failed to store captcha"))
		return
	}

	svg := renderCaptchaSVG(expr)

	Success(c, gin.H{
		"captcha_id": captchaID,
		"image":      "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg)),
	})
}

// --- Captcha helpers ---

// cryptoRandIntn returns a cryptographically secure random int in [0, n).
func cryptoRandIntn(n int) int {
	nBig := big.NewInt(int64(n))
	result, err := rand.Int(rand.Reader, nBig)
	if err != nil {
		// Fallback should never happen; rand.Reader uses OS CSPRNG.
		return 0
	}
	return int(result.Int64())
}

// generateMathCaptcha creates a simple arithmetic expression and returns
// (display string, answer string).  e.g. ("3 + 7", "10")
// Uses crypto/rand so the captcha answer is unpredictable.
func generateMathCaptcha() (string, string) {
	a := cryptoRandIntn(20) + 1
	b := cryptoRandIntn(20) + 1

	operators := []struct {
		symbol string
		fn     func(int, int) int
	}{
		{"+", func(x, y int) int { return x + y }},
		{"-", func(x, y int) int { return x - y }},
		{"*", func(x, y int) int { return x * y }},
	}
	op := operators[cryptoRandIntn(len(operators))]

	// Ensure subtraction does not produce negative results
	if op.symbol == "-" && a < b {
		a, b = b, a
	}

	expr := fmt.Sprintf("%d %s %d", a, op.symbol, b)
	answer := strconv.Itoa(op.fn(a, b))
	return expr, answer
}

// renderCaptchaSVG renders a math expression as a simple inline SVG image
// with colored noise lines. Returns the raw SVG string (caller base64-encodes).
func renderCaptchaSVG(expr string) string {
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	width := 150
	height := 50

	var svg strings.Builder
	fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`, width, height)

	// Background
	fmt.Fprintf(&svg, `<rect width="%d" height="%d" fill="#f0f0f0"/>`, width, height)

	// Noise lines
	for i := 0; i < 5; i++ {
		x1 := r.Intn(width)
		y1 := r.Intn(height)
		x2 := r.Intn(width)
		y2 := r.Intn(height)
		rc := r.Intn(150) + 50
		gc := r.Intn(150) + 50
		bc := r.Intn(150) + 50
		fmt.Fprintf(&svg, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="rgb(%d,%d,%d)" stroke-width="1"/>`,
			x1, y1, x2, y2, rc, gc, bc)
	}

	// Text characters with slight rotation
	startX := 10
	for i, ch := range expr {
		x := startX + i*14
		y := 32 + r.Intn(6) - 3
		angle := r.Intn(11) - 5 // -5..+5 degrees
		rc := r.Intn(100)
		gc := r.Intn(100)
		bc := r.Intn(100)
		fmt.Fprintf(&svg,
			`<text x="%d" y="%d" fill="rgb(%d,%d,%d)" font-size="24" font-family="monospace" transform="rotate(%d,%d,%d)">%s</text>`,
			x, y, rc, gc, bc, angle, x, y, string(ch))
	}

	svg.WriteString(`</svg>`)
	return svg.String()
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "user not authenticated"))
		return
	}
	user, err := h.svc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, user)
}

// UpdateMe updates the current user's own profile (display_name, email, phone, avatar).
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	if h.userSvc == nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, "user service not available"))
		return
	}
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "user not authenticated"))
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		Avatar      string `json:"avatar"` // base64 data URL or preset key
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	// M7: Validate avatar size unconditionally — base64 data URLs must not exceed 200 KB.
	// A 200 KB binary file encodes to ~272,000 base64 characters.
	if len(req.Avatar) > 272000 {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "avatar image must not exceed 200 KB"))
		return
	}

	if err := h.userSvc.UpdateProfile(c.Request.Context(), userID, req.DisplayName, req.Email, req.Phone, req.Avatar); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// Refresh issues a new JWT token given a valid or recently-expired token.
// POST /api/v1/auth/refresh  — no JWTAuth middleware (token may be expired).
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	token, expiresIn, err := h.svc.RefreshToken(c.Request.Context(), req.Token)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, LoginResponse{
		Token:     token,
		ExpiresIn: expiresIn,
	})
}

// BindLark saves the current user's Lark open_id for bot command identity mapping.
// PUT /me/lark-bind   body: {"lark_open_id": "ou_xxx"}
func (h *AuthHandler) BindLark(c *gin.Context) {
	if h.userSvc == nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, "user service not available"))
		return
	}
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "user not authenticated"))
		return
	}

	var req struct {
		LarkOpenID string `json:"lark_open_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.userSvc.BindLarkOpenID(c.Request.Context(), userID, req.LarkOpenID); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// ChangeMyPassword changes the current user's own password.
func (h *AuthHandler) ChangeMyPassword(c *gin.Context) {
	if h.userSvc == nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, "user service not available"))
		return
	}
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "user not authenticated"))
		return
	}

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	if err := h.userSvc.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

// Logout blacklists the current JWT token so it can no longer be used.
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	if h.redis == nil {
		Error(c, apperr.WithMessage(apperr.ErrInternal, "logout requires Redis (token blacklist unavailable)"))
		return
	}

	// Extract the raw token from the Authorization header.
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "missing authorization header"))
		return
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "invalid authorization format"))
		return
	}
	rawToken := parts[1]

	// Compute a stable token ID by hashing the raw JWT (each token is unique).
	hash := sha256.Sum256([]byte(rawToken))
	tokenID := hex.EncodeToString(hash[:16]) // 16 bytes = 32 hex chars, sufficient for uniqueness

	// Parse claims to get the remaining TTL for the blacklist entry.
	claims, err := middleware.ParseToken(rawToken, h.svc.GetJWTSecret())
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "invalid token"))
		return
	}

	// Blacklist until the token would have expired anyway.
	var ttl time.Duration
	if claims.ExpiresAt != nil {
		ttl = time.Until(claims.ExpiresAt.Time)
	}
	if ttl <= 0 {
		ttl = 1 * time.Minute // short-lived marker for already-expired tokens
	}

	if err := h.redis.BlacklistToken(c.Request.Context(), tokenID, ttl); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrRedis, "failed to blacklist token"))
		return
	}

	Success(c, nil)
}
