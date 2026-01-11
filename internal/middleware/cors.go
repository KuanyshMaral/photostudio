package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	// Базовые разрешённые origins (локальная разработка)
	allowedOrigins := map[string]bool{
		"http://localhost:3000": true,
		"http://localhost:5173": true,
		"http://127.0.0.1:3000": true,
		"http://127.0.0.1:5173": true,
	}

	// Дополнительные origins из ENV (на будущее)
	// пример: CORS_ALLOWED_ORIGINS=https://app.com,https://admin.app.com
	if extra := os.Getenv("CORS_ALLOWED_ORIGINS"); extra != "" {
		for _, o := range strings.Split(extra, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowedOrigins[o] = true
			}
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Если Origin есть и он разрешён — отражаем его (важно для credentials)
		if origin != "" && allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin") // важно для кешей/прокси
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Всегда полезно отдавать эти заголовки (и для preflight тоже)
		c.Writer.Header().Set("Access-Control-Allow-Headers",
			"Content-Type, Content-Length, Authorization, Accept, Origin, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods",
			"GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Max-Age", "600")

		// Preflight запросы должны завершаться ДО JWT/Role middleware
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent) // 204
			return
		}

		c.Next()
	}
}
