package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var flakyCounters sync.Map // map[string]int

func TestScenario(appEnv string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if appEnv == "production" {
			c.Next()
			return
		}
		scenario := c.GetHeader("X-Test-Scenario")
		if scenario == "" {
			c.Next()
			return
		}

		switch {
		case scenario == "slow":
			time.Sleep(3 * time.Second)
			c.Next()

		case scenario == "error":
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "SIMULATED_ERROR", "message": "Simulated server error"},
			})
			c.Abort()

		case scenario == "timeout":
			// Block until the client gives up or a generous ceiling passes,
			// simulating a hung request without leaking the goroutine forever.
			select {
			case <-time.After(time.Hour):
			case <-c.Request.Context().Done():
			}
			c.Abort()

		case scenario == "rate-limited":
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": "SIMULATED_RATE_LIMIT", "message": "Simulated rate limit"},
			})
			c.Abort()

		case strings.HasPrefix(scenario, "flaky:"):
			failCount, _ := strconv.Atoi(strings.TrimPrefix(scenario, "flaky:"))
			key := c.GetHeader("X-Test-Run-Id") + ":" + c.Request.URL.Path
			value, _ := flakyCounters.LoadOrStore(key, 0)
			attempts := value.(int)
			flakyCounters.Store(key, attempts+1)
			if attempts < failCount {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{"code": "SIMULATED_FLAKY_ERROR", "message": "Simulated failure"},
				})
				c.Abort()
				return
			}
			flakyCounters.Delete(key)
			c.Next()

		default:
			c.Next()
		}
	}
}
