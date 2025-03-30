package database

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"backend/models"
)

var DB *gorm.DB

func ConnectDatabase() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	dbSslMode := os.Getenv("DB_SSLMODE")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSslMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Warn), 
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	safeAutoMigrate(db, &models.User{}, &models.Forum{}, &models.Comment{}, &models.Like{}, &models.Notification{}, &models.Reply{}, &models.Report{}, &models.ForumReport{}, &models.Tag{})

	DB = db
	fmt.Println("Database nyambung")
}

func safeAutoMigrate(db *gorm.DB, models ...interface{}) {
    for _, model := range models {
        err := db.AutoMigrate(model)
        if err != nil {
            log.Fatalf("Failed to migrate table for model %T: %v", model, err)
        }
    }
}

