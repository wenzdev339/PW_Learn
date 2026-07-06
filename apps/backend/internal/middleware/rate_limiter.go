package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter is a process-wide token bucket: capacity and refill rate are
// both derived from limitPerMinute. rate.Limiter is safe for concurrent use.
func RateLimiter(limitPerMinute int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(float64(limitPerMinute)/60.0), limitPerMinute)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": "RATE_LIMITED", "message": "Too many requests"},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
