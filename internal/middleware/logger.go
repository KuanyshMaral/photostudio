package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"time"

	"photostudio/internal/pkg/chicontext"

	"github.com/go-chi/chi/v5/middleware"
)

// ErrorLogger logs detailed error information and recovers from panics.
func ErrorLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				if recovered := recover(); recovered != nil {
					err := fmt.Errorf("%v", recovered)
					logRequestError(ww, r, start, "panic", err.Error(), debug.Stack())

					ww.Header().Set("Content-Type", "application/json")
					ww.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(ww).Encode(map[string]interface{}{
						"success": false,
						"error": map[string]interface{}{
							"code":    "INTERNAL_SERVER_ERROR",
							"message": "Internal Server Error (Panic)",
							"details": err.Error(),
							"stack":   string(debug.Stack()),
						},
					})
					return
				}

				if ww.Status() >= http.StatusInternalServerError {
					logRequestError(ww, r, start, "http_error", fmt.Sprintf("status=%d", ww.Status()), debug.Stack())
				}
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

func logRequestError(ww middleware.WrapResponseWriter, r *http.Request, start time.Time, errType string, message string, stack []byte) {
	sanitizedQuery := sanitizeQuery(r.URL.RawQuery)
	log.Printf(
		"request_error type=%s status=%d method=%s path=%s query=%s client_ip=%s user_id=%d role=%s request_id=%s latency=%s error=%q stack=%s",
		errType,
		ww.Status(),
		r.Method,
		r.URL.Path,
		sanitizedQuery,
		r.RemoteAddr,
		chicontext.UserIDFromCtx(r.Context()),
		chicontext.RoleFromCtx(r.Context()),
		requestID(r),
		time.Since(start),
		message,
		string(stack),
	)
}

func sanitizeQuery(raw string) string {
	if raw == "" {
		return ""
	}
	v, err := url.ParseQuery(raw)
	if err != nil {
		return ""
	}
	if v.Has("token") {
		v.Set("token", "[REDACTED]")
	}
	return v.Encode()
}

func requestID(r *http.Request) string {
	reqID := r.Header.Get("X-Request-ID")
	if reqID == "" {
		reqID = r.Header.Get("X-Request-Id")
	}
	return reqID
}
