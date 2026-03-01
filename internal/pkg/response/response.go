package response

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
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

// ErrorResponse is a standard error response structure for Swagger documentation
type ErrorResponse struct {
	Success bool       `json:"success" example:"false"`
	Error   *ErrorData `json:"error"`
}

// SuccessResponse is a standard success response structure without data payload
type SuccessResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message,omitempty" example:"Success"`
}

// ErrorData represents error details in the response
type ErrorData struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Details    interface{} `json:"details,omitempty"`
	ErrorTrace string      `json:"error_trace,omitempty"` // Full error + stack trace (non-prod only)
}

// H is a convenience alias for map[string]any
type H = map[string]any

// SetDebug is kept for backward compatibility. Use APP_ENV instead.
func SetDebug(_ bool) {}

// ──────────────────────────────────────────────────
// JSON and Binding Helpers
// ──────────────────────────────────────────────────

// JSON writes a JSON-encoded body with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// BindJSON decodes the JSON body of r into v.
func BindJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────
// Success helpers
// ──────────────────────────────────────────────────

func Success(w http.ResponseWriter, statusCode int, data interface{}) {
	JSON(w, statusCode, H{
		"success": true,
		"data":    data,
	})
}

// ──────────────────────────────────────────────────
// Error helpers
// ──────────────────────────────────────────────────

// Error — static string message error (no trace)
func Error(w http.ResponseWriter, statusCode int, code string, message string) {
	JSON(w, statusCode, H{
		"success": false,
		"error": H{
			"code":    code,
			"message": message,
		},
	})
}

// ErrorWithDetails — static message with extra context map
func ErrorWithDetails(w http.ResponseWriter, statusCode int, code string, message string, details any) {
	JSON(w, statusCode, H{
		"success": false,
		"error": H{
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
func CustomError(w http.ResponseWriter, r *http.Request, statusCode int, code string, errOrMsg any) {
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

	errTrace := ""
	if isDevMode() && err != nil {
		errTrace = fmt.Sprintf("Error: %v\n\nStack Trace:\n%s", err.Error(), string(debug.Stack()))
		log.Printf("[ERROR] %s %s → %s: %s\n%s",
			r.Method, r.URL.Path, code, msg, errTrace)
	}

	body := H{
		"code":    code,
		"message": msg,
	}
	if errTrace != "" {
		body["error_trace"] = errTrace
	}

	JSON(w, statusCode, H{
		"success": false,
		"error":   body,
	})
}

// ServerError — convenience wrapper for 500 errors arising from unexpected Go errors.
func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	errTrace := ""
	if isDevMode() && err != nil {
		errTrace = fmt.Sprintf("Error: %v\n\nStack Trace:\n%s", err.Error(), string(debug.Stack()))
		log.Printf("[ERROR] 500 %s %s → INTERNAL_ERROR: %v\n%s",
			r.Method, r.URL.Path, err, errTrace)
	}

	msg := "Internal Server Error"
	if isDevMode() && err != nil {
		msg = err.Error()
	}

	body := H{
		"code":    "INTERNAL_ERROR",
		"message": msg,
	}
	if errTrace != "" {
		body["error_trace"] = errTrace
	}

	JSON(w, http.StatusInternalServerError, H{
		"success": false,
		"error":   body,
	})
}

// NotFound — 404 shorthand
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, "NOT_FOUND", message)
}

// Unauthorized — 401 shorthand
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden — 403 shorthand
func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, "FORBIDDEN", message)
}

// BadRequest — 400 shorthand
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, "BAD_REQUEST", message)
}

// Conflict — 409 shorthand
func Conflict(w http.ResponseWriter, message string) {
	Error(w, http.StatusConflict, "CONFLICT", message)
}
