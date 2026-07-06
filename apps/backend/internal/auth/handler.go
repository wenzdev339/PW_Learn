package auth

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const refreshCookieName = "refreshToken"
const refreshCookieMaxAge = 7 * 24 * 60 * 60 // seconds

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

func setRefreshCookie(c *gin.Context, value string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(refreshCookieName, value, refreshCookieMaxAge, "/", "", false, true)
}

func userJSON(u *models.User) gin.H {
	return gin.H{"id": u.ID, "email": u.Email, "name": u.Name, "role": u.Role}
}

// RegisterRoutes mounts the auth endpoints on rg (expected to be the
// "/api/v1/auth" group).
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config) {
	rg.POST("/register", func(c *gin.Context) {
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		user, err := RegisterUser(db, req.Email, req.Password, req.Name)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		tokens, err := issueTokens(cfg, user.ID, string(user.Role))
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		setRefreshCookie(c, tokens.RefreshToken)
		c.JSON(http.StatusCreated, gin.H{
			"data": gin.H{"user": userJSON(user), "accessToken": tokens.AccessToken},
		})
	})

	rg.POST("/login", func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		user, err := LoginUser(db, req.Email, req.Password)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		tokens, err := issueTokens(cfg, user.ID, string(user.Role))
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		setRefreshCookie(c, tokens.RefreshToken)
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{"user": userJSON(user), "accessToken": tokens.AccessToken},
		})
	})

	rg.POST("/refresh", func(c *gin.Context) {
		refreshToken, err := c.Cookie(refreshCookieName)
		if err != nil || refreshToken == "" {
			apperror.RespondError(c, apperror.New(http.StatusUnauthorized, "UNAUTHORIZED", "Missing refresh token"))
			return
		}
		tokens, err := RotateRefreshToken(db, cfg, refreshToken)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		setRefreshCookie(c, tokens.RefreshToken)
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"accessToken": tokens.AccessToken}})
	})

	rg.POST("/logout", func(c *gin.Context) {
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(refreshCookieName, "", -1, "/", "", false, true)
		c.Status(http.StatusNoContent)
	})

	rg.POST("/forgot-password", func(c *gin.Context) {
		var req forgotPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		if err := RequestPasswordReset(db, req.Email); err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "If the email exists, a reset link has been sent"}})
	})

	rg.POST("/reset-password", func(c *gin.Context) {
		var req resetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		if err := ResetPassword(db, req.Token, req.NewPassword); err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "Password has been reset"}})
	})
}
