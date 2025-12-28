package main

import (
	"log"
	"net/http"
	"os"
	"photostudio/internal/middleware"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"photostudio/internal/database"
	"photostudio/internal/modules/auth"
	"photostudio/internal/modules/booking"
	"photostudio/internal/modules/catalog"
	review "photostudio/internal/modules/review"
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

	// reviews
	reviewRepo := repository.NewReviewRepository(db)

	j := jwtsvc.New(secret, 24*time.Hour)

	// Initialize ownership checker
	ownershipChecker := middleware.NewOwnershipChecker(studioRepo, roomRepo)

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

	// reviews service/handler
	reviewService := review.NewService(reviewRepo, bookingRepo, studioRepo)
	reviewHandler := review.NewHandler(reviewService)

	r := gin.Default()

	v1 := r.Group("/api/v1")
	{
		// public
		authHandler.RegisterRoutes(v1)
		catalogHandler.RegisterRoutes(v1)

		// public reviews
		// GET /api/v1/studios/:id/reviews
		reviewHandler.RegisterRoutes(v1, nil)

		// protected (booking + protected catalog + protected reviews)
		protected := v1.Group("/")
		protected.Use(authMiddleware(j, userRepo))
		{
			bookingHandler.RegisterRoutes(protected)

			// protected reviews
			// POST /api/v1/reviews
			// POST /api/v1/reviews/:id/response
			reviewHandler.RegisterRoutes(v1, protected)

			// Protected catalog endpoints with ownership checks
			studios := protected.Group("/studios")
			{
				studios.POST("", catalogHandler.CreateStudio)
				studios.PUT("/:id", ownershipChecker.CheckStudioOwnership(), catalogHandler.UpdateStudio)
				studios.POST("/:id/rooms", ownershipChecker.CheckStudioOwnership(), catalogHandler.CreateRoom)
			}

			//rooms := protected.Group("/rooms")
			//{
			//	rooms.POST("/:id/equipment", ownershipChecker.CheckRoomOwnership(), catalogHandler.AddEquipment)
			//}
		}
	}

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func authMiddleware(jwt *jwtsvc.Service, userRepo *repository.UserRepository) gin.HandlerFunc {
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

		// Load full user object
		user, err := userRepo.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "User not found",
				},
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("user", user)

		c.Next()
	}
}
