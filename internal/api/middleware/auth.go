package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuthMiddleware handles authentication for protected routes
type AuthMiddleware struct {
	logger *zap.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware() *AuthMiddleware {
	logger, _ := zap.NewProduction()
	return &AuthMiddleware{
		logger: logger,
	}
}

// AuthRequired is a middleware that validates authentication
func (m *AuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid authorization header format. Expected: Bearer <token>",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token (in production, this would be JWT validation)
		// For now, we'll accept any non-empty token for demo purposes
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid token",
			})
			c.Abort()
			return
		}

		// Extract user information from token
		// In production, this would parse JWT claims
		userID := m.extractUserID(token)
		if userID == "" {
			// Default to anonymous for testing
			userID = "anonymous"
		}

		// Set user information in context
		c.Set("user_id", userID)
		c.Set("authenticated", true)

		m.logger.Info("Request authenticated", 
			zap.String("user_id", userID),
			zap.String("path", c.Request.URL.Path))

		c.Next()
	}
}

// OptionalAuth is a middleware that tries to authenticate but doesn't require it
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token := parts[1]
				userID := m.extractUserID(token)
				if userID != "" {
					c.Set("user_id", userID)
					c.Set("authenticated", true)
					c.Next()
					return
				}
			}
		}

		// No valid auth header, continue as anonymous
		c.Set("user_id", "anonymous")
		c.Set("authenticated", false)
		c.Next()
	}
}

// APIKeyAuth is a middleware that validates API keys
func (m *AuthMiddleware) APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check API key in header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// Check query parameter
			apiKey = c.Query("api_key")
		}

		// Validate API key (in production, check against database)
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "API key is required",
			})
			c.Abort()
			return
		}

		// Set API key info in context
		c.Set("api_key", apiKey)
		c.Set("authenticated", true)

		c.Next()
	}
}

// extractUserID extracts user ID from token
// In production, this would parse JWT claims
func (m *AuthMiddleware) extractUserID(token string) string {
	// Simple demo implementation
	// In production, parse JWT and extract claims
	if strings.HasPrefix(token, "user_") {
		return strings.TrimPrefix(token, "user_")
	}
	
	// For demo purposes, accept any token and return a default user ID
	return "demo_user"
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-API-Key")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// SecurityHeaders adds security-related headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Remove server info
		c.Header("Server", "")

		c.Next()
	}
}

// RateLimitMiddleware provides basic rate limiting
// In production, use a proper rate limiting library like github.com/ulule/limiter
func RateLimitMiddleware() gin.HandlerFunc {
	// Simple in-memory rate limiter for demo
	// In production, use Redis or other distributed store
	requests := make(map[string]int)
	
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		// Increment request count
		requests[clientIP]++
		
		// Simple rate limit: 100 requests per minute per IP
		if requests[clientIP] > 100 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}