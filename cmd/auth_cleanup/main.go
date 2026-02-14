package main

import (
	"log"
	"os"
	"photostudio/internal/database"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := database.Connect(databaseURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}

	res1 := db.Exec(`DELETE FROM refresh_tokens WHERE expires_at < NOW() OR revoked_at IS NOT NULL AND created_at < NOW() - INTERVAL '30 days'`)
	if res1.Error != nil {
		log.Fatalf("cleanup refresh_tokens failed: %v", res1.Error)
	}

	res2 := db.Exec(`DELETE FROM email_verification_codes WHERE expires_at < NOW() OR used_at IS NOT NULL`)
	if res2.Error != nil {
		log.Fatalf("cleanup email_verification_codes failed: %v", res2.Error)
	}

	log.Printf("auth cleanup completed: refresh_tokens=%d email_verification_codes=%d", res1.RowsAffected, res2.RowsAffected)
}