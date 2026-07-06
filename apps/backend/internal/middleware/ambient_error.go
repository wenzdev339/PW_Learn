package middleware

import (
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AmbientError(errorRate float64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if errorRate > 0 && rand.Float64() < errorRate {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "AMBIENT_SIMULATED_ERROR", "message": "Ambient simulated error"},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
