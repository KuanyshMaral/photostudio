package response

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// isDevMode returns true when the process is NOT running in production.
// Controlled by APP_ENV env var. Default: dev (details visible).
func isDevMode() bool {
	env := os.Getenv("APP_ENV")
	return env != "production" && env != "prod" && env != "release"
}

// ──────────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────────

// Response is a generic response structure for Swagger documentation
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorData  `json:"error,omitempty"`
}

// ErrorData represents error details in the response
type ErrorData struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Details    interface{} `json:"details,omitempty"`
	ErrorTrace string      `json:"error_trace,omitempty"` // Full error + stack trace (non-prod only)
}

// SetDebug is kept for backward compatibility. Use APP_ENV instead.
func SetDebug(_ bool) {}

// ──────────────────────────────────────────────────
// Success helpers
// ──────────────────────────────────────────────────

func Success(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, gin.H{
		"success": true,
		"data":    data,
	})
}

// ──────────────────────────────────────────────────
// Error helpers
// ──────────────────────────────────────────────────

// Error — static string message error (no trace)
func Error(c *gin.Context, statusCode int, code string, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

// ErrorWithDetails — static message with extra context map
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

// CustomError — the main workhorse.
// Accepts error or string as errOrMsg.
// In non-production: logs to terminal and includes full error_trace in the JSON response.
// In production: message still shows, but error_trace is omitted.
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

	_ = c.Error(err) // attach to Gin context (picked up by logger middleware)

	errTrace := ""
	if isDevMode() && err != nil {
		errTrace = fmt.Sprintf("Error: %v\n\nStack Trace:\n%s", err.Error(), string(debug.Stack()))
		log.Printf("[ERROR] %s %s → %s: %s\n%s",
			c.Request.Method, c.Request.URL.Path, code, msg, errTrace)
	}

	body := gin.H{
		"code":    code,
		"message": msg,
	}
	if errTrace != "" {
		body["error_trace"] = errTrace
	}

	c.JSON(statusCode, gin.H{
		"success": false,
		"error":   body,
	})
}

// ServerError — convenience wrapper for 500 errors arising from unexpected Go errors.
func ServerError(c *gin.Context, err error) {
	_ = c.Error(err)

	errTrace := ""
	if isDevMode() && err != nil {
		errTrace = fmt.Sprintf("Error: %v\n\nStack Trace:\n%s", err.Error(), string(debug.Stack()))
		log.Printf("[ERROR] 500 %s %s → INTERNAL_ERROR: %v\n%s",
			c.Request.Method, c.Request.URL.Path, err, errTrace)
	}

	msg := "Internal Server Error"
	if isDevMode() && err != nil {
		msg = err.Error()
	}

	body := gin.H{
		"code":    "INTERNAL_ERROR",
		"message": msg,
	}
	if errTrace != "" {
		body["error_trace"] = errTrace
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error":   body,
	})
}

// NotFound — 404 shorthand
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, "NOT_FOUND", message)
}

// Unauthorized — 401 shorthand
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden — 403 shorthand
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, "FORBIDDEN", message)
}

// BadRequest — 400 shorthand
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

// Conflict — 409 shorthand
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, "CONFLICT", message)
}
