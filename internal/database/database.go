package database

import (
	"log"
	"strings"

	"gorm.io/driver/postgres"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Connect(dsn string) (*gorm.DB, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		log.Println("Connecting to PostgreSQL...")
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	}

	log.Println("Using SQLite for local development:", dsn)

	return gorm.Open(
		gormsqlite.New(gormsqlite.Config{
			DriverName: "sqlite",
			DSN:        dsn,
		}),
		&gorm.Config{},
	)
}
