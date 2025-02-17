package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"backend/database"
	"backend/models"
	"backend/utils"
)

func AddComment(c *gin.Context) {
	var comment models.Comment

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

	comment.UserID = userData.ID

	forumIDStr := c.PostForm("forum_id")
	forumID, err := strconv.Atoi(forumIDStr)
	if err != nil || forumID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid forum ID"})
		return
	}
	comment.ForumID = uint(forumID)

	comment.Content = c.PostForm("content")
	if comment.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content cannot be empty"})
		return
	}

	if err := database.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add comment", "details": err.Error()})
		return
	}

	database.DB.Preload("User").First(&comment, comment.ID)

	response := gin.H{
		"id":            comment.ID,
		"content":       comment.Content,
		"forum_id":      comment.ForumID,
		"user_id":       comment.UserID,
		"username":      comment.User.Username,
		"name":          comment.User.Name,
		"profile":       comment.User.Profile,
		"relative_time": utils.TimeAgo(comment.CreatedAt),
	}

	var forum models.Forum
	if err := database.DB.First(&forum, comment.ForumID).Error; err == nil {
		var forumUser models.User
		if err := database.DB.First(&forumUser, forum.UserID).Error; err == nil {
			notification := models.Notification{
				UserID:    forumUser.ID,
				Content:   fmt.Sprintf("%s mengomentari postingan anda: %s ", userData.Username, comment.Content),
				ForumID:   forum.ID,
				CommentID: &comment.ID,
				CreatedAt: time.Now(),
			}
			database.DB.Create(&notification)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Comment added successfully",
		"comment": response,
	})
}

func GetAllComments(c *gin.Context) {
	var comments []models.Comment

	if err := database.DB.Preload("User").Preload("Replies.User").Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve comments", "details": err.Error()})
		return
	}

	var response []gin.H
	for _, comment := range comments {
		replies := []gin.H{}
		for _, reply := range comment.Replies {
			replies = append(replies, gin.H{
				"id":            reply.ID,
				"content":       reply.Content,
				"user_id":       reply.UserID,
				"username":      reply.User.Username,
				"created_at":    reply.CreatedAt.Format("2006-01-02 15:04:05"),
				"relative_time": utils.TimeAgo(reply.CreatedAt),
			})
		}

		response = append(response, gin.H{
			"id":            comment.ID,
			"content":       comment.Content,
			"forum_id":      comment.ForumID,
			"user_id":       comment.UserID,
			"username":      comment.User.Username,
			"created_at":    comment.CreatedAt.Format("2006-01-02 15:04:05"),
			"relative_time": utils.TimeAgo(comment.CreatedAt),
			"replies":       replies,
		})
	}

	c.JSON(http.StatusOK, gin.H{"comments": response})
}

func GetCommentByID(c *gin.Context) {
	commentID := c.Param("id")

	var comment models.Comment
	if err := database.DB.Preload("User").Preload("Replies.User").First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	replies := []gin.H{}
	for _, reply := range comment.Replies {
		replies = append(replies, gin.H{
			"id":            reply.ID,
			"content":       reply.Content,
			"user_id":       reply.UserID,
			"username":      reply.User.Username,
			"created_at":    reply.CreatedAt.Format("2006-01-02 15:04:05"),
			"relative_time": utils.TimeAgo(reply.CreatedAt),
		})
	}

	response := gin.H{
		"id":            comment.ID,
		"content":       comment.Content,
		"forum_id":      comment.ForumID,
		"user_id":       comment.UserID,
		"username":      comment.User.Username,
		"created_at":    comment.CreatedAt.Format("2006-01-02 15:04:05"),
		"relative_time": utils.TimeAgo(comment.CreatedAt),
		"replies":       replies,
	}

	c.JSON(http.StatusOK, gin.H{"comment": response})
}

func DeleteComment(c *gin.Context) {
	commentID := c.Param("id")

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

	var comment models.Comment
	if err := database.DB.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}

	if comment.UserID != userData.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own comments"})
		return
	}

	if err := database.DB.Delete(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}

func DeleteReply(c *gin.Context) {
	replyID := c.Param("id")

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

	var reply models.Reply
	if err := database.DB.First(&reply, replyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reply not found"})
		return
	}

	if reply.UserID != userData.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own replies"})
		return
	}

	if err := database.DB.Delete(&reply).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete reply", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reply deleted successfully"})
}
