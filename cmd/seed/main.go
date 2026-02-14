package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"photostudio/internal/database"
	"photostudio/internal/domain/auth"
	"photostudio/internal/domain/booking"
	"photostudio/internal/domain/catalog"
	"photostudio/internal/domain/notification"
	"photostudio/internal/domain/owner"
	"photostudio/internal/domain/review"
	"time"
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
		&auth.User{},
		&owner.StudioOwner{},
		&catalog.Studio{},
		&catalog.Room{},
		&catalog.Equipment{},
		&booking.Booking{},
		&review.Review{},
		&notification.Notification{},
		&catalog.StudioWorkingHours{},
	); err != nil {
		log.Fatal("AutoMigrate failed:", err)
	}

	// Cleanup old data (in safe order to avoid foreign key errors)
	log.Println("Cleaning old data...")
	db.Exec("DELETE FROM studio_working_hours")
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
	admin := auth.User{
		Email:         "admin@photostudio.kz",
		PasswordHash:  string(adminHash),
		Role:          auth.RoleAdmin,
		Name:          "Администратор",
		EmailVerified: true,
	}
	if err := db.Create(&admin).Error; err != nil {
		log.Fatal("Failed to create admin:", err)
	}
	log.Println("Admin created: admin@photostudio.kz / admin123")

	// Clients (3 users)
	clients := []auth.User{}
	clientEmails := []string{"asel@mail.kz", "bekzat@gmail.com", "dina@yandex.kz"}
	for i, email := range clientEmails {
		hash, _ := bcrypt.GenerateFromPassword([]byte("client123"), bcrypt.DefaultCost)
		client := auth.User{
			Email:         email,
			PasswordHash:  string(hash),
			Role:          auth.RoleClient,
			Name:          fmt.Sprintf("Клиент %d", i+1),
			Phone:         fmt.Sprintf("+7 777 123 45%02d", i+67),
			EmailVerified: true,
		}
		if err := db.Create(&client).Error; err != nil {
			log.Fatal("Failed to create client:", err)
		}
		clients = append(clients, client)
	}

	// Studio Owners (3 users)
	owners := []auth.User{}
	ownerEmails := []string{"aidar@lightpro.kz", "gulnaz@creativespace.kz", "yerlan@fashionstudio.kz"}
	for i, email := range ownerEmails {
		hash, _ := bcrypt.GenerateFromPassword([]byte("owner123"), bcrypt.DefaultCost)
		u := auth.User{
			Email:         email,
			PasswordHash:  string(hash),
			Role:          auth.RoleStudioOwner,
			Name:          fmt.Sprintf("Владелец %d", i+1),
			StudioStatus:  "verified",
			EmailVerified: true,
		}
		if err := db.Create(&u).Error; err != nil {
			log.Fatal("Failed to create owner:", err)
		}
		owners = append(owners, u)

		// StudioOwner details
		studioOwner := owner.StudioOwner{
			UserID:      u.ID,
			CompanyName: fmt.Sprintf("Studio Company %d", i+1),
			BIN:         fmt.Sprintf("1234567890%02d", i+12),
		}
		if err := db.Create(&studioOwner).Error; err != nil {
			log.Fatal("Failed to create studio owner details:", err)
		}
	}

	// ================== STUDIOS ==================
	log.Println("Creating studios...")
	studios := make([]catalog.Studio, 0, 5)
	for i := 0; i < 5; i++ {
		ownerIdx := i % len(owners)
		studio := catalog.Studio{
			OwnerID:      owners[ownerIdx].ID,
			Name:         fmt.Sprintf("Studio %d Pro", i+1),
			Description:  "Профессиональная студия с современным оборудованием",
			Address:      fmt.Sprintf("ул. Тестовая %d", i+100),
			District:     "Центральный",
			City:         "Алматы",
			Rating:       4.0 + rand.Float64()*1.0,
			TotalReviews: rand.Intn(100),
			Phone:        fmt.Sprintf("+7 727 000 00%02d", i),
			Photos:       []string{fmt.Sprintf("/static/studios/test%d.jpg", i)},
		}

		// Сохраняем студию и получаем её ID
		if err := db.Create(&studio).Error; err != nil {
			log.Fatal("Failed to create studio:", err)
		}

		// Перезагружаем студию из БД, чтобы убедиться что все поля заполнены
		var loadedStudio catalog.Studio
		if err := db.First(&loadedStudio, studio.ID).Error; err != nil {
			log.Fatal("Failed to load studio after creation:", err)
		}

		studios = append(studios, loadedStudio)
		log.Printf("Created studio: %s (ID: %d, OwnerID: %d)", loadedStudio.Name, loadedStudio.ID, loadedStudio.OwnerID)
	}

	// ================== WORKING HOURS ==================
	log.Println("Creating working hours...")

	for _, studio := range studios {
		// Создаем структурированные рабочие часы
		workingHours := []catalog.WorkingHours{
			{DayOfWeek: 0, OpenTime: "00:00", CloseTime: "00:00", IsClosed: true},  // Вс
			{DayOfWeek: 1, OpenTime: "10:00", CloseTime: "20:00", IsClosed: false}, // Пн
			{DayOfWeek: 2, OpenTime: "10:00", CloseTime: "20:00", IsClosed: false}, // Вт
			{DayOfWeek: 3, OpenTime: "10:00", CloseTime: "20:00", IsClosed: false}, // Ср
			{DayOfWeek: 4, OpenTime: "10:00", CloseTime: "20:00", IsClosed: false}, // Чт
			{DayOfWeek: 5, OpenTime: "10:00", CloseTime: "20:00", IsClosed: false}, // Пт
			{DayOfWeek: 6, OpenTime: "12:00", CloseTime: "18:00", IsClosed: false}, // Сб
		}

		studioHours := &catalog.StudioWorkingHours{
			StudioID: studio.ID,
			Hours:    workingHours, // Прямой массив WorkingHours
		}

		if err := db.Create(studioHours).Error; err != nil {
			log.Fatal("Failed to create studio working hours:", err)
		}

		log.Printf("Created working hours for studio ID: %d", studio.ID)
	}

	log.Println("✅ Working hours created")

	// ================== ROOMS ==================
	log.Println("Creating rooms...")
	allRooms := []catalog.Room{}
	for _, studio := range studios {
		for j := 1; j <= 3; j++ {
			room := catalog.Room{
				StudioID:        studio.ID,
				Name:            fmt.Sprintf("Зал %d", j),
				Description:     "Комфортный зал для съёмок",
				AreaSqm:         40 + rand.Intn(40),
				Capacity:        5 + rand.Intn(10),
				RoomType:        catalog.ValidRoomTypes()[rand.Intn(len(catalog.ValidRoomTypes()))],
				PricePerHourMin: 5000 + float64(rand.Intn(10000)),
				IsActive:        true,
			}
			if err := db.Create(&room).Error; err != nil {
				log.Fatal("Failed to create room:", err)
			}
			allRooms = append(allRooms, room)
		}
	}

	// ================== BOOKINGS ==================
	log.Println("Creating bookings...")
	for i := 0; i < 10; i++ {
		studio := studios[rand.Intn(len(studios))]
		client := clients[rand.Intn(len(clients))]

		// Найдем комнату, принадлежащую этой студии
		var studioRooms []catalog.Room
		if err := db.Where("studio_id = ?", studio.ID).Find(&studioRooms).Error; err != nil || len(studioRooms) == 0 {
			log.Println("No rooms found for studio, skipping booking")
			continue
		}

		room := studioRooms[rand.Intn(len(studioRooms))]

		days := rand.Intn(30) - 15 // -15 to +15 days
		startHour := 9 + rand.Intn(12)
		duration := 1 + rand.Intn(3)

		start := time.Now().AddDate(0, 0, days).Truncate(24 * time.Hour).Add(time.Duration(startHour) * time.Hour)
		end := start.Add(time.Duration(duration) * time.Hour)

		booking := booking.Booking{
			RoomID:        room.ID,
			StudioID:      studio.ID,
			UserID:        client.ID,
			StartTime:     start,
			EndTime:       end,
			TotalPrice:    float64(duration) * 5000,
			Status:        booking.BookingStatus([]string{"pending", "confirmed", "completed"}[rand.Intn(3)]),
			PaymentStatus: booking.PaymentStatus([]string{"unpaid", "paid"}[rand.Intn(2)]),
			Notes:         fmt.Sprintf("Бронирование %d", i+1),
		}
		if err := db.Create(&booking).Error; err != nil {
			log.Println("Failed to create booking:", err)
		}
	}

	// ================== DEMO USER ==================
	log.Println("Creating demo user...")

	// Создаем демо пользователя
	demoHash, _ := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
	demoUser := auth.User{
		Email:         "demo@studiobooking.kz",
		PasswordHash:  string(demoHash),
		Name:          "Алексей Петров",
		Role:          auth.RoleClient,
		EmailVerified: true,
	}

	if err := db.Create(&demoUser).Error; err != nil {
		// Если пользователь уже существует, обновим его
		log.Println("Demo user already exists, updating...")
	}

	// Создаем несколько бронирований для демо пользователя
	if len(allRooms) >= 3 {
		createDemoBooking(db, demoUser.ID, allRooms[0], "2026-01-05", "10:00", "14:00", "completed", 20000)
		createDemoBooking(db, demoUser.ID, allRooms[1], "2026-01-10", "12:00", "16:00", "completed", 24000)
		createDemoBooking(db, demoUser.ID, allRooms[2], "2026-01-15", "09:00", "12:00", "completed", 15000)
		createDemoBooking(db, demoUser.ID, allRooms[0], "2026-01-25", "10:00", "14:00", "confirmed", 20000)
		createDemoBooking(db, demoUser.ID, allRooms[1], "2026-01-28", "14:00", "18:00", "confirmed", 28000)
		createDemoBooking(db, demoUser.ID, allRooms[2], "2026-01-22", "11:00", "15:00", "pending", 24000)
		createDemoBooking(db, demoUser.ID, allRooms[0], "2026-01-08", "10:00", "12:00", "cancelled", 10000)
	}

	// ================== REVIEWS ==================
	log.Println("Creating reviews...")
	for i := 0; i < 5; i++ {
		studio := studios[rand.Intn(len(studios))]
		client := clients[rand.Intn(len(clients))]

		review := review.Review{
			StudioID: studio.ID,
			UserID:   client.ID,
			Rating:   3 + rand.Intn(3),
			Comment:  fmt.Sprintf("Отличная студия! Рекомендую %d", i+1),
		}
		if err := db.Create(&review).Error; err != nil {
			log.Println("Failed to create review:", err)
		}
	}

	// ================== NOTIFICATIONS ==================
	log.Println("Creating notifications...")
	for _, owner := range owners {
		notification := notification.Notification{
			UserID:  owner.ID,
			Type:    notification.NotifVerificationApproved,
			Title:   "Студия верифицирована",
			Message: "Ваша студия готова к работе!",
			IsRead:  rand.Intn(2) == 0,
		}
		if err := db.Create(&notification).Error; err != nil {
			log.Println("Failed to create notification:", err)
		}
	}

	log.Println("🎉 Seed completed!")
	log.Println("\nTest accounts:")
	log.Println("Admin: admin@photostudio.kz / admin123")
	log.Println("Clients: asel@mail.kz / client123, bekzat@gmail.com / client123, dina@yandex.kz / client123")
	log.Println("Owners: aidar@lightpro.kz / owner123, gulnaz@creativespace.kz / owner123, yerlan@fashionstudio.kz / owner123")
	log.Println("Demo: demo@studiobooking.kz / demo123")
	log.Printf("\nCreated %d studios, %d rooms, %d bookings", len(studios), len(allRooms), 10+7)
}

func createDemoBooking(db *gorm.DB, userID int64, room catalog.Room, dateStr, startStr, endStr, status string, price float64) {
	date, _ := time.Parse("2006-01-02", dateStr)
	startTime, _ := time.Parse("15:04", startStr)
	endTime, _ := time.Parse("15:04", endStr)

	start := time.Date(date.Year(), date.Month(), date.Day(), startTime.Hour(), startTime.Minute(), 0, 0, time.Local)
	end := time.Date(date.Year(), date.Month(), date.Day(), endTime.Hour(), endTime.Minute(), 0, 0, time.Local)

	booking := booking.Booking{
		RoomID:        room.ID,
		StudioID:      room.StudioID,
		UserID:        userID,
		StartTime:     start,
		EndTime:       end,
		TotalPrice:    price,
		Status:        booking.BookingStatus(status),
		PaymentStatus: booking.PaymentPaid, // Исправлено с PaymentStatusPaid на PaymentPaid
		Notes:         "Demo booking",
	}

	db.Create(&booking)
}