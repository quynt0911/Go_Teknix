package database

import (
	"fmt"
	"log"
	"task-management-system/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	var err error

	// Kết nối PostgreSQL
	dsn := "host=localhost user=postgres password=091123 dbname=taskdb port=5432 sslmode=disable"
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}

	// Auto migrate các bảng
	if err := DB.AutoMigrate(&models.User{}, &models.Task{}); err != nil {
		log.Fatal("❌ AutoMigrate error:", err)
	}

	fmt.Println("✅ Connected to Database")
}
