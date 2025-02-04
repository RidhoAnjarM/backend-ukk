package controllers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "backend/models" 
    "backend/database" 
)

func GetProfile(c *gin.Context) {
    // Ambil data pengguna dari context yang diset oleh middleware
    user, exists := c.Get("user")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
        return
    }

    // Konversi ke model User
    userData, ok := user.(models.User)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data in context"})
        return
    }

    // Preload data terkait (misalnya, forum yang dibuat oleh pengguna)
    var userWithForums models.User
    if err := database.DB.Preload("Forums").First(&userWithForums, userData.ID).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data", "details": err.Error()})
        return
    }

    // Siapkan respons
    response := gin.H{
        "id":            userWithForums.ID,
        "name":          userWithForums.Name,
        "username":      userWithForums.Username,
        "profile":       userWithForums.Profile,
        "role":          userWithForums.Role,
        "status":        userWithForums.Status,
        "suspend_until": userWithForums.SuspendUntil,
        "created_at":    userWithForums.CreatedAt,
        "forums":        userWithForums.Forums, 
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Profile retrieved successfully",
        "profile": response,
    })
}