package middleware

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS returns a CORS middleware configured for development and production.
// Set the CORS_ALLOWED_ORIGINS environment variable to a comma-separated list
// of allowed origins (e.g. "http://localhost:5173,https://sreagent.example.com").
// When unset, defaults to localhost development origins only.
func CORS() gin.HandlerFunc {
	originsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if originsEnv != "" {
		for _, o := range strings.Split(originsEnv, ",") {
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
