package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// StructuredLogging provides structured logging middleware
func StructuredLogging(logger *slog.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		requestID := ""
		if param.Keys != nil {
			if id, exists := param.Keys["request_id"]; exists {
				requestID = id.(string)
			}
		}

		// Skip logging for health check endpoint
		if param.Path == "/health" {
			return ""
		}

		logger.Info("HTTP Request",
			"request_id", requestID,
			"method", param.Method,
			"path", param.Path,
			"status", param.StatusCode,
			"latency_ms", param.Latency.Milliseconds(),
			"client_ip", param.ClientIP,
			"user_agent", param.Request.UserAgent(),
			"error", param.ErrorMessage,
		)

		return ""
	})
}