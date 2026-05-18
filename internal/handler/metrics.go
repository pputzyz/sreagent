package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// metricsToken is loaded once at startup from the METRICS_TOKEN env var.
// If empty, the /metrics endpoint is open (backwards compat).
var metricsToken = os.Getenv("METRICS_TOKEN")

// MetricsHandler exposes app metrics in Prometheus exposition format.
// Uses the default gatherer (which includes Go runtime metrics via promhttp's init).
// When METRICS_TOKEN is set, requests must include Authorization: Bearer <token>.
func MetricsHandler(c *gin.Context) {
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
