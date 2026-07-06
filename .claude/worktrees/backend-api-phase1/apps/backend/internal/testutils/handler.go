package testutils

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/seed"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes mounts POST "/reset" on rg (expected "/test"). Blocked
// unless cfg.AppEnv == "test" or cfg.AllowTestReset is true.
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
