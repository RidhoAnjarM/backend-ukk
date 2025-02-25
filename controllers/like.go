package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"backend/models"
	"backend/database"
)

type LikeInput struct {
	ForumID uint `json:"forum_id" binding:"required"`
}

func LikeForum(c *gin.Context) {
	userID := c.MustGet("userID").(uint) 

	var input LikeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existingLike models.Like
	if err := database.DB.Where("user_id = ? AND forum_id = ?", userID, input.ForumID).First(&existingLike).Error; err == nil {
		database.DB.Delete(&existingLike)
		database.DB.Model(&models.Forum{}).Where("id = ?", input.ForumID).Update("likes_count", gorm.Expr("likes_count - 1"))
		c.JSON(http.StatusOK, gin.H{"message": "Unlike berhasil", "liked": false})
		return
	}

	like := models.Like{
		UserID:  userID,
		ForumID: &input.ForumID,
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&like).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyukai forum"})
		return
	}

	database.DB.Model(&models.Forum{}).Where("id = ?", input.ForumID).Update("likes_count", gorm.Expr("likes_count + 1"))

	c.JSON(http.StatusOK, gin.H{"message": "Forum berhasil disukai", "liked": true})
}

func GetForumLikesCount(c *gin.Context) {
	forumID := c.Param("id")

	var count int64
	database.DB.Model(&models.Like{}).Where("forum_id = ?", forumID).Count(&count)
	database.DB.Model(&models.Forum{}).Where("id = ?", forumID).Update("likes_count", count)

	c.JSON(http.StatusOK, gin.H{"forum_id": forumID, "likes_count": count})
}
