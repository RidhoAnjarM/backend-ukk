package controllers

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"backend/database"
	"backend/models"
	"backend/utils"
)

func GetUserByID(c *gin.Context) {
	id := c.Param("id")

	_, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	if err := database.DB.Preload("Forums.User").Preload("Forums.Category").Preload("Forums.Comments.User").Preload("Forums.Comments.Replies.User").Preload("Forums.Tags").First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var forumsResponse []gin.H
	for _, forum := range user.Forums {
		var like models.Like
		liked := database.DB.Where("user_id = ? AND forum_id = ?", userID, forum.ID).First(&like).Error == nil
		var comments []gin.H
		for _, comment := range forum.Comments {
			var replies []gin.H
			for _, reply := range comment.Replies {
				replies = append(replies, gin.H{
					"id":            reply.ID,
					"content":       reply.Content,
					"user_id":       reply.UserID,
					"username":      reply.User.Username,
					"name":          reply.User.Name,
					"profile":       reply.User.Profile,
					"created_at":    reply.CreatedAt.Format("2006-01-02 15:04:05"),
					"relative_time": utils.TimeAgo(reply.CreatedAt),
				})
			}

			comments = append(comments, gin.H{
				"id":            comment.ID,
				"content":       comment.Content,
				"user_id":       comment.UserID,
				"username":      comment.User.Username,
				"name":          comment.User.Name,
				"profile":       comment.User.Profile,
				"created_at":    comment.CreatedAt.Format("2006-01-02 15:04:05"),
				"relative_time": utils.TimeAgo(comment.CreatedAt),
				"replies":       replies,
			})
		}

		var tags []gin.H
		for _, tag := range forum.Tags {
			tags = append(tags, gin.H{
				"id":   tag.ID,
				"name": tag.Name,
			})
		}

		forumResponse := gin.H{
			"id":            forum.ID,
			"title":         forum.Title,
			"description":   forum.Description,
			"photo":         forum.Photo,
			"user_id":       forum.UserID,
			"username":      forum.User.Username,
			"name":          forum.User.Name,
			"profile":       forum.User.Profile,
			"category_id":   forum.CategoryID,
			"category_name": forum.Category.Name,
			"relative_time": utils.TimeAgo(forum.CreatedAt),
			"like":          forum.LikesCount,
			"liked":         liked,
			"comments":      comments,
			"tags":          tags,
		}
		forumsResponse = append(forumsResponse, forumResponse)
	}

	response := gin.H{
		"id":               user.ID,
		"name":             user.Name,
		"username":         user.Username,
		"profile":          user.Profile,
		"role":             user.Role,
		"status":           user.Status,
		"suspend_until":    user.SuspendUntil,
		"suspend_duration": user.SuspendDuration,
		"created_at":       user.CreatedAt.Format("2006-01-02 15:04:05"),
		"forums":           forumsResponse,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile retrieved successfully",
		"profile": response,
	})
}

func GetAllUsers(c *gin.Context) {
	var users []models.User
	if err := database.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}

	var response []gin.H
	for _, user := range users {
		// Jika pengguna sedang di-suspend dan SuspendUntil tidak null
		if user.Status == "suspended" && user.SuspendUntil != nil {
			// Hitung selisih waktu antara SuspendUntil dan waktu sekarang
			timeRemaining := time.Until(*user.SuspendUntil)

			// Jika waktu suspend sudah habis (timeRemaining <= 0)
			if timeRemaining <= 0 {
				// Update status pengguna menjadi active
				user.Status = "active"
				user.SuspendUntil = nil
				user.SuspendDuration = 0

				// Simpan perubahan ke database
				database.DB.Save(&user)
			} else {
				// Hitung durasi suspend yang tersisa dalam hari
				user.SuspendDuration = int(timeRemaining.Hours() / 24)
				if user.SuspendDuration < 1 {
					user.SuspendDuration = 1 // Minimal 1 hari jika kurang dari 24 jam
				}
			}
		} else if user.Status == "active" {
			// Jika status pengguna adalah active, pastikan suspend_duration adalah 0
			if user.SuspendDuration != 0 {
				user.SuspendDuration = 0
				user.SuspendUntil = nil

				// Simpan perubahan ke database
				database.DB.Save(&user)
			}
		}

		response = append(response, gin.H{
			"id":               user.ID,
			"name":             user.Name,
			"username":         user.Username,
			"profile":          user.Profile,
			"status":           user.Status,
			"role":             user.Role,
			"suspend_until":    user.SuspendUntil,
			"suspend_duration": user.SuspendDuration,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"users": response,
	})
}

// Update user by ID
func UpdateUser(c *gin.Context) {
	var user models.User
	id := c.Param("id")

	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	name := c.PostForm("name")
	if name != "" {
		user.Name = name
	}

	username := c.PostForm("username")
	if username != "" && username != user.Username {
		var existingUser models.User
		if err := database.DB.Where("username = ? AND id != ?", username, id).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username already registered"})
			return
		}
		user.Username = username
	}

	password := c.PostForm("password")
	if password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		user.Password = string(hashedPassword)
	}

	role := c.PostForm("role")
	if role != "" {
		user.Role = role
	}

	status := c.PostForm("status")
	if status != "" {
		user.Status = status
	}

	file, err := c.FormFile("profile")
	if err == nil {
		uploadPath := fmt.Sprintf("./uploads/%s", file.Filename)
		if _, err := os.Stat("./uploads"); os.IsNotExist(err) {
			os.MkdirAll("./uploads", os.ModePerm)
		}
		if err := c.SaveUploadedFile(file, uploadPath); err == nil {
			user.Profile = fmt.Sprintf("/uploads/%s", file.Filename)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload photo"})
			return
		}
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "User updated successfully",
		"id":       user.ID,
		"name":     user.Name,
		"username": user.Username,
		"profile":  user.Profile,
		"role":     user.Role,
		"status":   user.Status,
	})
}

// Delete user by ID
func DeleteUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := database.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}

func GetUsername(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var user models.User
	if err := database.DB.Select("name, username, profile").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":     user.Name,
		"username": user.Username,
		"profile":  user.Profile,
	})
}

func GetUserStats(c *gin.Context) {
    var totalUsers int64
    var weeklyUsers int64

    if err := database.DB.Model(&models.User{}).Count(&totalUsers).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch total users"})
        return
    }

    startOfWeek := time.Now().AddDate(0, 0, -int(time.Now().Weekday())+1).Truncate(24 * time.Hour) // Senin minggu ini
    if err := database.DB.Model(&models.User{}).
        Where("created_at >= ?", startOfWeek).
        Count(&weeklyUsers).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch weekly users"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "total_users":   totalUsers,
        "weekly_users":  weeklyUsers,
    })
}
