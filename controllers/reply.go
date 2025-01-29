package controllers

import (
	"net/http"
	"strconv"

	"fmt"
	"github.com/gin-gonic/gin"
	"time"

	"backend/database"
	"backend/models"
	"backend/utils"
)

func ReplyComment(c *gin.Context) {
	var reply models.Reply

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

	reply.UserID = userData.ID

	parentIDStr := c.PostForm("parent_id")
	if parentIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parent ID is required"})
		return
	}

	parentID, err := strconv.Atoi(parentIDStr)
	if err != nil || parentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Parent ID"})
		return
	}

	var parentComment models.Comment
	if err := database.DB.First(&parentComment, parentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Parent comment not found"})
		return
	}

	reply.CommentID = parentComment.ID

	reply.Content = c.PostForm("content")
	if reply.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content cannot be empty"})
		return
	}

	if err := database.DB.Create(&reply).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add reply", "details": err.Error()})
		return
	}

	notification := models.Notification{
		UserID:    parentComment.UserID,
		Content:   fmt.Sprintf("%s membalas komentar anda: %s", userData.Username, reply.Content),
		ForumID:   parentComment.ForumID,
		CommentID: &parentComment.ID,
		ReplyID:   &reply.ID,
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&notification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification", "details": err.Error()})
		return
	}

	if err := database.DB.Preload("User").First(&reply, reply.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reply with user data"})
		return
	}

	response := gin.H{
		"id":            reply.ID,
		"content":       reply.Content,
		"user_id":       reply.UserID,
		"username":      reply.User.Username,
		"profile":       reply.User.Profile,
		"parent_id":     reply.CommentID,
		"relative_time": utils.TimeAgo(reply.CreatedAt),
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Reply added successfully",
		"reply":   response,
	})
}
