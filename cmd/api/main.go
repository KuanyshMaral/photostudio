package main

import (
	"log"
	"os"
	"photostudio/internal/domain"
	"photostudio/internal/middleware"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"

	"photostudio/internal/database"
	"photostudio/internal/modules/admin"
	"photostudio/internal/modules/auth"
	"photostudio/internal/modules/booking"
	"photostudio/internal/modules/catalog"
	"photostudio/internal/modules/review"
	jwtsvc "photostudio/internal/pkg/jwt"
	"photostudio/internal/repository"
)

func main() {
	// Load .env file if it exists (only in local dev)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, continuing with system env vars")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is empty")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "studio.db" // default sqlite file
		log.Println("⚠️ DATABASE_URL not set → using SQLite: studio.db")
	}

	db, err := database.Connect(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Running AutoMigrate for local development...")
	models := []interface{}{
		domain.User{},
		domain.StudioOwner{VerificationDocs: []string{}},
		domain.Studio{},
		domain.Room{},
		domain.Equipment{},
		domain.Booking{},
		domain.Review{Photos: []string{}},
	}
	if strings.HasSuffix(databaseURL, ".db") {
		log.Println("Running AutoMigrate for local development...")
		for _, model := range models {
			if err := db.AutoMigrate(model); err != nil {
				log.Fatalf("AutoMigrate failed for %T: %v", model, err)
			}
		}
		log.Println("✅ AutoMigrate completed")
	} else {
		log.Println("Skipping AutoMigrate (non-sqlite database)")
	}

	// Repositories
	userRepo := repository.NewUserRepository(db)
	studioRepo := repository.NewStudioRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	equipmentRepo := repository.NewEquipmentRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	reviewRepo := repository.NewReviewRepository(db)
	studioOwnerRepo := repository.NewStudioOwnerRepository(db)

	// Shared services
	jwtService := jwtsvc.New(jwtSecret, 24*time.Hour)

	// Ownership checker (for catalog module)
	ownershipChecker := middleware.NewOwnershipChecker(studioRepo, roomRepo)

	// Module services & handlers
	authService := auth.NewService(userRepo, studioOwnerRepo, jwtService)
	authHandler := auth.NewHandler(authService)

	catalogService := catalog.NewService(studioRepo, roomRepo, equipmentRepo)
	catalogHandler := catalog.NewHandler(catalogService)

	bookingService := booking.NewService(bookingRepo, roomRepo)
	bookingHandler := booking.NewHandler(bookingService)

	reviewService := review.NewService(reviewRepo, bookingRepo, studioRepo)
	reviewHandler := review.NewHandler(reviewService)

	adminService := admin.NewService(userRepo, studioRepo, bookingRepo, reviewRepo)
	adminHandler := admin.NewHandler(adminService)

	// Router setup
	r := gin.New() // Better than gin.Default() — we add only what we need
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.CORS())

	v1 := r.Group("/api/v1")

	// Public routes
	authHandler.RegisterPublicRoutes(v1)
	catalogHandler.RegisterRoutes(v1) // only GET endpoints

	// Public reviews (list only)
	reviewHandler.RegisterRoutes(v1, nil)

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.JWTAuth(jwtService))

	{
		authHandler.RegisterProtectedRoutes(protected)

		// Booking
		bookingHandler.RegisterRoutes(protected)

		// Protected reviews (create, respond)
		reviewHandler.RegisterRoutes(nil, protected)

		// Protected catalog (owner actions)
		studios := protected.Group("/studios")
		{
			studios.POST("", catalogHandler.CreateStudio)
			studios.PUT("/:id", ownershipChecker.CheckStudioOwnership(), catalogHandler.UpdateStudio)
			studios.POST("/:id/rooms", ownershipChecker.CheckStudioOwnership(), catalogHandler.CreateRoom)
		}

		// Admin
		adminGroup := protected.Group("/admin")
		adminHandler.RegisterRoutes(adminGroup)
		// You can uncomment when ready
		// rooms := protected.Group("/rooms")
		// rooms.POST("/:id/equipment", ownershipChecker.CheckRoomOwnership(), catalogHandler.AddEquipment)
	}

	// Static files for uploads
	r.Static("/static", "./uploads")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Printf("🚀 Server starting on :%s", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}
