package main

import (
	"context"
	"log"
	"net/http"
	"os"

	_ "photostudio/docs"
	"photostudio/internal/config"
	"photostudio/internal/database"
	"photostudio/internal/domain/admin"
	"photostudio/internal/domain/attachment"
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
	"photostudio/internal/domain/relationship"
	"photostudio/internal/domain/review"
	"photostudio/internal/domain/subscription"
	"photostudio/internal/domain/upload"
	"photostudio/internal/middleware"
	jwtsvc "photostudio/internal/pkg/jwt"
	"photostudio/internal/pkg/response"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

//	@title			PhotoStudio API
//	@version		1.0
//	@description	API server for booking system.
//	@basePath		/api/v1

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, continuing with system env vars")
	}

	authConfig, err := config.LoadAuthRuntimeConfig()
	if err != nil {
		log.Fatalf("invalid auth runtime config: %v", err)
	}
	uploadCfg := config.LoadUploadConfig()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "studio.db"
		log.Println("⚠️ DATABASE_URL not set → using SQLite: studio.db")
	}

	db, err := database.Connect(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	models := []interface{}{
		&auth.User{},
		&catalog.Studio{},
		&catalog.Room{},
		&catalog.Equipment{},
		&booking.Booking{},
		&review.Review{},
		&notification.Notification{},
		&notification.UserPreferences{},
		&notification.DeviceToken{},
		&chat.Room{},
		&chat.RoomMember{},
		&chat.Message{},
		&favorite.Favorite{},
		&owner.OwnerPIN{},
		&owner.ProcurementItem{},
		&owner.MaintenanceItem{},
		&owner.CompanyProfile{},
		&owner.PortfolioProject{},
		&catalog.StudioWorkingHours{},
		&payment.RobokassaPayment{},
		&payment.Payment{},
		&payment.RecurringSubscription{},
	}

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
	studioWorkingHoursRepo := catalog.NewStudioWorkingHoursRepository(db)

	favoriteRepo := favorite.NewFavoriteRepository(db)
	ownerCRMRepo := owner.NewOwnerCRMRepository(db)
	robokassaPaymentRepo := payment.NewRobokassaPaymentRepository(db)
	paymentRepoV2 := payment.NewRepository(db)

	// Profile Repositories
	clientProfileRepo := profile.NewClientRepository(sqlxDB)
	ownerProfileRepo := profile.NewOwnerRepository(sqlxDB)
	adminProfileRepo := profile.NewAdminRepository(sqlxDB)

	// Lead Repository
	leadRepo := lead.NewRepository(sqlxDB)

	// Shared services
	jwtService := jwtsvc.NewWithLegacy(authConfig.JWTSecret, authConfig.JWTAccessTTL, true)

	// Ownership checker (for catalog/booking modules)
	ownershipChecker := middleware.NewOwnershipChecker(studioRepo, roomRepo)

	profileService := profile.NewService(clientProfileRepo, ownerProfileRepo, adminProfileRepo)

	uploadRepo := upload.NewRepository(db)
	uploadService := upload.NewService(uploadRepo, uploadCfg.LocalPath, uploadCfg.PublicURL)
	uploadHandler := upload.NewHandler(uploadService)

	attachmentRepo := attachment.NewRepository(db)
	attachmentService := attachment.NewService(attachmentRepo, uploadService)
	attachmentHandler := attachment.NewHandler(attachmentService)

	authMailer := auth.NewDevConsoleMailer(authConfig.AppEnv == "dev" || authConfig.AppEnv == "development")
	authService := auth.NewService(userRepo, ownerProfileRepo, profileService, jwtService, authMailer, authConfig.VerificationCodePepper, authConfig.VerifyCodeTTL, authConfig.VerifyResendCooldown, authConfig.RefreshTokenPepper, authConfig.RefreshTTL)
	authHandler := auth.NewHandler(authService, profileService, bookingRepo, authConfig.CookieSecure, authConfig.CookieSameSite, authConfig.CookiePath)

	leadService := lead.NewService(leadRepo, userRepo)
	leadHandler := lead.NewHandler(leadService, profileService)

	catalogService := catalog.NewService(studioRepo, roomRepo, equipmentRepo, studioWorkingHoursRepo)
	catalogHandler := catalog.NewHandler(catalogService, userRepo, attachmentService)

	notifRepo := notification.NewRepository(db)
	prefRepo := notification.NewPreferencesRepository(db)
	deviceTokenRepo := notification.NewDeviceTokenRepository(db)
	chatHub := chat.NewHub()

	notificationService := notification.NewService(notifRepo, prefRepo, deviceTokenRepo)
	notificationService.SetRealtimePublisher(chatHub)
	notificationExtendedService := notification.NewExtendedService(notificationService, &notification.ExternalServices{
		EmailService: nil,
		PushService:  nil,
	})
	_ = notificationExtendedService

	notificationHandler := notification.NewHandler(notificationService)
	notificationHandler.SetRealtimeHub(chatHub)
	preferencesHandler := notification.NewPreferencesHandler(notificationService)
	deviceTokensHandler := notification.NewDeviceTokensHandler(notificationService)

	cleanupService := notification.NewCleanupService(notifRepo, deviceTokenRepo)
	cleanupConfig := notification.DefaultCleanupConfig()
	stopCleanup := cleanupService.ScheduleCleanup(context.Background(), cleanupConfig)
	defer close(stopCleanup)

	bookingService := booking.NewService(bookingRepo, roomRepo, notificationService, studioWorkingHoursRepo)
	bookingHandler := booking.NewHandler(bookingService)

	reviewService := review.NewService(reviewRepo, bookingRepo, studioRepo)
	reviewHandler := review.NewHandler(reviewService)

	adminRepo := admin.NewAdminRepository(db)
	adminService := admin.NewService(
		userRepo,
		studioRepo,
		bookingRepo,
		reviewRepo,
		ownerProfileRepo,
		adminRepo,
		profileService,
		jwtService,
		nil,
	)

	adminAuthHandler := admin.NewAuthHandler(adminService)
	adminManagementHandler := admin.NewManagementHandler(adminService)
	adminHandler := admin.NewHandler(adminService, adminAuthHandler, adminManagementHandler)

	relationshipRepo := relationship.NewRepository(db)
	relationshipService := relationship.NewService(relationshipRepo)
	relationshipHandler := relationship.NewHandler(relationshipService)

	chatRepo := chat.NewRepository(db)
	chatService := chat.NewService(chatRepo, relationshipService)
	chatHandler := chat.NewHandler(chatService, chatHub)
	favoriteHandler := favorite.NewHandler(favoriteRepo)

	ownerHandler := owner.NewHandler(ownerCRMRepo)

	managerHandler := manager.NewHandler(bookingRepo, ownerCRMRepo)

	mworkService := mwork.NewService(userRepo)
	mworkHandler := mwork.NewHandler(mworkService)

	paymentLogger := func(format string, args ...interface{}) { log.Printf(format, args...) }
	paymentService := payment.NewService(robokassaPaymentRepo, bookingRepo, bookingRepo, paymentLogger, paymentRepoV2)
	paymentHandler := payment.NewHandler(paymentService, paymentLogger)

	clientProfileHandler := profile.NewClientHandler(profileService)
	ownerProfileHandler := profile.NewOwnerHandler(profileService)
	adminProfileHandler := profile.NewAdminHandler(profileService)

	subscriptionRepo := subscription.NewRepository(db)
	subscriptionService := subscription.NewService(subscriptionRepo, roomRepo)
	subscriptionHandler := subscription.NewHandler(subscriptionService)

	// -----------------------------------------------------------------------
	// Chi Router Setup (Gin is fully removed)
	// -----------------------------------------------------------------------
	chiRouter := chi.NewRouter()

	chiRouter.Use(middleware.CORS())
	chiRouter.Use(middleware.ErrorLogger())

	response.SetDebug(true)

	// Static files
	fileServer := http.FileServer(http.Dir(uploadCfg.LocalPath))
	chiRouter.Handle("/static/uploads/*", http.StripPrefix("/static/uploads", fileServer))

	// Swagger UI — served by chi via http-swagger
	chiRouter.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Public payment webhooks
	paymentHandler.RegisterPublicWebhookRoutes(chiRouter)

	chiRouter.Route("/api/v1", func(r chi.Router) {

		// ---------------------------------------------------------
		// Namespaces without overlaps
		// ---------------------------------------------------------

		r.Route("/admin", func(r chi.Router) {
			adminHandler.RegisterPublicRoutes(r)
			adminHandler.RegisterProtectedRoutes(r, jwtService)

			r.Group(func(r chi.Router) {
				r.Use(admin.ChiAdminJWTAuth(jwtService))
				r.Route("/leads", func(r chi.Router) {
					lead.RegisterAdminRoutes(r, leadHandler)
				})
			})
		})

		r.Route("/auth", func(r chi.Router) {
			authHandler.RegisterPublicRoutes(r)
		})

		r.Route("/leads", func(r chi.Router) {
			lead.RegisterPublicRoutes(r, leadHandler)
		})

		reviewer := reviewHandler
		r.Route("/reviews", func(r chi.Router) {
			reviewer.RegisterRoutes(r, nil) // public
			r.Group(func(r chi.Router) {    // protected
				r.Use(middleware.ChiJWTAuth(jwtService))
				reviewer.RegisterRoutes(nil, r)
			})
		})

		r.Route("/attachments", func(r chi.Router) {
			attachment.RegisterPublicRoutes(r, attachmentHandler)
			r.Group(func(r chi.Router) {
				r.Use(middleware.ChiJWTAuth(jwtService))
				attachment.RegisterProtectedRoutes(r, attachmentHandler)
			})
		})

		// ---------------------------------------------------------
		// Namespaces WITH overlaps and heavy protected sections
		// ---------------------------------------------------------

		r.Route("/studios", func(r chi.Router) {
			r.Group(func(r chi.Router) { catalogHandler.RegisterPublicStudioRoutes(r) })
			r.Group(func(r chi.Router) {
				r.Use(middleware.ChiJWTAuth(jwtService))
				catalogHandler.RegisterProtectedStudioRoutes(r, ownershipChecker)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.ChiJWTAuth(jwtService))
				r.Route("/{id}/bookings", func(r chi.Router) {
					bookingHandler.RegisterStudioRoutes(r, ownershipChecker)
				})
			})
		})

		r.Route("/rooms", func(r chi.Router) {
			r.Group(func(r chi.Router) { catalogHandler.RegisterPublicRoomRoutes(r) })
			r.Group(func(r chi.Router) {
				r.Use(middleware.ChiJWTAuth(jwtService))
				catalogHandler.RegisterProtectedRoomRoutes(r, ownershipChecker)
			})
		})

		r.Route("/room-types", func(r chi.Router) {
			r.Group(func(r chi.Router) { catalogHandler.RegisterPublicRoomTypes(r) })
		})

		r.Route("/users", func(r chi.Router) {
			r.Use(middleware.ChiJWTAuth(jwtService))
			authHandler.RegisterProtectedRoutes(r)
		})

		r.Route("/booking", func(r chi.Router) {
			r.Use(middleware.ChiJWTAuth(jwtService))
			bookingHandler.RegisterRoutes(r)
		})

		r.Route("/subscriptions", func(r chi.Router) {
			r.Group(func(r chi.Router) { subscription.RegisterPublicRoutes(r, subscriptionHandler) })

			r.Group(func(r chi.Router) {
				r.Use(middleware.ChiJWTAuth(jwtService))
				r.Post("/", paymentHandler.CreateSubscription)
				r.Get("/me", paymentHandler.MySubscription)
				r.Post("/cancel", paymentHandler.CancelSubscription)
			})
		})

		r.Route("/owner", func(r chi.Router) {
			r.Use(middleware.ChiJWTAuth(jwtService))
			r.Use(middleware.ChiRequireRole(string(auth.RoleStudioOwner)))
			ownerHandler.RegisterRoutes(r)
			r.Route("/subscription", func(r chi.Router) {
				subscription.RegisterOwnerRoutes(r, subscriptionHandler)
			})
		})

		r.Route("/company", func(r chi.Router) {
			r.Use(middleware.ChiJWTAuth(jwtService))
			r.Use(middleware.ChiRequireRole(string(auth.RoleStudioOwner)))
			ownerHandler.RegisterCompanyRoutes(r)
		})

		r.Route("/manager", func(r chi.Router) {
			r.Use(middleware.ChiJWTAuth(jwtService))
			r.Use(middleware.ChiRequireRole(string(auth.RoleStudioOwner)))
			managerHandler.RegisterRoutes(r)
		})

		r.Route("/payments/robokassa", func(r chi.Router) {
			r.Use(middleware.ChiJWTAuth(jwtService))
			r.Post("/create", paymentHandler.CreatePayment)
			r.Post("/init", paymentHandler.InitPayment)
		})

		// ---------------------------------------------------------
		// Other Protected Namespaces
		// ---------------------------------------------------------
		r.Group(func(r chi.Router) {
			r.Use(middleware.ChiJWTAuth(jwtService))

			r.Route("/profiles", func(r chi.Router) {
				profile.RegisterRoutes(r, clientProfileHandler, ownerProfileHandler, adminProfileHandler)
			})
			r.Route("/chat", func(r chi.Router) { chat.RegisterRoutes(r, chatHandler) })
			r.Route("/notifications/read", func(r chi.Router) {
				// Special root alias to handle /notifications/read/all according to Swagger
				r.Post("/all", notificationHandler.MarkAllAsRead)
			})
			r.Route("/notifications", func(r chi.Router) {
				notification.RegisterRoutes(r, notificationHandler, preferencesHandler, deviceTokensHandler)
			})
			r.Route("/favorites", func(r chi.Router) { favoriteHandler.RegisterRoutes(r) })
			r.Route("/uploads", func(r chi.Router) { upload.RegisterRoutes(r, uploadHandler) })
			r.Route("/relationships", func(r chi.Router) { relationship.RegisterRoutes(r, relationshipHandler) })
		})
	})

	// Internal chi routes (mwork sync)
	chiRouter.Route("/internal", func(r chi.Router) {
		r.Use(middleware.ChiInternalTokenAuth)
		r.Route("/mwork", func(r chi.Router) { mworkHandler.RegisterRoutes(r) })

		// MWork-authenticated booking + catalog access
		r.Group(func(r chi.Router) {
			r.Use(middleware.ChiMWorkUserAuth(userRepo))
			r.Post("/mwork/bookings", bookingHandler.CreateBooking)
			r.Get("/mwork/bookings", bookingHandler.GetMyBookings)
			r.Get("/mwork/studios", catalogHandler.GetStudios)
			r.Get("/mwork/studios/{id}", catalogHandler.GetStudioByID)
			r.Get("/mwork/rooms/{id}/availability", bookingHandler.GetRoomAvailability)
			r.Get("/mwork/rooms/{id}/busy-slots", bookingHandler.GetBusySlots)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Printf("🚀 Server starting on :%s", port)

	if err := http.ListenAndServe(":"+port, chiRouter); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
