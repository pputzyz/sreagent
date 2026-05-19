package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// NewMetricsHandler returns a gin.HandlerFunc that exposes app metrics in
// Prometheus exposition format. When metricsToken is non-empty, requests must
// include Authorization: Bearer <token>.
func NewMetricsHandler(metricsToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If a metrics token is configured, require Bearer auth
		if metricsToken != "" {
			auth := c.GetHeader("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != metricsToken {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    40001,
					"message": "unauthorized: invalid or missing metrics token",
				})
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
