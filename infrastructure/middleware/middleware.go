package middleware

import (
	"net/http"

	"go-gin-gorm-example/infrastructure/httplib"
	"go-gin-gorm-example/infrastructure/limiter"

	"github.com/gin-gonic/gin"
)

func RateLimiterMiddleware(rateLimiter *limiter.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rateLimiter.Allow() {
			c.Next()
			return
		}
		httplib.SetErrorResponse(c, http.StatusTooManyRequests, "rate limit exceeded")
		c.Abort()
	}
}
