package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"photostudio/internal/database"
	"photostudio/internal/modules/auth"
	"photostudio/internal/modules/booking"
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
	j := jwtsvc.New(secret, 24*time.Hour)

	authService := auth.NewService(userRepo, j)
	authHandler := auth.NewHandler(authService)

	bookingRepo := repository.NewBookingRepository(db)
	roomRepo := repository.NewRoomRepository(db)

	bookingService := booking.NewService(bookingRepo, roomRepo)
	bookingHandler := booking.NewHandler(bookingService)

	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		authHandler.RegisterRoutes(v1)
		bookingHandler.RegisterRoutes(v1)
	}

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
