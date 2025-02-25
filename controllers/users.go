package controllers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"backend/database"
	"backend/models"
	"backend/utils"
)

func GetUserByID(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := database.DB.Preload("Forums.User").Preload("Forums.Category").Preload("Forums.Comments.User").Preload("Forums.Comments.Replies.User").First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var forumsResponse []gin.H
	for _, forum := range user.Forums {
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
		}

		forumResponse := gin.H{
			"id":            forum.ID,
			"title":         forum.Title,
			"photo":         forum.Photo,
			"user_id":       forum.UserID,
			"username":      forum.User.Username,
			"name":          forum.User.Name,
			"profile":       forum.User.Profile,
			"category_id":   forum.CategoryID,
			"category_name": forum.Category.Name,
			"relative_time": utils.TimeAgo(forum.CreatedAt),
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

	// Update name
	name := c.PostForm("name")
	if name != "" {
		user.Name = name
	}

	// Update username (check if already exists)
	username := c.PostForm("username")
	if username != "" && username != user.Username {
		var existingUser models.User
		if err := database.DB.Where("username = ? AND id != ?", username, id).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username already registered"})
			return
		}
		user.Username = username
	}

	// Update password
	password := c.PostForm("password")
	if password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		user.Password = string(hashedPassword)
	}

	// Update role
	role := c.PostForm("role")
	if role != "" {
		user.Role = role
	}

	// Update status
	status := c.PostForm("status")
	if status != "" {
		user.Status = status
	}

	// Handle profile picture upload
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

	// Save changes
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
