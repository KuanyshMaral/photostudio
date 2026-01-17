package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"photostudio/internal/database"
	"photostudio/internal/domain"

	"golang.org/x/crypto/bcrypt"
	_ "gorm.io/gorm"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	db, err := database.Connect("studio.db")
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}

	// AutoMigrate to ensure schema is up to date
	log.Println("Running AutoMigrate...")
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.StudioOwner{},
		&domain.Studio{},
		&domain.Room{},
		&domain.Equipment{},
		&domain.Booking{},
		&domain.Review{},
		&domain.Notification{},
	); err != nil {
		log.Fatal("AutoMigrate failed:", err)
	}

	// Cleanup old data (in safe order to avoid foreign key errors)
	log.Println("Cleaning old data...")
	db.Exec("DELETE FROM notifications")
	db.Exec("DELETE FROM reviews")
	db.Exec("DELETE FROM bookings")
	db.Exec("DELETE FROM equipment")
	db.Exec("DELETE FROM rooms")
	db.Exec("DELETE FROM studios")
	db.Exec("DELETE FROM studio_owners")
	db.Exec("DELETE FROM users")

	// ================== USERS ==================
	log.Println("Creating users...")

	// Admin
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	admin := domain.User{
		Email:         "admin@photostudio.kz",
		PasswordHash:  string(adminHash),
		Role:          domain.RoleAdmin,
		Name:          "Администратор",
		EmailVerified: true,
	}
	db.Create(&admin)
	log.Println("Admin created: admin@photostudio.kz / admin123")

	// Clients (3 users)
	clients := []domain.User{}
	clientEmails := []string{"asel@mail.kz", "bekzat@gmail.com", "dina@yandex.kz"}
	for i, email := range clientEmails {
		hash, _ := bcrypt.GenerateFromPassword([]byte("client123"), bcrypt.DefaultCost)
		client := domain.User{
			Email:         email,
			PasswordHash:  string(hash),
			Role:          domain.RoleClient,
			Name:          fmt.Sprintf("Клиент %d", i+1),
			Phone:         fmt.Sprintf("+7 777 123 45%02d", i+67),
			EmailVerified: true,
		}
		db.Create(&client)
		clients = append(clients, client)
	}

	// Studio Owners (3 users)
	owners := []domain.User{}
	ownerEmails := []string{"aidar@lightpro.kz", "gulnaz@creativespace.kz", "yerlan@fashionstudio.kz"}
	for i, email := range ownerEmails {
		hash, _ := bcrypt.GenerateFromPassword([]byte("owner123"), bcrypt.DefaultCost)
		owner := domain.User{
			Email:         email,
			PasswordHash:  string(hash),
			Role:          domain.RoleStudioOwner,
			Name:          fmt.Sprintf("Владелец %d", i+1),
			StudioStatus:  "verified", // or "pending" for one
			EmailVerified: true,
		}
		db.Create(&owner)
		owners = append(owners, owner)

		// StudioOwner details
		db.Create(&domain.StudioOwner{
			UserID:      owner.ID,
			CompanyName: fmt.Sprintf("Studio Company %d", i+1),
			BIN:         fmt.Sprintf("1234567890%02d", i+12),
		})
	}

	// ================== STUDIOS ==================
	log.Println("Creating studios...")
	studios := make([]domain.Studio, 0, 5)
	for i := 0; i < 5; i++ {
		owner := owners[i%len(owners)]
		studio := domain.Studio{
			OwnerID:      owner.ID,
			Name:         fmt.Sprintf("Studio %d Pro", i+1),
			Description:  "Профессиональная студия с современным оборудованием",
			Address:      fmt.Sprintf("ул. Тестовая %d", i+100),
			District:     "Центральный",
			City:         "Алматы",
			Rating:       4.0 + rand.Float64()*1.0,
			TotalReviews: rand.Intn(100),
			Phone:        fmt.Sprintf("+7 727 000 00%02d", i),
			Photos:       []string{fmt.Sprintf("/static/studios/test%d.jpg", i)},
			WorkingHours: map[string]domain.DaySchedule{
				"monday": {Open: "09:00", Close: "22:00"},
				// add other days if needed
			},
		}
		db.Create(&studio)
		studios = append(studios, studio)
	}

	// ================== ROOMS ==================
	log.Println("Creating rooms...")
	for _, studio := range studios {
		for j := 1; j <= 3; j++ {
			room := domain.Room{
				StudioID:        studio.ID,
				Name:            fmt.Sprintf("Зал %d", j),
				Description:     "Комфортный зал для съёмок",
				AreaSqm:         40 + rand.Intn(40),
				Capacity:        5 + rand.Intn(10),
				RoomType:        domain.ValidRoomTypes()[rand.Intn(len(domain.ValidRoomTypes()))],
				PricePerHourMin: 5000 + float64(rand.Intn(10000)),
				IsActive:        true,
			}
			db.Create(&room)
		}
	}

	// ================== BOOKINGS ==================
	log.Println("Creating bookings...")
	for i := 0; i < 10; i++ {
		studio := studios[rand.Intn(len(studios))]
		client := clients[rand.Intn(len(clients))]
		roomID := int64(rand.Intn(3)+1) + studio.ID*3 // approximate room ID for studio

		days := rand.Intn(30) - 15 // -15 to +15 days
		startHour := 9 + rand.Intn(12)
		duration := 1 + rand.Intn(3)

		start := time.Now().AddDate(0, 0, days).Truncate(24 * time.Hour).Add(time.Duration(startHour) * time.Hour)
		end := start.Add(time.Duration(duration) * time.Hour)

		booking := domain.Booking{
			RoomID:        roomID,
			StudioID:      studio.ID,
			UserID:        client.ID,
			StartTime:     start,
			EndTime:       end,
			TotalPrice:    float64(duration) * 5000, // approximate price
			Status:        domain.BookingStatus([]string{"pending", "confirmed", "completed"}[rand.Intn(3)]),
			PaymentStatus: domain.PaymentStatus([]string{"unpaid", "paid"}[rand.Intn(2)]),
			Notes:         fmt.Sprintf("Бронирование %d", i+1),
		}
		db.Create(&booking)
	}

	// ================== REVIEWS ==================
	log.Println("Creating reviews...")
	for i := 0; i < 5; i++ {
		studio := studios[rand.Intn(len(studios))]
		client := clients[rand.Intn(len(clients))]

		review := domain.Review{
			StudioID: studio.ID,
			UserID:   client.ID,
			Rating:   3 + rand.Intn(3),
			Comment:  fmt.Sprintf("Отличная студия! Рекомендую %d", i+1),
		}
		db.Create(&review)
	}

	// ================== NOTIFICATIONS ==================
	log.Println("Creating notifications...")
	for _, owner := range owners {
		db.Create(&domain.Notification{
			UserID:  owner.ID,
			Type:    domain.NotifVerificationApproved,
			Title:   "Студия верифицирована",
			Message: "Ваша студия готова к работе!",
			IsRead:  rand.Intn(2) == 0,
		})
	}

	log.Println("🎉 Seed completed!")
	log.Println("Test accounts:")
	log.Println("Admin: admin@photostudio.kz / admin123")
	log.Println("Clients: client1@test.com ... client3@test.com / client123")
	log.Println("Owners: owner1@studio.kz ... owner3@studio.kz / owner123")
}
