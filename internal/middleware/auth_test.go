package middleware

import (
	"net/http"
	"net/http/httptest"
	"photostudio/internal/pkg/jwt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestJWTAuth_ValidToken(t *testing.T) {
	// Arrange
	secret := "test-secret-123"
	jwtService := jwt.New(secret, 1*time.Hour)
	validToken, _ := jwtService.GenerateToken(42, "client")

	// Create test router with middleware + test endpoint
	router := gin.New()
	router.Use(JWTAuth(jwtService)) // apply middleware

	router.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		role, _ := c.Get("role")
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"role":    role,
		})
	})

	// Act
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "42")
	assert.Contains(t, w.Body.String(), "client")
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	jwtService := jwt.New("wrong-secret", 1*time.Hour)

	router := gin.New()
	router.Use(JWTAuth(jwtService))

	router.GET("/protected", func(c *gin.Context) {
		t.Fatal("This handler should not be reached")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-jwt-here")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_TOKEN")
}

func TestJWTAuth_NoToken(t *testing.T) {
	jwtService := jwt.New("secret", 1*time.Hour)

	router := gin.New()
	router.Use(JWTAuth(jwtService))

	router.GET("/protected", func(c *gin.Context) {
		t.Fatal("Should not reach here")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	// No Authorization header
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "AUTH_HEADER_MISSING")
}

func TestJWTAuth_WrongFormat(t *testing.T) {
	jwtService := jwt.New("secret", 1*time.Hour)

	router := gin.New()
	router.Use(JWTAuth(jwtService))

	router.GET("/protected", func(c *gin.Context) {
		t.Fatal("Should not reach here")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Basic dGVzdA==")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_AUTH_FORMAT")
}

func TestJWTAuth_QueryTokenRejectedForNormalHTTP(t *testing.T) {
	secret := "test-secret-123"
	jwtService := jwt.New(secret, 1*time.Hour)
	validToken, _ := jwtService.GenerateToken(42, "client")

	router := gin.New()
	router.Use(JWTAuth(jwtService))
	router.GET("/api/v1/notifications", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/notifications?token="+validToken, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "AUTH_HEADER_MISSING")
}

func TestJWTAuth_QueryTokenAllowedForNotificationsWSUpgrade(t *testing.T) {
	secret := "test-secret-123"
	jwtService := jwt.New(secret, 1*time.Hour)
	validToken, _ := jwtService.GenerateToken(42, "client")

	router := gin.New()
	router.Use(JWTAuth(jwtService))
	router.GET("/api/v1/notifications/ws", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/notifications/ws?token="+validToken, nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTAuth_QueryTokenRejectedForOtherWSPath(t *testing.T) {
	secret := "test-secret-123"
	jwtService := jwt.New(secret, 1*time.Hour)
	validToken, _ := jwtService.GenerateToken(42, "client")

	router := gin.New()
	router.Use(JWTAuth(jwtService))
	router.GET("/api/v1/rooms/ws", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/rooms/ws?token="+validToken, nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "AUTH_HEADER_MISSING")
}

func TestJWTAuth_AdminTokenRejectedOnRegularUserEndpoint(t *testing.T) {
	secret := "test-secret-123"
	jwtService := jwt.New(secret, 1*time.Hour)
	adminToken, _ := jwtService.GenerateAdminToken("a729c521-9b39-402b-95c5-0414e00a456c", "admin")

	router := gin.New()
	router.Use(JWTAuth(jwtService))
	router.GET("/api/v1/chats", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/chats", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Admin token is not allowed")
}

func TestJWTAuth_AdminTokenAllowedForAddChatMember(t *testing.T) {
	secret := "test-secret-123"
	jwtService := jwt.New(secret, 1*time.Hour)
	adminToken, _ := jwtService.GenerateAdminToken("a729c521-9b39-402b-95c5-0414e00a456c", "admin")

	router := gin.New()
	router.Use(JWTAuth(jwtService))
	router.POST("/api/v1/chats/:id/members", func(c *gin.Context) {
		_, hasAdminID := c.Get("admin_id")
		_, hasUserID := c.Get("user_id")
		isAdminToken, _ := c.Get("is_admin_token")
		c.JSON(http.StatusOK, gin.H{
			"has_admin_id":   hasAdminID,
			"has_user_id":    hasUserID,
			"is_admin_token": isAdminToken,
		})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/chats/7ae6034b-f7c7-4505-9c13-f457a7f47561/members", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"has_admin_id\":true")
	assert.Contains(t, w.Body.String(), "\"has_user_id\":false")
	assert.Contains(t, w.Body.String(), "\"is_admin_token\":true")
}
