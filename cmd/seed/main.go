package main

import (
	"fmt"
	"log"
	"math/rand"
	"photostudio/internal/database"
	"photostudio/internal/domain"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Use modern approach: create a new source and rand instance
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	db, err := database.Connect("studio.db")
	if err != nil {
		log.Fatal(err)
	}

	// Auto migrate all models
	db.AutoMigrate(
		&domain.User{},
		&domain.StudioOwner{},
		&domain.Studio{},
		&domain.Room{},
		&domain.Equipment{},
		&domain.Booking{},
		&domain.Review{},
		&domain.Notification{},
	)

	// Clear existing data (optional - for clean seed)
	log.Println("🗑️  Clearing existing data...")
	db.Exec("DELETE FROM reviews")
	db.Exec("DELETE FROM bookings")
	db.Exec("DELETE FROM equipment")
	db.Exec("DELETE FROM rooms")
	db.Exec("DELETE FROM studios")
	db.Exec("DELETE FROM studio_owners")
	db.Exec("DELETE FROM users")

	log.Println("👤 Creating users...")

	// Helper function to hash passwords
	hashPassword := func(password string) string {
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		return string(hash)
	}

	// ================= ADMIN =================
	admin := domain.User{
		Email:         "admin@studiobooking.kz",
		PasswordHash:  hashPassword("admin123"),
		Role:          domain.RoleAdmin,
		Name:          "Администратор Системы",
		Phone:         "+7 727 100 0000",
		EmailVerified: true,
	}
	db.Create(&admin)
	log.Println("  ✅ Admin created")

	// ================= STUDIO OWNERS =================
	owners := []domain.User{
		{Email: "owner1@studio.kz", PasswordHash: hashPassword("owner123"), Role: domain.RoleStudioOwner, Name: "Алексей Петров", Phone: "+7 727 111 1111", StudioStatus: domain.StatusVerified, EmailVerified: true},
		{Email: "owner2@studio.kz", PasswordHash: hashPassword("owner123"), Role: domain.RoleStudioOwner, Name: "Мария Иванова", Phone: "+7 727 222 2222", StudioStatus: domain.StatusVerified, EmailVerified: true},
		{Email: "owner3@studio.kz", PasswordHash: hashPassword("owner123"), Role: domain.RoleStudioOwner, Name: "Дмитрий Сидоров", Phone: "+7 727 333 3333", StudioStatus: domain.StatusPending, EmailVerified: true},
		{Email: "owner4@studio.kz", PasswordHash: hashPassword("owner123"), Role: domain.RoleStudioOwner, Name: "Елена Козлова", Phone: "+7 727 444 4444", StudioStatus: domain.StatusVerified, EmailVerified: true},
	}
	for i := range owners {
		db.Create(&owners[i])
	}
	log.Printf("  ✅ %d Studio owners created", len(owners))

	// Create StudioOwner details for verified owners
	studioOwners := []domain.StudioOwner{
		{UserID: owners[0].ID, CompanyName: "Light Studio Pro LLC", BIN: "123456789001", ContactPerson: "Алексей Петров"},
		{UserID: owners[1].ID, CompanyName: "Creative Space LLP", BIN: "123456789002", ContactPerson: "Мария Иванова"},
		{UserID: owners[2].ID, CompanyName: "Fashion Studio Inc", BIN: "123456789003", ContactPerson: "Дмитрий Сидоров"},
		{UserID: owners[3].ID, CompanyName: "Portrait Lab LLP", BIN: "123456789004", ContactPerson: "Елена Козлова"},
	}
	for i := range studioOwners {
		db.Create(&studioOwners[i])
	}

	// ================= CLIENTS =================
	clients := []domain.User{}
	clientNames := []string{
		"Анна Смирнова", "Иван Кузнецов", "Ольга Попова", "Сергей Волков",
		"Наталья Соколова", "Андрей Лебедев", "Екатерина Морозова", "Павел Новиков",
		"Татьяна Павлова", "Михаил Семёнов",
	}

	for i := 1; i <= 10; i++ {
		client := domain.User{
			Email:         fmt.Sprintf("client%d@test.com", i),
			PasswordHash:  hashPassword("client123"),
			Role:          domain.RoleClient,
			Name:          clientNames[i-1],
			Phone:         fmt.Sprintf("+7 777 %03d %02d%02d", rng.Intn(1000), rng.Intn(100), rng.Intn(100)),
			EmailVerified: true,
		}
		db.Create(&client)
		clients = append(clients, client)
	}
	log.Printf("  ✅ %d Clients created", len(clients))

	// ================= STUDIOS =================
	log.Println("🏢 Creating studios...")
	studios := []domain.Studio{
		{
			OwnerID:      owners[0].ID,
			Name:         "Light Studio Pro",
			Description:  "Профессиональная фотостудия с современным оборудованием. Три зала различной стилистики для любых видов съёмок.",
			Address:      "ул. Абая, 150",
			City:         "Алматы",
			District:     "Алмалинский",
			Rating:       4.8,
			TotalReviews: 0, // Will be updated after reviews
			Phone:        "+7 727 123 4567",
		},
		{
			OwnerID:      owners[0].ID,
			Name:         "Creative Space",
			Description:  "Креативное пространство для фотосессий и видеосъёмок. Лофт стиль, высокие потолки, естественный свет.",
			Address:      "пр. Достык, 89",
			City:         "Алматы",
			District:     "Медеуский",
			Rating:       4.5,
			TotalReviews: 0,
			Phone:        "+7 727 234 5678",
		},
		{
			OwnerID:      owners[1].ID,
			Name:         "Fashion Studio",
			Description:  "Специализируемся на fashion съёмках. Профессиональный свет Broncolor, циклорама, гримёрка.",
			Address:      "ул. Сатпаева, 22",
			City:         "Алматы",
			District:     "Бостандыкский",
			Rating:       4.9,
			TotalReviews: 0,
			Phone:        "+7 727 345 6789",
		},
		{
			OwnerID:      owners[3].ID,
			Name:         "Portrait Lab",
			Description:  "Уютная студия для портретной съёмки. Естественный свет, минималистичный интерьер.",
			Address:      "ул. Жандосова, 55",
			City:         "Алматы",
			District:     "Ауэзовский",
			Rating:       4.6,
			TotalReviews: 0,
			Phone:        "+7 727 456 7890",
		},
		{
			OwnerID:      owners[3].ID,
			Name:         "Commercial Studio",
			Description:  "Большая студия для коммерческих съёмок. Циклорама 6x4м, профессиональное оборудование.",
			Address:      "ул. Розыбакиева, 100",
			City:         "Алматы",
			District:     "Алмалинский",
			Rating:       4.7,
			TotalReviews: 0,
			Phone:        "+7 727 567 8901",
		},
	}

	for i := range studios {
		db.Create(&studios[i])
	}
	log.Printf("  ✅ %d Studios created", len(studios))

	// ================= ROOMS =================
	log.Println("🏠 Creating rooms...")
	roomTypes := []domain.RoomType{domain.RoomFashion, domain.RoomPortrait, domain.RoomCreative, domain.RoomCommercial}
	roomNames := []string{"Белый зал", "Чёрный зал", "Лофт зал", "Циклорама", "Natural Light"}
	roomDescriptions := []string{
		"Просторный зал с профессиональным освещением и белым фоном",
		"Зал с драматичным освещением и чёрным фоном для контрастных съёмок",
		"Индустриальный лофт с кирпичными стенами и высокими потолками",
		"Зал с циклорамой для съёмки продукции и портретов",
		"Студия с большими окнами и естественным освещением",
	}

	var rooms []domain.Room
	for _, studio := range studios {
		numRooms := 3
		for j := 0; j < numRooms; j++ {
			room := domain.Room{
				StudioID:        studio.ID,
				Name:            roomNames[rng.Intn(len(roomNames))],
				Description:     roomDescriptions[rng.Intn(len(roomDescriptions))],
				AreaSqm:         int(float64(30 + rng.Intn(50))),
				Capacity:        5 + rng.Intn(10),
				RoomType:        roomTypes[rng.Intn(len(roomTypes))],
				PricePerHourMin: float64(5000 + rng.Intn(10000)),
				IsActive:        true,
			}
			db.Create(&room)
			rooms = append(rooms, room)
		}
	}
	log.Printf("  ✅ %d Rooms created", len(rooms))

	// ================= BOOKINGS =================
	log.Println("📅 Creating bookings...")
	statuses := []domain.BookingStatus{domain.BookingPending, domain.BookingConfirmed, domain.BookingCompleted}
	paymentStatuses := []domain.PaymentStatus{domain.PaymentUnpaid, domain.PaymentPaid}

	var bookings []domain.Booking
	for i := 0; i < 50; i++ {
		daysOffset := rng.Intn(60) - 30 // от -30 до +30 дней
		startHour := 10 + rng.Intn(8)   // 10:00 - 17:00
		duration := 1 + rng.Intn(3)     // 1-3 hours

		startTime := time.Now().AddDate(0, 0, daysOffset).Truncate(24 * time.Hour).Add(time.Duration(startHour) * time.Hour)
		endTime := startTime.Add(time.Duration(duration) * time.Hour)

		room := rooms[rng.Intn(len(rooms))]
		client := clients[rng.Intn(len(clients))]

		status := statuses[rng.Intn(len(statuses))]
		paymentStatus := paymentStatuses[rng.Intn(len(paymentStatuses))]

		// For past bookings - mark as completed and paid
		if daysOffset < 0 {
			status = domain.BookingCompleted
			paymentStatus = domain.PaymentPaid
		}

		// Get studio ID from room
		var studioID int64
		db.Model(&domain.Room{}).Select("studio_id").Where("id = ?", room.ID).Scan(&studioID)

		booking := domain.Booking{
			RoomID:        room.ID,
			StudioID:      studioID,
			UserID:        client.ID,
			StartTime:     startTime,
			EndTime:       endTime,
			TotalPrice:    room.PricePerHourMin * float64(duration),
			Status:        status,
			PaymentStatus: paymentStatus,
		}
		db.Create(&booking)
		bookings = append(bookings, booking)
	}
	log.Printf("  ✅ %d Bookings created", len(bookings))

	// ================= REVIEWS =================
	log.Println("⭐ Creating reviews...")
	comments := []string{
		"Отличная студия! Рекомендую всем фотографам.",
		"Хорошее освещение, удобная локация. Вернусь ещё.",
		"Профессиональное оборудование, вежливый персонал.",
		"Немного тесновато, но в целом неплохо для портретов.",
		"Супер! Лучшая студия в городе.",
		"Цена соответствует качеству. Доволен.",
		"Чисто, аккуратно, всё работает. 5 звёзд.",
		"Хорошее место для начинающих фотографов.",
		"Отличный сервис и профессиональный подход.",
		"Современное оборудование, приятная атмосфера.",
	}

	var reviews []domain.Review
	for i := 0; i < 30; i++ {
		studio := studios[rng.Intn(len(studios))]
		client := clients[rng.Intn(len(clients))]
		rating := 3 + rng.Intn(3) // 3-5

		review := domain.Review{
			StudioID:   studio.ID,
			UserID:     client.ID,
			Rating:     rating,
			Comment:    comments[rng.Intn(len(comments))],
			IsVerified: true,
			IsHidden:   false,
		}
		db.Create(&review)
		reviews = append(reviews, review)

		// Update studio rating and review count
		db.Exec("UPDATE studios SET total_reviews = total_reviews + 1 WHERE id = ?", studio.ID)
	}
	log.Printf("  ✅ %d Reviews created", len(reviews))

	// Update studio ratings based on reviews
	for _, studio := range studios {
		var avgRating float64
		db.Model(&domain.Review{}).Where("studio_id = ?", studio.ID).Select("AVG(rating)").Scan(&avgRating)
		if avgRating > 0 {
			db.Model(&domain.Studio{}).Where("id = ?", studio.ID).Update("rating", avgRating)
		}
	}
	log.Println("  ✅ Studio ratings updated")

	// ================= NOTIFICATIONS =================
	// Note: Notifications are skipped in seed data to avoid GORM serialization issues
	// They will be created automatically by the system when events occur
	log.Println("🔔 Skipping notifications (created by system events)")

	log.Println("\n🎉 SEED COMPLETED SUCCESSFULLY!")
	log.Println("\n📊 Summary:")
	log.Printf("  • Users: %d (1 admin + %d owners + %d clients)", 1+len(owners)+len(clients), len(owners), len(clients))
	log.Printf("  • Studios: %d", len(studios))
	log.Printf("  • Rooms: %d", len(rooms))
	log.Printf("  • Bookings: %d", len(bookings))
	log.Printf("  • Reviews: %d", len(reviews))
	log.Println("\n🔑 Test Accounts:")
	log.Println("  Admin:")
	log.Println("    Email: admin@studiobooking.kz")
	log.Println("    Password: admin123")
	log.Println("\n  Studio Owners:")
	log.Println("    Email: owner1@studio.kz")
	log.Println("    Email: owner2@studio.kz")
	log.Println("    Password: owner123")
	log.Println("\n  Clients:")
	log.Println("    Email: client1@test.com")
	log.Println("    Email: client2@test.com")
	log.Println("    Password: client123")
}
