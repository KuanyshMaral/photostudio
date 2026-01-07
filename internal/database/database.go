package database

import (
	"log"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

func Connect(dsn string) (*gorm.DB, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		log.Println("Connecting to PostgreSQL...")
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	}

	log.Println("Using SQLite for local development:", dsn)

	return gorm.Open(
		sqlite.New(sqlite.Config{
			DriverName: "sqlite",
			DSN:        dsn,
		}),
		&gorm.Config{},
	)
}
