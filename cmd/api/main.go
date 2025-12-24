package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"photostudio/internal/modules/catalog"
	"photostudio/internal/repository"
)

func main() {
	dsn := "postgres://user:password@localhost:5432/photostudio?sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	repo := repository.NewStudioRepository(db)
	handler := catalog.NewHandler(repo)

	r := gin.Default()
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)

	r.Run(":8080")
}
