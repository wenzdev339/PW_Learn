package middleware

import (
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
)

func AmbientLatency(latencyMs, jitterMs int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if latencyMs == 0 && jitterMs == 0 {
			c.Next()
			return
		}
		jitter := 0
		if jitterMs > 0 {
			jitter = rand.Intn(jitterMs)
		}
		time.Sleep(time.Duration(latencyMs+jitter) * time.Millisecond)
		c.Next()
	}
}
