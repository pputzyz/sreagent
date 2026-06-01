package handler

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
)

// NewMetricsHandler returns a gin.HandlerFunc that exposes app metrics in
// Prometheus exposition format. When metricsToken is non-empty, requests must
// include Authorization: Bearer <token>.
func NewMetricsHandler(metricsToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// #12: In release mode with no token configured, restrict to localhost only.
		if gin.Mode() == gin.ReleaseMode && metricsToken == "" {
			ip := c.ClientIP()
			if ip != "127.0.0.1" && ip != "::1" {
				Error(c, apperr.WithMessage(apperr.ErrForbidden, "metrics endpoint requires authentication in release mode"))
				return
			}
		}
		// If a metrics token is configured, require Bearer auth
		if metricsToken != "" {
			auth := c.GetHeader("Authorization")
			token := strings.TrimPrefix(auth, "Bearer ")
			if !strings.HasPrefix(auth, "Bearer ") || subtle.ConstantTimeCompare([]byte(token), []byte(metricsToken)) != 1 {
				Error(c, apperr.WithMessage(apperr.ErrUnauthorized, "unauthorized: invalid or missing metrics token"))
				return
			}
		}

		gatherer := prometheus.DefaultGatherer
		mfs, err := gatherer.Gather()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		contentType := expfmt.Negotiate(c.Request.Header)
		c.Header("Content-Type", string(contentType))

		enc := expfmt.NewEncoder(c.Writer, contentType)
		for _, mf := range mfs {
			if err := enc.Encode(mf); err != nil {
				return
			}
		}
	}
}
