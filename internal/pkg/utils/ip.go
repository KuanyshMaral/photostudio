package utils

import (
	"net"
	"net/http"
	"strings"
)

// GetClientIP безопасно извлекает IP адрес клиента, очищая его от порта
func GetClientIP(r *http.Request) string {
	// Проверяем заголовки прокси
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}

	// Если заголовков нет, берем RemoteAddr и отрезаем порт
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Если SplitHostPort вернул ошибку (например, порта изначально не было), возвращаем как есть
		return r.RemoteAddr
	}
	return ip
}
