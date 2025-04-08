package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"fmt"
	"time"

	"github.com/gin-gonic/gin"

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

	// Handle upload gambar (opsional)
	file, err := c.FormFile("image")
	if err == nil {
		uploadDir := "./uploads/replies"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			os.MkdirAll(uploadDir, os.ModePerm)
		}

		filename := fmt.Sprintf("reply-%d-%d-%s", userData.ID, time.Now().Unix(), file.Filename)
		uploadPath := filepath.Join(uploadDir, filename)

		if err := c.SaveUploadedFile(file, uploadPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
			return
		}
		reply.ImageURL = fmt.Sprintf("/uploads/replies/%s", filename)
	}

	if err := database.DB.Create(&reply).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add reply", "details": err.Error()})
		return
	}

	if reply.ParentReplyID != nil {
		var parentReply models.Reply
		if err := database.DB.First(&parentReply, *reply.ParentReplyID).Error; err == nil {
			if parentReply.UserID != userData.ID {
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
		}
	} else {
		if parentComment.UserID != userData.ID {
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
		"image_url":       reply.ImageURL, // Tambah image_url di response
		"parent_id":       reply.CommentID,
		"parent_reply_id": reply.ParentReplyID,
		"relative_time":   utils.TimeAgo(reply.CreatedAt),
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Reply added successfully",
		"reply":   response,
	})
}
