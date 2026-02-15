package response

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

var debugMode = false

// Response is a generic response structure for Swagger documentation
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorData  `json:"error,omitempty"`
}

// ErrorData represents error details in the response
type ErrorData struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// SetDebug enables or disables detailed error responses
func SetDebug(debug bool) {
	debugMode = debug
}

func Success(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, gin.H{
		"success": true,
		"data":    data,
	})
}

func Error(c *gin.Context, statusCode int, code string, message string) {
	resp := gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	}
	c.JSON(statusCode, resp)
}

func ErrorWithDetails(c *gin.Context, statusCode int, code string, message string, details any) {
	resp := gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
			"details": details,
		},
	}
	c.JSON(statusCode, resp)
}

// ServerError sends a 500 error with details if debug mode is on
func ServerError(c *gin.Context, err error) {
	_ = c.Error(err) // Ensure it's logged by middleware

	msg := "Internal Server Error"
	details := ""

	if debugMode {
		msg = err.Error()
		details = fmt.Sprintf("%+v", err)
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": msg,
			"details": details,
		},
	})
}

// CustomError sends an error response with details derived from the error object or string
func CustomError(c *gin.Context, statusCode int, code string, errOrMsg any) {
	var err error
	var msg string

	switch v := errOrMsg.(type) {
	case error:
		err = v
		msg = v.Error()
	case string:
		err = fmt.Errorf("%s", v)
		msg = v
	default:
		err = fmt.Errorf("%v", v)
		msg = fmt.Sprintf("%v", v)
	}

	_ = c.Error(err) // Ensure it's logged by middleware

	details := ""
	if debugMode {
		details = fmt.Sprintf("%+v", err)
	}

	c.JSON(statusCode, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": msg,
			"details": details,
		},
	})
}
