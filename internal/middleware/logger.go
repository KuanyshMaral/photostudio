package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorLogger logs detailed error information and recovers from panics.
func ErrorLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		defer func() {
			if recovered := recover(); recovered != nil {
				logRequestError(c, start, "panic", fmt.Sprintf("%v", recovered), debug.Stack())
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			if len(c.Errors) == 0 {
				if c.Writer.Status() >= http.StatusInternalServerError {
					logRequestError(c, start, "http_error", fmt.Sprintf("status=%d", c.Writer.Status()), debug.Stack())
				}
				return
			}

			for _, err := range c.Errors {
				logRequestError(c, start, fmt.Sprintf("%v", err.Type), err.Error(), debug.Stack())
				if err.Meta != nil {
					log.Printf("request_error_meta request_id=%s meta=%+v", requestID(c), err.Meta)
				}
			}
		}()

		c.Next()
	}
}

func logRequestError(c *gin.Context, start time.Time, errType string, message string, stack []byte) {
	log.Printf(
		"request_error type=%s status=%d method=%s path=%s query=%s client_ip=%s user_id=%d role=%s request_id=%s latency=%s error=%q stack=%s",
		errType,
		c.Writer.Status(),
		c.Request.Method,
		c.Request.URL.Path,
		c.Request.URL.RawQuery,
		c.ClientIP(),
		c.GetInt64("user_id"),
		c.GetString("role"),
		requestID(c),
		time.Since(start),
		message,
		string(stack),
	)
}

func requestID(c *gin.Context) string {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = c.GetHeader("X-Request-Id")
	}
	return requestID
}
