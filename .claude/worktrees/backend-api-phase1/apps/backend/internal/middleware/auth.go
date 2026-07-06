package middleware

import (
	"net/http"
	"strings"

	"backend/internal/apperror"
	"backend/internal/token"

	"github.com/gin-gonic/gin"
)

func RequireAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			apperror.RespondError(c, apperror.New(http.StatusUnauthorized, "UNAUTHORIZED", "Missing access token"))
			c.Abort()
			return
		}
		claims, err := token.Verify(strings.TrimPrefix(header, "Bearer "), secret)
		if err != nil {
			apperror.RespondError(c, apperror.New(http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired access token"))
			c.Abort()
			return
		}
		c.Set("userID", claims.Subject)
		c.Set("userRole", claims.Role)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("userRole")
		if role != "ADMIN" {
			apperror.RespondError(c, apperror.New(http.StatusForbidden, "FORBIDDEN", "Admin role required"))
			c.Abort()
			return
		}
		c.Next()
	}
}
