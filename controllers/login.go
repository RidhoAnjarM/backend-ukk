package controllers

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"

	"backend/database"
	"backend/models"
)

var jwtSecret = []byte(os.Getenv("rahasia"))

// Login user
func Login(c *gin.Context) {
	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var user models.User
	if err := database.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	if user.Status == "suspended" {
		if user.SuspendUntil != nil && time.Now().Before(*user.SuspendUntil) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "Akun anda terkena suspend",
				"suspend_until": user.SuspendUntil.Format("2006-01-02 15:04:05"),
			})
			return
		} else {
			// Jika waktu suspend sudah habis, ubah status kembali ke "active"
			user.Status = "active"
			user.SuspendUntil = nil
			database.DB.Save(&user)
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   float64(user.ID),
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Login berhasil",
		"id":       user.ID,
		"username": user.Username,
		"profile":  user.Profile,
		"password": user.Password,
		"role":     user.Role,
		"status":   user.Status,
		"token":    tokenString,
	})
}

// Register user
func Register(c *gin.Context) {
	var input models.User

	input.Name = c.PostForm("name")
	input.Username = c.PostForm("username")
	input.Password = c.PostForm("password")
	input.Role = c.PostForm("role")
	input.Status = c.PostForm("status")

	if input.Role == "" {
		input.Role = "user"
	}
	if input.Status == "" {
		input.Status = "active"
	}

	if input.Username == "" || input.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username dan password harus diisi"})
		return
	}

	if input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama harus diisi"})
	}

	var existingUser models.User
	if err := database.DB.Where("username = ?", input.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username sudah digunakan"})
		return
	}

	file, err := c.FormFile("profile")
	if err == nil {
		uploadPath := fmt.Sprintf("./uploads/%s", file.Filename)
		if _, err := os.Stat("./uploads"); os.IsNotExist(err) {
			os.MkdirAll("./uploads", os.ModePerm)
		}
		if err := c.SaveUploadedFile(file, uploadPath); err == nil {
			input.Profile = fmt.Sprintf("/uploads/%s", file.Filename)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload photo"})
			return
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	input.Password = string(hashedPassword)

	if err := database.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "User berhasil dibuat",
		"id":       input.ID,
		"name":     input.Name,
		"username": input.Username,
		"profile":  input.Profile,
		"role":     input.Role,
		"status":   input.Status,
	})
}
