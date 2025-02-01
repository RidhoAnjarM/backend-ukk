package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/database"
	"backend/models"
	"backend/utils"
)

func GetNotifications(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userData, ok := user.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	var notifications []models.Notification
	err := database.DB.
		Preload("Comment.User").  // Preload User untuk Comment
		Preload("Reply.User").    // Preload User untuk Reply
		Preload("Forum").         // Preload Forum
		Where("user_id = ?", userData.ID).
		Order("created_at DESC").
		Find(&notifications).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	var result []gin.H
	for _, notif := range notifications {
		notificationData := gin.H{
			"id":         notif.ID,
			"content":    notif.Content,
			"isRead":     notif.IsRead,
			"created_at": notif.CreatedAt,
		}
	
		// Handle Forum
		if notif.Forum != nil {
			notificationData["forum_id"] = notif.Forum.ID
			notificationData["forum_title"] = notif.Forum.Title
			notificationData["photo"] = notif.Forum.Photo
			notificationData["forum_relative_time"] = utils.TimeAgo(notif.Forum.CreatedAt)
		} else {
			notificationData["forum_id"] = nil
			notificationData["forum_title"] = ""
			notificationData["photo"] = ""
			notificationData["forum_relative_time"] = ""
		}
	
		// Handle Comment & Reply
		if notif.Reply != nil {
			// Jika ini adalah notifikasi balasan komentar
			notificationData["reply_id"] = notif.Reply.ID
			notificationData["reply"] = notif.Reply.Content
			notificationData["reply_user"] = notif.Reply.User.Username
			notificationData["reply_profile"] = notif.Reply.User.Profile
			notificationData["reply_relative_time"] = utils.TimeAgo(notif.Reply.CreatedAt)
	
			// Kosongkan data komentar karena ini balasan, bukan komentar utama
			notificationData["comment_id"] = nil
			notificationData["comment"] = ""
			notificationData["user"] = ""
			notificationData["profile"] = ""
			notificationData["relative_time"] = ""
		} else if notif.Comment != nil {
			// Jika ini adalah notifikasi komentar biasa
			notificationData["comment_id"] = notif.Comment.ID
			notificationData["comment"] = notif.Comment.Content
			notificationData["user"] = notif.Comment.User.Username
			notificationData["profile"] = notif.Comment.User.Profile
			notificationData["relative_time"] = utils.TimeAgo(notif.Comment.CreatedAt)
	
			// Kosongkan data balasan karena ini komentar utama, bukan balasan
			notificationData["reply_id"] = nil
			notificationData["reply"] = ""
			notificationData["reply_user"] = ""
			notificationData["reply_profile"] = ""
			notificationData["reply_relative_time"] = ""
		}
	
		result = append(result, notificationData)
	}
	

	c.JSON(http.StatusOK, gin.H{
		"notifications": result,
	})
}


func MarkNotificationAsRead(c *gin.Context) {
	notificationID := c.Param("id")

	var notification models.Notification
	if err := database.DB.First(&notification, notificationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	notification.IsRead = true
	if err := database.DB.Save(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

func MarkAllNotificationsAsRead(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userData, ok := user.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	if err := database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userData.ID, false).
		Update("is_read", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notifications as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

func DeleteNotification(c *gin.Context) {
	notificationID := c.Param("id")

	var notification models.Notification
	if err := database.DB.First(&notification, notificationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	if err := database.DB.Delete(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted successfully"})
}

func DeleteAllNotifications(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userData, ok := user.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	if err := database.DB.Where("user_id = ?", userData.ID).Delete(&models.Notification{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete all notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications deleted successfully"})
}

