package main

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"backend/controllers"
	"backend/database"
	"backend/routes"
)

func main() {
	r := gin.Default()

	r.Static("/uploads", "./uploads")

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	database.ConnectDatabase()

	// Cek sekali saat start dan jalankan pengecekan berkala
	controllers.CheckAndResetTags()
	controllers.CheckAndResetTagsPeriodically()

	routes.SetupRouter(r)

	r.Run(":5000")
}
