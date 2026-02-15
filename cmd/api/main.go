package main

import (
	"context"
	"log"
	"os"

	_ "photostudio/docs"
	"photostudio/internal/config"
	"photostudio/internal/database"
	"photostudio/internal/domain/admin"
	"photostudio/internal/domain/auth"
	"photostudio/internal/domain/booking"
	"photostudio/internal/domain/catalog"
	"photostudio/internal/domain/chat"
	"photostudio/internal/domain/favorite"
	"photostudio/internal/domain/lead"
	"photostudio/internal/domain/manager"
	"photostudio/internal/domain/mwork"
	"photostudio/internal/domain/notification"
	"photostudio/internal/domain/owner"
	"photostudio/internal/domain/payment"
	"photostudio/internal/domain/profile"
	"photostudio/internal/domain/review"
	"photostudio/internal/middleware"
	jwtsvc "photostudio/internal/pkg/jwt"
	"photostudio/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           PhotoStudio API
// @version         1.0
// @description     API server for booking system.
// @basePath        /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load .env file if it exists (only in local dev)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, continuing with system env vars")
	}

	authConfig, err := config.LoadAuthRuntimeConfig()
	if err != nil {
		log.Fatalf("invalid auth runtime config: %v", err)
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
		&auth.User{},
		&owner.StudioOwner{},
		&catalog.Studio{},
		&catalog.Room{},
		&catalog.Equipment{},
		&booking.Booking{},
		&review.Review{},
		&notification.Notification{},
		&notification.UserPreferences{},
		&notification.DeviceToken{},
		&chat.Conversation{},
		&chat.Message{},
		&chat.BlockedUser{},
		&favorite.Favorite{},
		&owner.OwnerPIN{},
		&owner.ProcurementItem{},
		&owner.MaintenanceItem{},
		&owner.CompanyProfile{},
		&owner.PortfolioProject{},
		&catalog.StudioWorkingHours{}, // Добавляем новую таблицу
		&payment.RobokassaPayment{},
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

	// SQLx connection for new modules
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB: %v", err)
	}
	sqlxDB := sqlx.NewDb(sqlDB, "postgres")

	// Repositories
	userRepo := auth.NewUserRepository(db)
	studioRepo := catalog.NewStudioRepository(db)
	roomRepo := catalog.NewRoomRepository(db)
	equipmentRepo := catalog.NewEquipmentRepository(db)
	bookingRepo := booking.NewBookingRepository(db)
	reviewRepo := review.NewReviewRepository(db)
	studioOwnerRepo := owner.NewOwnerRepository(db)
	studioWorkingHoursRepo := catalog.NewStudioWorkingHoursRepository(db)

	chatRepo := chat.NewChatRepository(db)
	favoriteRepo := favorite.NewFavoriteRepository(db)
	ownerCRMRepo := owner.NewOwnerCRMRepository(db)
	robokassaPaymentRepo := payment.NewRobokassaPaymentRepository(db)

	// Profile Repositories
	clientProfileRepo := profile.NewClientRepository(sqlxDB)
	ownerProfileRepo := profile.NewOwnerRepository(sqlxDB)
	adminProfileRepo := profile.NewAdminRepository(sqlxDB)

	// Lead Repository
	leadRepo := lead.NewRepository(sqlxDB)

	// Shared services
	jwtService := jwtsvc.NewWithLegacy(authConfig.JWTSecret, authConfig.JWTAccessTTL, true)

	// Ownership checker (for catalog module)
	ownershipChecker := middleware.NewOwnershipChecker(studioRepo, roomRepo)

	// Module services & handlers
	profileService := profile.NewService(clientProfileRepo, ownerProfileRepo, adminProfileRepo)

	authMailer := auth.NewDevConsoleMailer(authConfig.AppEnv == "dev" || authConfig.AppEnv == "development")
	authService := auth.NewService(userRepo, studioOwnerRepo, profileService, jwtService, authMailer, authConfig.VerificationCodePepper, authConfig.VerifyCodeTTL, authConfig.VerifyResendCooldown, authConfig.RefreshTokenPepper, authConfig.RefreshTTL)
	authHandler := auth.NewHandler(authService, profileService, bookingRepo, authConfig.CookieSecure, authConfig.CookieSameSite, authConfig.CookiePath)

	leadService := lead.NewService(leadRepo, userRepo)
	leadHandler := lead.NewHandler(leadService, profileService)

	// Ensure admin profile for initial admin if needed (optional, or rely on manual creation/db seed)

	catalogService := catalog.NewService(studioRepo, roomRepo, equipmentRepo, studioWorkingHoursRepo)
	catalogHandler := catalog.NewHandler(catalogService, userRepo)

	// Notification repositories (new architecture)
	notifRepo := notification.NewRepository(db)
	prefRepo := notification.NewPreferencesRepository(db)
	deviceTokenRepo := notification.NewDeviceTokenRepository(db)

	notificationService := notification.NewService(notifRepo, prefRepo, deviceTokenRepo)
	notificationExtendedService := notification.NewExtendedService(notificationService, &notification.ExternalServices{
		EmailService: nil,  // TODO: integrate email service
		PushService:  nil,  // TODO: integrate push service
	})
	// keep extended service referenced for now (integration point)
	_ = notificationExtendedService

	// Initialize notification handlers
	notificationHandler := notification.NewHandler(notificationService)
	preferencesHandler := notification.NewPreferencesHandler(notificationService)
	deviceTokensHandler := notification.NewDeviceTokensHandler(notificationService)

	// Initialize cleanup service
	cleanupService := notification.NewCleanupService(notifRepo, deviceTokenRepo)
	cleanupConfig := notification.DefaultCleanupConfig()
	// Start scheduled cleanup in background
	stopCleanup := cleanupService.ScheduleCleanup(context.Background(), cleanupConfig)
	defer close(stopCleanup) // Stop cleanup on shutdown

	bookingService := booking.NewService(bookingRepo, roomRepo, notificationService, studioWorkingHoursRepo)
	bookingHandler := booking.NewHandler(bookingService)

	reviewService := review.NewService(reviewRepo, bookingRepo, studioRepo)
	reviewHandler := review.NewHandler(reviewService)
	_ = reviewHandler
	// Admin domain
	adminRepo := admin.NewAdminRepository(db)
	adminService := admin.NewService(
		userRepo,
		studioRepo,
		bookingRepo,
		reviewRepo,
		studioOwnerRepo,
		adminRepo,
		profileService,
		jwtService,
		nil, // NotificationSender (if any)
	)

	adminAuthHandler := admin.NewAuthHandler(adminService)
	adminManagementHandler := admin.NewManagementHandler(adminService)
	adminHandler := admin.NewHandler(adminService, adminAuthHandler, adminManagementHandler)

	chatService := chat.NewService(chatRepo, userRepo, studioRepo, bookingRepo, notificationService)
	chatHandler := chat.NewHandler(chatService)
	favoriteHandler := favorite.NewHandler(favoriteRepo)
	chatHub := chat.NewHub()
	chatWSHandler := chat.NewWSHandler(chatHub, jwtService, chatService)

	ownerHandler := owner.NewHandler(ownerCRMRepo)

	managerHandler := manager.NewHandler(bookingRepo, ownerCRMRepo)

	mworkService := mwork.NewService(userRepo)
	mworkHandler := mwork.NewHandler(mworkService)

	paymentLogger := func(format string, args ...interface{}) { log.Printf(format, args...) }
	// Adapter for booking service to match payment expectations if needed, or update payment service
	// For now assuming existing payment service signature is correct for the codebase
	paymentService := payment.NewService(robokassaPaymentRepo, bookingRepo, bookingRepo, paymentLogger) // bookingRepo implements all needed interfaces now
	paymentHandler := payment.NewHandler(paymentService, paymentLogger)

	// Initialize new profile handlers
	clientProfileHandler := profile.NewClientHandler(profileService)
	ownerProfileHandler := profile.NewOwnerHandler(profileService)
	adminProfileHandler := profile.NewAdminHandler(profileService)

	// CORS Setup
	r := gin.Default()
	r.Use(middleware.CORS())
	r.Use(middleware.ErrorLogger()) // Add error logger middleware

	// Serve static files

	// Docs
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Set debug mode for detailed errors (as requested by user)
	// Ideally this should be config-driven, but we enable it for now to expose errors
	response.SetDebug(true)

	// Routes
	v1 := r.Group("/api/v1")

	// Public routes
	authHandler.RegisterPublicRoutes(v1)
	catalogHandler.RegisterRoutes(v1)          // Полный набор публичных маршрутов
	lead.RegisterPublicRoutes(v1, leadHandler) // Changed from RegisterRoutes
	adminHandler.RegisterPublicRoutes(v1)      // New Admin Login (Public)

	// Webhooks
	paymentHandler.RegisterWebhookRoutes(v1)

	// Admin routes (Protected by AdminJWTAuth)
	adminGroup := v1.Group("/admin")
	adminGroup.Use(admin.AdminJWTAuth(jwtService))
	{
		adminHandler.RegisterProtectedRoutes(adminGroup)
		lead.RegisterAdminRoutes(adminGroup, leadHandler)
	}

	// Protected routes
	protected := v1.Group("/")
	protected.Use(middleware.JWTAuth(jwtService))
	{
		authHandler.RegisterProtectedRoutes(protected)
		profile.RegisterRoutes(protected, clientProfileHandler, ownerProfileHandler, adminProfileHandler)

		catalogHandler.RegisterProtectedRoutes(protected, ownershipChecker)
		bookingHandler.RegisterRoutes(protected) // Полный набор маршрутов бронирования
		bookingHandler.RegisterStudioRoutes(protected, ownershipChecker)

		// Notification routes
		notification.RegisterRoutes(protected, notificationHandler, preferencesHandler, deviceTokensHandler)

		// Chat routes
		protected.GET("/chat/ws", chatWSHandler.HandleWebSocket)
		chatHandler.RegisterRoutes(protected)
		favoriteHandler.RegisterRoutes(protected)

		// Manager routes
		// managerHandler.RegisterRoutes(protected) // Undefined in list?

		// MWork routes
		mworkHandler.RegisterRoutes(protected) // Check if defined

		// Payment routes
		paymentHandler.RegisterProtectedRoutes(protected)

		// Owner CRM routes (require studio_owner role)
		ownerCRMGroup := protected.Group("")
		ownerCRMGroup.Use(middleware.RequireRole(string(auth.RoleStudioOwner)))
		{
			ownerHandler.RegisterRoutes(ownerCRMGroup)
			ownerHandler.RegisterCompanyRoutes(ownerCRMGroup)
		}

		managerGroup := protected.Group("")
		managerGroup.Use(middleware.RequireRole(string(auth.RoleStudioOwner)))
		{
			managerHandler.RegisterRoutes(managerGroup)
		}

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
