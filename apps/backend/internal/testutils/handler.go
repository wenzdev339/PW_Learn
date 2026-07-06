package testutils

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/seed"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes ผูก POST /test/reset ไว้ — endpoint นี้มีไว้ให้ Playwright
// (หรือใครก็ตามที่เขียนเทส) เรียกล้างฐานข้อมูลกลับไปเป็น seed data ชุดเดิม
// ก่อนเริ่มรันชุดเทส เพื่อให้ทุกครั้งที่รันเทส ข้อมูลตั้งต้นเหมือนเดิมเป๊ะ
// ไม่ปนกับข้อมูลที่เทสก่อนหน้าสร้างทิ้งไว้ ("test isolation")
//
// จงใจบล็อกไม่ให้เรียกได้ตอนรันจริง (production) เพราะถ้าใครยิง endpoint นี้
// เข้ามาโดยไม่ตั้งใจ ข้อมูลลูกค้าจริงทั้งหมดจะหายทันที
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config) {
	rg.POST("/reset", func(c *gin.Context) {
		if cfg.AppEnv != "test" && !cfg.AllowTestReset {
			apperror.RespondError(c, apperror.New(http.StatusForbidden, "FORBIDDEN", "Test reset requires APP_ENV=test or ALLOW_TEST_RESET=true"))
			return
		}
		if err := seed.Run(db); err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "Database reset and reseeded"}})
	})
}
