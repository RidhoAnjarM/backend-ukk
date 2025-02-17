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

	parentReplyIDStr := c.PostForm("parent_reply_id")
	if parentReplyIDStr != "" {
		parentReplyID, err := strconv.Atoi(parentReplyIDStr)
		if err == nil && parentReplyID > 0 {
			var parentReply models.Reply
			if err := database.DB.First(&parentReply, parentReplyID).Error; err == nil {
				tempID := uint(parentReplyID)
				reply.ParentReplyID = &tempID
			}
		}
	}

	reply.Content = c.PostForm("content")
	if reply.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content cannot be empty"})
		return
	}

	if err := database.DB.Create(&reply).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add reply", "details": err.Error()})
		return
	}

	if reply.ParentReplyID != nil {
		var parentReply models.Reply
		if err := database.DB.First(&parentReply, *reply.ParentReplyID).Error; err == nil {
			notification := models.Notification{
				UserID:    parentReply.UserID,
				Content:   fmt.Sprintf("%s membalas komentar Anda: %s", userData.Username, reply.Content),
				ForumID:   parentComment.ForumID,
				CommentID: &parentComment.ID,
				ReplyID:   &reply.ID,
				CreatedAt: time.Now(),
			}
			database.DB.Create(&notification)
		}
	} else {
		notification := models.Notification{
			UserID:    parentComment.UserID,
			Content:   fmt.Sprintf("%s membalas komentar Anda: %s", userData.Username, reply.Content),
			ForumID:   parentComment.ForumID,
			CommentID: &parentComment.ID,
			ReplyID:   &reply.ID,
			CreatedAt: time.Now(),
		}
		database.DB.Create(&notification)
	}

	if err := database.DB.Preload("User").First(&reply, reply.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reply with user data"})
		return
	}

	response := gin.H{
		"id":              reply.ID,
		"content":         reply.Content,
		"user_id":         reply.UserID,
		"username":        reply.User.Username,
		"name":            reply.User.Name,
		"profile":         reply.User.Profile,
		"parent_id":       reply.CommentID,
		"parent_reply_id": reply.ParentReplyID,
		"relative_time":   utils.TimeAgo(reply.CreatedAt),
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Reply added successfully",
		"reply":   response,
	})
}
