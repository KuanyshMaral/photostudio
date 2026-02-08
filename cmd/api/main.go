package main

import (
	"log"
	"os"
	"photostudio/internal/domain"
	"photostudio/internal/middleware"
	"photostudio/internal/modules/favorite"
	"photostudio/internal/modules/mwork"
	"time"

	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"

	"photostudio/internal/database"
	"photostudio/internal/modules/admin"
	"photostudio/internal/modules/auth"
	"photostudio/internal/modules/booking"
	"photostudio/internal/modules/catalog"
	"photostudio/internal/modules/chat"
	"photostudio/internal/modules/manager"
	"photostudio/internal/modules/notification"
	"photostudio/internal/modules/owner"
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

	models := []interface{}{
		&domain.User{},
		&domain.StudioOwner{},
		&domain.Studio{},
		&domain.Room{},
		&domain.Equipment{},
		&domain.Booking{},
		&domain.Review{},
		&domain.Notification{},
		&domain.Conversation{},
		&domain.Message{},
		&domain.BlockedUser{},
		&domain.Favorite{},
		&domain.OwnerPIN{},
		&domain.ProcurementItem{},
		&domain.MaintenanceItem{},
		&domain.CompanyProfile{},
		&domain.PortfolioProject{},
		&domain.StudioWorkingHours{}, // Добавляем новую таблицу
	}

	// Check if migrations should be run via environment variable
	runMigrations := os.Getenv("DB_AUTO_MIGRATE")
	if runMigrations == "true" || runMigrations == "1" {
		log.Println("🔄 Running database migrations (DB_AUTO_MIGRATE=true)...")
		for _, model := range models {
			if err := db.AutoMigrate(model); err != nil {
				log.Fatalf("AutoMigrate failed for %T: %v", model, err)
			}
		}
		log.Println("✅ AutoMigrate completed successfully")
	} else {
		log.Println("⏭️  Skipping AutoMigrate (DB_AUTO_MIGRATE not set or false)")
	}

	// Repositories
	userRepo := repository.NewUserRepository(db)
	studioRepo := repository.NewStudioRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	equipmentRepo := repository.NewEquipmentRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	reviewRepo := repository.NewReviewRepository(db)
	studioOwnerRepo := repository.NewOwnerRepository(db)
	studioWorkingHoursRepo := repository.NewStudioWorkingHoursRepository(db)

	notificationRepo := repository.NewNotificationRepository(db)
	chatRepo := repository.NewChatRepository(db)
	favoriteRepo := repository.NewFavoriteRepository(db)
	ownerCRMRepo := repository.NewOwnerCRMRepository(db)
	// Shared services
	jwtService := jwtsvc.New(jwtSecret, 24*time.Hour)

	// Ownership checker (for catalog module)
	ownershipChecker := middleware.NewOwnershipChecker(studioRepo, roomRepo)

	// Module services & handlers
	authService := auth.NewService(userRepo, studioOwnerRepo, jwtService)
	authHandler := auth.NewHandler(authService, bookingRepo)

	catalogService := catalog.NewService(studioRepo, roomRepo, equipmentRepo, studioWorkingHoursRepo)
	catalogHandler := catalog.NewHandler(catalogService, userRepo)

	notificationService := notification.NewService(notificationRepo)
	notificationHandler := notification.NewHandler(notificationService)

	// В main.go, найдите создание bookingService и обновите:
	bookingService := booking.NewService(bookingRepo, roomRepo, notificationService, studioWorkingHoursRepo)
	bookingHandler := booking.NewHandler(bookingService)

	reviewService := review.NewService(reviewRepo, bookingRepo, studioRepo)
	reviewHandler := review.NewHandler(reviewService)

	adminService := admin.NewService(userRepo, studioRepo, bookingRepo, reviewRepo, studioOwnerRepo, notificationService)
	adminHandler := admin.NewHandler(adminService)

	chatService := chat.NewService(chatRepo, userRepo, studioRepo, bookingRepo, notificationService)
	chatHandler := chat.NewHandler(chatService)
	favoriteHandler := favorite.NewHandler(favoriteRepo)
	chatHub := chat.NewHub()
	chatWSHandler := chat.NewWSHandler(chatHub, jwtService, chatService)

	ownerHandler := owner.NewHandler(ownerCRMRepo)

	managerHandler := manager.NewHandler(bookingRepo, ownerCRMRepo)

	mworkService := mwork.NewService(userRepo)
	mworkHandler := mwork.NewHandler(mworkService)

	// Router setup
	r := gin.New() // Better than gin.Default() — we add only what we need
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.CORS())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

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
		notificationHandler.RegisterRoutes(protected)

		chatHandler.RegisterRoutes(protected)
		favoriteHandler.RegisterRoutes(protected)
		studios := protected.Group("/studios")
		{
			studios.POST("", catalogHandler.CreateStudio)
			studios.PUT("/:id", ownershipChecker.CheckStudioOwnership(), catalogHandler.UpdateStudio)
			studios.POST("/:id/rooms", ownershipChecker.CheckStudioOwnership(), catalogHandler.CreateRoom)
			studios.POST("/:id/photos", ownershipChecker.CheckStudioOwnership(), catalogHandler.UploadStudioPhotos)
			studios.GET("/:id/bookings", ownershipChecker.CheckStudioOwnership(), bookingHandler.GetStudioBookings)
		}

		// Admin routes (require admin role)
		adminGroup := protected.Group("/admin")
		adminGroup.Use(middleware.RequireRole("admin"))
		{
			adminHandler.RegisterRoutes(adminGroup)
		}

		// Owner routes (for GetMyStudios)
		ownerGroup := protected.Group("/studios")
		ownerGroup.Use(middleware.RequireRole(string(domain.RoleStudioOwner)))
		{
			ownerGroup.GET("/my", catalogHandler.GetMyStudios)
		}

		// Owner CRM routes (require studio_owner role)
		ownerCRMGroup := protected.Group("")
		ownerCRMGroup.Use(middleware.RequireRole(string(domain.RoleStudioOwner)))
		{
			ownerHandler.RegisterRoutes(ownerCRMGroup)
			ownerHandler.RegisterCompanyRoutes(ownerCRMGroup)
		}

		managerGroup := protected.Group("")
		managerGroup.Use(middleware.RequireRole(string(domain.RoleStudioOwner)))
		{
			managerHandler.RegisterRoutes(managerGroup)
		}

		// You can uncomment when ready
		rooms := protected.Group("/rooms")
		rooms.POST("/:id/equipment", ownershipChecker.CheckRoomOwnership(), catalogHandler.AddEquipment)
		rooms.GET("", catalogHandler.GetRooms)        // GET /api/v1/rooms
		rooms.GET("/:id", catalogHandler.GetRoomByID) // GET /api/v1/rooms/:id
		rooms.PUT("/:id", ownershipChecker.CheckRoomOwnership(), catalogHandler.UpdateRoom)
		rooms.DELETE("/:id", ownershipChecker.CheckRoomOwnership(), catalogHandler.DeleteRoom)

	}
	// Chat WebSocket route (public, auth via query param)
	r.GET("/ws/chat", chatWSHandler.HandleWebSocket)

	internal := r.Group("/internal")
	internal.Use(middleware.InternalTokenAuth())
	{
		mworkHandler.RegisterRoutes(internal)

		// MWork-authenticated booking routes (with user ID mapping)
		mworkBookings := internal.Group("/mwork")
		mworkBookings.Use(middleware.MWorkUserAuth(userRepo))
		{
			// POST /internal/mwork/bookings - create booking with X-MWork-User-ID header
			mworkBookings.POST("/bookings", bookingHandler.CreateBooking)
			// GET /internal/mwork/bookings - list my bookings
			mworkBookings.GET("/bookings", bookingHandler.GetMyBookings)
			// GET /internal/mwork/studios - list studios (public data)
			mworkBookings.GET("/studios", catalogHandler.GetStudios)
			// GET /internal/mwork/studios/:id - studio details
			mworkBookings.GET("/studios/:id", catalogHandler.GetStudioByID)
			// GET /internal/mwork/rooms/:id/availability - room availability
			mworkBookings.GET("/rooms/:id/availability", bookingHandler.GetRoomAvailability)
			// GET /internal/mwork/rooms/:id/busy-slots - room busy slots
			mworkBookings.GET("/rooms/:id/busy-slots", bookingHandler.GetBusySlots)
		}
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
