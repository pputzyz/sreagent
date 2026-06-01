package middleware

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS returns a CORS middleware configured for development and production.
// originsCSV is a comma-separated list of allowed origins (from config).
// When empty, defaults to localhost development origins (debug mode) or an
// empty list (release mode — origins must be explicitly configured).
func CORS(originsCSV string) gin.HandlerFunc {
	// Startup validation: reject wildcard with credentials
	if originsCSV != "" {
		for _, o := range strings.Split(originsCSV, ",") {
			trimmed := strings.TrimSpace(o)
			if trimmed == "*" {
				panic("CORS misconfiguration: AllowCredentials=true with wildcard origin is unsafe. Set explicit origins in CORS_ALLOWED_ORIGINS.")
			}
			// #11: Validate origin URL format
			if trimmed != "" {
				u, err := url.Parse(trimmed)
				if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
					panic(fmt.Sprintf("CORS misconfiguration: invalid origin %q — must have http/https scheme and valid host", trimmed))
				}
				if u.Path != "" && u.Path != "/" {
					panic(fmt.Sprintf("CORS misconfiguration: origin %q must not contain a path", trimmed))
				}
			}
		}
	}

	var allowedOrigins []string
	if originsCSV != "" {
		for _, o := range strings.Split(originsCSV, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	}
	if len(allowedOrigins) == 0 {
		if gin.Mode() == gin.ReleaseMode {
			// In release mode, CORS_ALLOWED_ORIGINS must be explicitly set.
			// Defaulting to localhost with AllowCredentials is a security risk.
			allowedOrigins = []string{}
		} else {
			allowedOrigins = []string{
				"http://localhost:5173",
				"http://localhost:8080",
				"http://127.0.0.1:5173",
				"http://127.0.0.1:8080",
			}
		}
	}

	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
