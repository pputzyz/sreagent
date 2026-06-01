package middleware

import (
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// requestIDSanitizer strips control characters and newlines from X-Request-ID.
var requestIDSanitizer = regexp.MustCompile(`[\x00-\x1f\x7f\r\n]`)

// RequestLogger returns a middleware that logs HTTP requests using zap.
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		// #4: Sanitize — strip control chars/newlines, limit to 64 chars
		requestID = requestIDSanitizer.ReplaceAllString(requestID, "")
		if len(requestID) > 64 {
			requestID = requestID[:64]
		}
		c.Set("request_id", requestID)
		c.Set("logger", logger.With(zap.String("request_id", requestID)))
		c.Header("X-Request-ID", requestID)

		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("body_size", c.Writer.Size()),
		}

		if statusCode >= 500 {
			logger.Error("Server error", fields...)
		} else if statusCode >= 400 {
			logger.Warn("Client error", fields...)
		} else {
			logger.Info("Request", fields...)
		}
	}
}
