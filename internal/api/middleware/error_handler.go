package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"tiktok-whisper/internal/api/errors"
)

// ErrorHandler middleware handles errors consistently across the API
func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		requestID := c.GetString("request_id")

		var apiErr *errors.APIError

		switch err := recovered.(type) {
		case *errors.APIError:
			apiErr = err
			apiErr.RequestID = requestID
		case error:
			// Log the original error for debugging
			logger.Error("Internal server error",
				"error", err.Error(),
				"request_id", requestID,
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
			)

			// Return a generic internal error to the client
			apiErr = &errors.APIError{
				Kind:      errors.KindInternal,
				Message:   "Internal server error",
				RequestID: requestID,
			}
		default:
			// Handle panics that aren't errors
			logger.Error("Unknown panic occurred",
				"recovered", recovered,
				"request_id", requestID,
			)

			apiErr = &errors.APIError{
				Kind:      errors.KindInternal,
				Message:   "Internal server error",
				RequestID: requestID,
			}
		}

		// Set response headers
		c.Header("Content-Type", "application/json")

		// Return the error response
		c.AbortWithStatusJSON(apiErr.HTTPStatus(), apiErr)
	})
}

// HandleError is a helper function for handlers to return errors
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	if apiErr, ok := err.(*errors.APIError); ok {
		requestID := c.GetString("request_id")
		apiErr.RequestID = requestID
		c.Header("Content-Type", "application/json")
		c.AbortWithStatusJSON(apiErr.HTTPStatus(), apiErr)
		return
	}

	// If it's not an APIError, panic so the error middleware can handle it
	panic(err)
}