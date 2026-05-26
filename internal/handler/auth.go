package handler

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	sredis "github.com/sreagent/sreagent/internal/pkg/redis"
	"github.com/sreagent/sreagent/internal/service"
)

type AuthHandler struct {
	svc      *service.AuthService
	userSvc  *service.UserService
	ldapSvc  *service.LDAPService  // optional — nil when LDAP is not configured
	redis    *sredis.Client // optional — nil when Redis is not configured
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

	// --- Rate limit check (Redis-based, per username) ---
	if err := h.svc.CheckLoginRateLimit(ctx, req.Username); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrForbidden, err.Error()))
		return
	}

	// --- Captcha verification (optional — only if captcha_id is provided) ---
	if req.CaptchaID != "" && h.redis != nil {
		expected, err := h.redis.GetCaptcha(ctx, req.CaptchaID)
		if err != nil {
			h.svc.RecordLoginFail(ctx, req.Username)
			Error(c, apperr.WithMessage(apperr.ErrInvalidCreds, "invalid or expired captcha"))
			return
		}
		if expected == "" || !strings.EqualFold(expected, req.CaptchaVal) {
			h.svc.RecordLoginFail(ctx, req.Username)
			Error(c, apperr.WithMessage(apperr.ErrInvalidCreds, "incorrect captcha"))
			return
		}
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
		"image":      "data:image/svg+xml;base64," + svg,
	})
}

// --- Captcha helpers ---

// generateMathCaptcha creates a simple arithmetic expression and returns
// (display string, answer string).  e.g. ("3 + 7", "10")
func generateMathCaptcha() (string, string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	a := r.Intn(20) + 1
	b := r.Intn(20) + 1

	operators := []struct {
		symbol string
		fn     func(int, int) int
	}{
		{"+", func(x, y int) int { return x + y }},
		{"-", func(x, y int) int { return x - y }},
		{"*", func(x, y int) int { return x * y }},
	}
	op := operators[r.Intn(len(operators))]

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
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
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
