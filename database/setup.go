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
    // Load .env hanya untuk development lokal
    if os.Getenv("FLY_APP_NAME") == "" && os.Getenv("RAILWAY_ENVIRONMENT") == "" {
        err := godotenv.Load()
        if err != nil {
            log.Println("Warning: Error loading .env file, relying on environment variables")
        }
    }

    // Ambil DATABASE_URL dari environment variable
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        // Fallback untuk lokal, jika DATABASE_URL tidak ada
        dbHost := os.Getenv("DB_HOST")
        dbUser := os.Getenv("DB_USER")
        dbPassword := os.Getenv("DB_PASSWORD")
        dbName := os.Getenv("DB_NAME")
        dbPort := os.Getenv("DB_PORT")
        dbSslMode := os.Getenv("DB_SSLMODE")
        dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
            dbHost, dbUser, dbPassword, dbName, dbPort, dbSslMode)
    }

    // Validasi DSN
    if dsn == "" {
        log.Fatal("Error: DATABASE_URL or DB connection details are not set")
    }

    // Connect ke database
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        DisableForeignKeyConstraintWhenMigrating: true,
        Logger:                                   logger.Default.LogMode(logger.Warn),
    })
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    // Migrasi tabel
    safeAutoMigrate(db, &models.User{}, &models.Forum{}, &models.Comment{}, &models.Like{}, &models.Notification{}, &models.Reply{}, &models.Report{}, &models.ForumReport{}, &models.Tag{}, &models.ResetLog{})

    DB = db
    fmt.Println("Database connected successfully")
}

func safeAutoMigrate(db *gorm.DB, models ...interface{}) {
    for _, model := range models {
        err := db.AutoMigrate(model)
        if err != nil {
            log.Fatalf("Failed to migrate table for model %T: %v", model, err)
        }
    }
}