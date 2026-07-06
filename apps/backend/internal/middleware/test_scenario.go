package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// flakyCounters นับว่า request ของ "flaky:N" scenario ถูกยิงมากี่ครั้งแล้ว
// ต่อ 1 คีย์ (runId+path) เก็บเป็น sync.Map เพราะ Gin จัดการหลาย request
// พร้อมกันในหลาย goroutine ห้ามใช้ map ธรรมดาเพราะจะ race กัน
var flakyCounters sync.Map // map[string]int

// TestScenario คือหัวใจของ backend นี้: มันทำให้เราสั่ง "จำลองปัญหาเครือข่าย"
// ได้แบบเจาะจงและ reproducible 100% ผ่าน header X-Test-Scenario แทนที่จะ
// ต้องรอให้ปัญหาจริงเกิดขึ้นเอง (ซึ่งมักทำให้เทสไม่เสถียร/flaky)
//
// วิธีใช้จาก Playwright: ส่ง header "X-Test-Scenario" แนบไปกับ request แล้ว
// backend จะตอบกลับตามพฤติกรรมที่สั่งทันที เช่น เทสว่า UI แสดง error banner
// ถูกไหมเมื่อ API ล่ม โดยไม่ต้องพึ่งดวงว่า API จะล่มจริงตอนไหน
func TestScenario(appEnv string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ปิดการจำลองทั้งหมดถ้ารันจริง (production) กัน header หลุดเข้ามา
		// ป่วน user จริงโดยไม่ตั้งใจ
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
			// จำลอง API ตอบช้า — ใช้เทส loading spinner/skeleton ของฝั่ง UI
			time.Sleep(3 * time.Second)
			c.Next()

		case scenario == "error":
			// จำลอง server พัง — ใช้เทสว่า UI แสดง error message ถูกไหม
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
			// จำลองโดน rate limit — ใช้เทส UI ที่ต้องจัดการ 429 (เช่น retry
			// พร้อม backoff)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": "SIMULATED_RATE_LIMIT", "message": "Simulated rate limit"},
			})
			c.Abort()

		case strings.HasPrefix(scenario, "flaky:"):
			// "flaky:2" แปลว่า: 2 ครั้งแรกที่ยิง path นี้ (ด้วย runId เดียวกัน)
			// ให้ fail แล้วครั้งที่ 3 เป็นต้นไปให้ผ่าน — ใช้พิสูจน์ว่า retry
			// logic ฝั่ง client/UI ทำงานถูกต้องจริง แบบกำหนดผลลัพธ์ล่วงหน้าได้
			// (ต้องแนบ header X-Test-Run-Id ต่างกันในแต่ละเทส ไม่งั้นเทสจะแย่ง
			// ตัวนับกัน)
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
