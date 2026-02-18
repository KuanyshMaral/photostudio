package database

import (
	"log"
	"strings"

	gormsqlite "github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(dsn string) (*gorm.DB, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		log.Println("Connecting to PostgreSQL...")
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	}

	log.Println("Using SQLite for local development:", dsn)

	return gorm.Open(
		gormsqlite.Open(dsn),
		&gorm.Config{},
	)
}
