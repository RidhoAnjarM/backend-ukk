package controllers

import (
	"net/http"

	"backend/database"
	"backend/models"
	"backend/utils"
	"github.com/gin-gonic/gin"
)

func GetProfile(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	userData, ok := user.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data in context"})
		return
	}

	var userWithForums models.User
	if err := database.DB.Preload("Forums.User").Preload("Forums.Category").Preload("Forums.Comments.User").Preload("Forums.Comments.Replies.User").First(&userWithForums, userData.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data", "details": err.Error()})
		return
	}

	var forumsResponse []gin.H
	for _, forum := range userWithForums.Forums {
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
			"comments":      comments,
		}
		forumsResponse = append(forumsResponse, forumResponse)
	}

	response := gin.H{
		"id":            userWithForums.ID,
		"name":          userWithForums.Name,
		"username":      userWithForums.Username,
		"profile":       userWithForums.Profile,
		"role":          userWithForums.Role,
		"status":        userWithForums.Status,
		"suspend_until": userWithForums.SuspendUntil,
		"created_at":    userWithForums.CreatedAt.Format("2006-01-02 15:04:05"),
		"forums":        forumsResponse,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile retrieved successfully",
		"profile": response,
	})
}
