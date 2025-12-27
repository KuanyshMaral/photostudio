package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"photostudio/internal/database"
	"photostudio/internal/modules/auth"
	"photostudio/internal/modules/booking"
	"photostudio/internal/modules/catalog"
	jwtsvc "photostudio/internal/pkg/jwt"
	"photostudio/internal/repository"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is empty")
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is empty")
	}

	db, err := database.Connect(dsn)
	if err != nil {
		log.Fatal(err)
	}

	userRepo := repository.NewUserRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	studioRepo := repository.NewStudioRepository(db)
	equipmentRepo := repository.NewEquipmentRepository(db)
	bookingRepo := repository.NewBookingRepository(db)

	j := jwtsvc.New(secret, 24*time.Hour)

	authService := auth.NewService(userRepo, j)
	authHandler := auth.NewHandler(authService)

	catalogService := catalog.NewService(
		studioRepo,
		roomRepo,
		equipmentRepo,
	)
	catalogHandler := catalog.NewHandler(catalogService)

	bookingService := booking.NewService(bookingRepo, roomRepo)
	bookingHandler := booking.NewHandler(bookingService)

	r := gin.Default()

	v1 := r.Group("/api/v1")
	{
		// public
		authHandler.RegisterRoutes(v1)
		catalogHandler.RegisterRoutes(v1)

		// protected (booking endpoints)
		protected := v1.Group("/")
		protected.Use(authMiddleware(j))
		{
			bookingHandler.RegisterRoutes(protected)
		}
	}

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func authMiddleware(jwt *jwtsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Missing Authorization header",
				},
			})
			return
		}

		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid Authorization header",
				},
			})
			return
		}

		tokenStr := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Empty token",
				},
			})
			return
		}

		claims, err := jwt.ValidateToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid token",
				},
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}
