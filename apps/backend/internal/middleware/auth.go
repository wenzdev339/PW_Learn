package middleware

import (
	"net/http"
	"strings"

	"backend/internal/apperror"
	"backend/internal/token"

	"github.com/gin-gonic/gin"
)

// RequireAuth เช็คว่า request แนบ header "Authorization: Bearer <token>"
// มาไหม แล้วตรวจลายเซ็นของ JWT ด้วย secret ที่กำหนด (ต้องตรงกับ secret ที่ใช้
// sign ตอน login — ดู internal/token/token.go และ internal/auth/service.go)
// ถ้าผ่านหมด จะเก็บ userID/userRole ไว้ใน Gin context ผ่าน c.Set() เพื่อให้
// handler ถัดไปเรียกอ่านด้วย c.MustGet("userID") ได้โดยไม่ต้อง decode token ซ้ำ
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

// RequireAdmin ต้องเรียกต่อจาก RequireAuth เสมอ (พึ่งค่า "userRole" ที่
// RequireAuth เซ็ตไว้ใน context) — เช็คซ้ำอีกชั้นว่า role ต้องเป็น ADMIN
// เท่านั้นถึงจะเข้าถึง route ได้ ใช้กับกลุ่ม /api/v1/admin/* ทั้งหมด
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
