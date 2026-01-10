package main

import (
	"log"
	"photostudio/internal/database"
	"photostudio/internal/domain"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	db, err := database.Connect("studio.db")
	if err != nil {
		log.Fatal(err)
	}

	// Auto migrate
	db.AutoMigrate(
		&domain.User{},
		&domain.StudioOwner{},
		&domain.Studio{},
		&domain.Room{},
		&domain.Equipment{},
		&domain.Booking{},
		&domain.Review{},
	)

	// Clean previous data (safe for dev)
	log.Println("Cleaning old data...")
	db.Exec("DELETE FROM reviews")
	db.Exec("DELETE FROM bookings")
	db.Exec("DELETE FROM equipment")
	db.Exec("DELETE FROM rooms")
	db.Exec("DELETE FROM studios")
	db.Exec("DELETE FROM studio_owners")
	db.Exec("DELETE FROM users")

	// ================= ADMIN =================
	adminPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	admin := domain.User{
		Email:        "admin@studiobooking.kz",
		PasswordHash: string(adminPassword),
		Role:         domain.RoleAdmin,
		Name:         "Admin",
	}
	db.Create(&admin)
	log.Println("✅ Admin created")

	// ================= CLIENT =================
	clientPassword, _ := bcrypt.GenerateFromPassword([]byte("client123"), bcrypt.DefaultCost)
	client := domain.User{
		Email:        "client@test.com",
		PasswordHash: string(clientPassword),
		Role:         domain.RoleClient,
		Name:         "Test Client",
		Phone:        "+7 777 123 4567",
	}
	db.Create(&client)
	log.Println("✅ Client created")

	// BOOKING
	const (
		clientID = 2 // usually the first client after admin
		roomID   = 1 // usually the first room
	)

	// Example time: tomorrow 14:00–16:00
	start := time.Now().Truncate(24 * time.Hour).Add(24*time.Hour + 14*time.Hour)
	end := start.Add(2 * time.Hour)

	booking := domain.Booking{
		RoomID:        roomID,
		StudioID:      1, // usually first studio
		UserID:        clientID,
		StartTime:     start,
		EndTime:       end,
		TotalPrice:    16000, // example price
		Status:        domain.BookingPending,
		PaymentStatus: domain.PaymentUnpaid,
		Notes:         "Test booking for demo",
	}

	if err := db.Create(&booking).Error; err != nil {
		log.Fatalf("Failed to create booking: %v", err)
	}

	log.Println("✅ Booking created successfully!")

	// ================= OWNER =================
	ownerPassword, _ := bcrypt.GenerateFromPassword([]byte("owner123"), bcrypt.DefaultCost)
	owner := domain.User{
		Email:        "owner@studio.kz",
		PasswordHash: string(ownerPassword),
		Role:         domain.RoleStudioOwner,
		Name:         "Studio Owner",
		StudioStatus: domain.StatusVerified,
	}
	db.Create(&owner)
	log.Println("✅ Owner created")

	// Owner details
	studioOwner := domain.StudioOwner{
		UserID:      owner.ID,
		CompanyName: "Light Studio LLC",
		BIN:         "123456789012",
	}
	db.Create(&studioOwner)

	// ================= STUDIOS =================
	studios := []domain.Studio{
		{
			OwnerID:      owner.ID,
			Name:         "Light Studio Pro",
			Description:  "Профессиональная фотостудия",
			Address:      "ул. Абая 150",
			District:     "Алмалинский",
			City:         "Алматы",
			Rating:       4.8,
			Phone:        "+7 727 123 4567",
			WorkingHours: nil,
		},
		{
			OwnerID:      owner.ID,
			Name:         "Creative Space",
			Description:  "Креативная фотостудия",
			Address:      "пр. Достык 89",
			District:     "Медеуский",
			City:         "Алматы",
			Rating:       4.5,
			WorkingHours: nil,
		},
	}

	for i := range studios {
		db.Create(&studios[i])
	}
	log.Println("✅ Studios created")

	// ================= ROOMS =================
	rooms := []domain.Room{
		{
			StudioID:        studios[0].ID,
			Name:            "Белый зал",
			Description:     "Классический белый фон",
			AreaSqm:         50,
			Capacity:        10,
			RoomType:        domain.RoomFashion,
			PricePerHourMin: 8000,
			IsActive:        true,
		},
		{
			StudioID:        studios[0].ID,
			Name:            "Чёрный зал",
			Description:     "Драматичное освещение",
			AreaSqm:         40,
			Capacity:        8,
			RoomType:        domain.RoomPortrait,
			PricePerHourMin: 10000,
			IsActive:        true,
		},
		{
			StudioID:        studios[1].ID,
			Name:            "Лофт",
			Description:     "Индустриальный стиль",
			AreaSqm:         80,
			Capacity:        15,
			RoomType:        domain.RoomCreative,
			PricePerHourMin: 15000,
			IsActive:        true,
		},
	}

	for i := range rooms {
		db.Create(&rooms[i])
	}
	log.Println("✅ Rooms created")

	log.Println("🎉 SEED COMPLETED!")
}
