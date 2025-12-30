package response

import "github.com/gin-gonic/gin"

func Success(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, gin.H{
		"success": true,
		"data":    data,
	})
}

func Error(c *gin.Context, statusCode int, code string, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

func ErrorWithDetails(c *gin.Context, statusCode int, code string, message string, details any) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
			"details": details,
		},
	})
}
