package main

import (
    "time"
    "os" // Tambah os untuk baca PORT
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
    "backend/controllers"
    "backend/database"
    "backend/routes"
)

func main() {
    r := gin.Default()

    r.Static("/uploads", "./uploads")

    // Konfigurasi CORS
    r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"*"}, // Sementara pakai "*" untuk test, ganti dengan URL Vercel nanti
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Authorization", "Content-Type"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    }))

    database.ConnectDatabase()

    controllers.CheckAndResetTags()
    controllers.CheckAndResetTagsPeriodically()

    routes.SetupRouter(r)

    // Baca port dari environment variable
    port := os.Getenv("PORT")
    if port == "" {
        port = "5000" // Fallback untuk lokal
    }
    r.Run(":" + port)
}