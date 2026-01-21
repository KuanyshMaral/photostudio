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
	"gorm.io/gorm/clause"
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

	// ================== DEMO USER BOOKINGS (for stats) ==================
	log.Println("Creating demo user booking history (user_id=1)...")

	// 1) Ensure demo user exists with id=1
	demoUser := domain.User{
		ID:            1,
		Email:         "demo@studiobooking.kz",
		PasswordHash:  "$2a$10$dummyhashdummyhashdummyhashdummyhashdummyhash", // replace if you have real one
		Name:          "Алексей Петров",
		Role:          domain.RoleClient,
		EmailVerified: true,
		CreatedAt:     time.Date(2025, 6, 15, 10, 0, 0, 0, time.Local),
		UpdatedAt:     time.Now(),
	}

	// Upsert by primary key ID
	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"email", "password_hash", "name", "role", "email_verified", "created_at", "updated_at"}),
	}).Create(&demoUser)

	// 2) Delete previous demo bookings
	db.Where("user_id = ?", demoUser.ID).Delete(&domain.Booking{})

	// 3) Pick real rooms from DB (avoid fake roomID math)
	var rooms []domain.Room
	if err := db.Order("id ASC").Limit(3).Find(&rooms).Error; err != nil || len(rooms) == 0 {
		log.Println("⚠️ No rooms found for demo bookings. Skipping demo booking history.")
	} else {
		// helper to choose room safely (repeat if меньше 3)
		getRoom := func(idx int) domain.Room {
			return rooms[idx%len(rooms)]
		}

		// Dates like in assignment example relative to *today*
		// Past completed (3)
		completed1Room := getRoom(0)
		completed2Room := getRoom(1)
		completed3Room := getRoom(0)

		// Future confirmed (2) + future pending (1)
		confirmed1Room := getRoom(0)
		confirmed2Room := getRoom(2)
		pendingRoom := getRoom(1)

		// Cancelled (1)
		cancelledRoom := getRoom(0)

		// Helper to create booking quickly
		create := func(room domain.Room, start, end time.Time, status domain.BookingStatus, total float64, createdAt time.Time) {
			b := domain.Booking{
				RoomID:        room.ID,
				StudioID:      room.StudioID,
				UserID:        demoUser.ID,
				StartTime:     start,
				EndTime:       end,
				TotalPrice:    total,
				Status:        status,
				PaymentStatus: domain.PaymentStatus("paid"),
				Notes:         "Demo booking",
				CreatedAt:     createdAt,
				UpdatedAt:     createdAt,
			}
			db.Create(&b)
		}

		// === Completed (3) in the past ===
		create(
			completed1Room,
			time.Date(2026, 1, 5, 10, 0, 0, 0, time.Local),
			time.Date(2026, 1, 5, 14, 0, 0, 0, time.Local),
			domain.BookingStatus("completed"),
			20000,
			time.Date(2026, 1, 4, 12, 0, 0, 0, time.Local),
		)

		create(
			completed2Room,
			time.Date(2026, 1, 10, 12, 0, 0, 0, time.Local),
			time.Date(2026, 1, 10, 16, 0, 0, 0, time.Local),
			domain.BookingStatus("completed"),
			24000,
			time.Date(2026, 1, 9, 15, 0, 0, 0, time.Local),
		)

		create(
			completed3Room,
			time.Date(2026, 1, 15, 9, 0, 0, 0, time.Local),
			time.Date(2026, 1, 15, 12, 0, 0, 0, time.Local),
			domain.BookingStatus("completed"),
			15000,
			time.Date(2026, 1, 14, 10, 0, 0, 0, time.Local),
		)

		// === Upcoming confirmed (2) ===
		create(
			confirmed1Room,
			time.Date(2026, 1, 25, 10, 0, 0, 0, time.Local),
			time.Date(2026, 1, 25, 14, 0, 0, 0, time.Local),
			domain.BookingStatus("confirmed"),
			20000,
			time.Date(2026, 1, 19, 12, 0, 0, 0, time.Local),
		)

		create(
			confirmed2Room,
			time.Date(2026, 1, 28, 14, 0, 0, 0, time.Local),
			time.Date(2026, 1, 28, 18, 0, 0, 0, time.Local),
			domain.BookingStatus("confirmed"),
			28000,
			time.Date(2026, 1, 19, 14, 0, 0, 0, time.Local),
		)

		// === Pending (1) ===
		create(
			pendingRoom,
			time.Date(2026, 1, 22, 11, 0, 0, 0, time.Local),
			time.Date(2026, 1, 22, 15, 0, 0, 0, time.Local),
			domain.BookingStatus("pending"),
			24000,
			time.Date(2026, 1, 19, 9, 0, 0, 0, time.Local),
		)

		// === Cancelled (1) ===
		create(
			cancelledRoom,
			time.Date(2026, 1, 8, 10, 0, 0, 0, time.Local),
			time.Date(2026, 1, 8, 12, 0, 0, 0, time.Local),
			domain.BookingStatus("cancelled"),
			10000,
			time.Date(2026, 1, 7, 10, 0, 0, 0, time.Local),
		)

		log.Println("✅ Demo booking history created (total=7, upcoming=2, completed=3, cancelled=1)")
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
