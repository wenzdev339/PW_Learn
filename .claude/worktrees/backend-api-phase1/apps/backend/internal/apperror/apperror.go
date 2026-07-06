package apperror

import "github.com/gin-gonic/gin"

type AppError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *AppError) Error() string {
	return e.Message
}

func New(statusCode int, code, message string) *AppError {
	return &AppError{StatusCode: statusCode, Code: code, Message: message}
}

// RespondError writes the standard {"error": {"code", "message"}} envelope.
// If err is an *AppError its status/code are used, otherwise a generic 500.
func RespondError(c *gin.Context, err error) {
	if appErr, ok := err.(*AppError); ok {
		c.JSON(appErr.StatusCode, gin.H{
			"error": gin.H{"code": appErr.Code, "message": appErr.Message},
		})
		return
	}
	c.JSON(500, gin.H{
		"error": gin.H{"code": "INTERNAL_ERROR", "message": "Internal server error"},
	})
}

// RespondValidationError writes a 400 VALIDATION_ERROR envelope, used when
// request binding/validation (e.g. c.ShouldBindJSON) fails.
func RespondValidationError(c *gin.Context, err error) {
	c.JSON(400, gin.H{
		"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()},
	})
}
