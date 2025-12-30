package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"photostudio/internal/pkg/jwt"
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
