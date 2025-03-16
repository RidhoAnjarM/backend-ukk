package controllers

import (
	"fmt"
	"gorm.io/gorm"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"backend/database"
	"backend/models"
	"backend/utils"
)

func CreateForum(c *gin.Context) {
	var forum models.Forum
	var tags []models.Tag

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userData, ok := user.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User data is invalid"})
		return
	}

	forum.UserID = uint(userData.ID)
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	file, err := c.FormFile("photo")
	if err == nil {
		uploadPath := fmt.Sprintf("./uploads/%s", file.Filename)
		if _, err := os.Stat("./uploads"); os.IsNotExist(err) {
			os.MkdirAll("./uploads", os.ModePerm)
		}
		if err := c.SaveUploadedFile(file, uploadPath); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save uploaded file"})
			return
		}
		forum.Photo = fmt.Sprintf("/uploads/%s", file.Filename)
	}

	forum.Title = c.PostForm("title")
	if forum.Title == "" {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title is required"})
		return
	}

	forum.Description = c.PostForm("description")
	if forum.Description == "" {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Description is required"})
		return
	}

	categoryIDStr := c.PostForm("category_id")
	if categoryIDStr != "" {
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil || categoryID <= 0 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Category ID"})
			return
		}

		var category models.Category
		if err := tx.First(&category, categoryID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}

		forum.CategoryID = &category.ID
	}

	tagNames := c.PostFormArray("tags")
	for _, tagName := range tagNames {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}

		var tag models.Tag
		if err := tx.Where("name = ?", tagName).First(&tag).Error; err != nil {
			tag = models.Tag{Name: tagName, UsageCount: 0}
			if err := tx.Create(&tag).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag"})
				return
			}
		}
		tags = append(tags, tag)
	}

	forum.Tags = tags
	if err := tx.Create(&forum).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create forum", "details": err.Error()})
		return
	}

	// Update usage count AFTER successful forum creation
	if forum.CategoryID != nil {
		if err := tx.Model(&models.Category{}).Where("id = ?", forum.CategoryID).Update("usage_count", gorm.Expr("usage_count + 1")).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category usage count"})
			return
		}
	}

	for _, tag := range tags {
		if err := tx.Model(&models.Tag{}).Where("id = ?", tag.ID).Update("usage_count", gorm.Expr("usage_count + 1")).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tag usage count"})
			return
		}
	}

	tx.Commit()

	database.DB.Preload("User").Preload("Category").Preload("Tags").First(&forum, forum.ID)

	response := models.ForumCreateResponse{
		ID:           uint(forum.ID),
		Title:        forum.Title,
		Description:  forum.Description,
		Photo:        forum.Photo,
		Username:     forum.User.Username,
		CategoryName: forum.Category.Name,
		Tags:         forum.Tags,
		CreatedAt:    forum.CreatedAt.Format("2006-01-02 15:04:05"),
		RelativeTime: utils.TimeAgo(forum.CreatedAt),
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Forum created successfully",
		"forum":   response,
	})
}

func GetAllForums(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var forums []models.Forum
	if err := database.DB.Preload("User").Preload("Category").Preload("Comments.User").Preload("Comments.Replies.User").Preload("Tags").Order("created_at desc").Find(&forums).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch forums"})
		return
	}

	if len(forums) > 1 {
		latestForum := forums[0]
		remainingForums := forums[1:]
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(remainingForums), func(i, j int) {
			remainingForums[i], remainingForums[j] = remainingForums[j], remainingForums[i]
		})
		forums = append([]models.Forum{latestForum}, remainingForums...)
	}

	var response []gin.H
	for _, forum := range forums {
		// Cek apakah user sudah like forum ini
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

		response = append(response, gin.H{
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
			"createAt":      forum.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}

func GetForumByID(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	var forum models.Forum

	// Ambil forum beserta relasi yang diperlukan
	if err := database.DB.Preload("User").
		Preload("Category").
		Preload("Comments.User").
		Preload("Comments.Replies.User").
		Preload("Tags").
		First(&forum, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Forum not found"})
		return
	}

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
				"profile":       reply.User.Profile,
				"name":          reply.User.Name,
				"created_at":    reply.CreatedAt.Format("2006-01-02 15:04:05"),
				"relative_time": utils.TimeAgo(reply.CreatedAt),
			})
		}

		comments = append(comments, gin.H{
			"id":            comment.ID,
			"content":       comment.Content,
			"user_id":       comment.UserID,
			"parent_id":     comment.ParentID,
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

	response := gin.H{
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
		"like":          forum.LikesCount,
		"liked":         liked,
		"relative_time": utils.TimeAgo(forum.CreatedAt),
		"tags":          tags,
		"comments":      comments,
	}

	c.JSON(http.StatusOK, response)
}

func UpdateForum(c *gin.Context) {
	forumID := c.Param("id")

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

	var forum models.Forum
	if err := database.DB.First(&forum, forumID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Forum not found"})
		return
	}

	// Pastikan hanya pembuat forum atau admin yang dapat mengedit
	if forum.UserID != uint(userData.ID) && userData.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own forum"})
		return
	}

	// Batas waktu 30 menit untuk pembaruan
	timeSinceCreation := time.Since(forum.CreatedAt)
	if timeSinceCreation > 30*time.Minute {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forum can only be updated within 30 minutes of creation"})
		return
	}

	// Update data forum
	newTitle := c.PostForm("title")
	if newTitle != "" {
		forum.Title = newTitle
	}

	categoryIDStr := c.PostForm("category_id")
	if categoryIDStr != "" {
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
			return
		}
		categoryIDUint := uint(categoryID)
		forum.CategoryID = &categoryIDUint
	}

	file, err := c.FormFile("photo")
	if err == nil {
		uploadPath := fmt.Sprintf("./uploads/%s", file.Filename)
		if _, err := os.Stat("./uploads"); os.IsNotExist(err) {
			os.MkdirAll("./uploads", os.ModePerm)
		}
		c.SaveUploadedFile(file, uploadPath)
		forum.Photo = fmt.Sprintf("/uploads/%s", file.Filename)
	}

	if err := database.DB.Save(&forum).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update forum", "details": err.Error()})
		return
	}

	database.DB.Preload("User").Preload("Category").First(&forum, forum.ID)

	response := models.ForumResponse{
		ID:           int(forum.ID),
		Title:        forum.Title,
		Photo:        forum.Photo,
		UserID:       forum.UserID,
		Username:     forum.User.Username,
		CategoryID:   *forum.CategoryID,
		CategoryName: forum.Category.Name,
		RelativeTime: utils.TimeAgo(forum.CreatedAt),
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Forum updated successfully",
		"forum":   response,
	})
}

func DeleteForum(c *gin.Context) {
	forumID := c.Param("id")

	// Cek autentikasi user
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

	// Cek apakah forum ada
	var forum models.Forum
	if err := database.DB.Preload("Comments.Replies").First(&forum, forumID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Forum not found"})
		return
	}

	// Hanya pemilik forum atau admin yang bisa menghapus
	if forum.UserID != uint(userData.ID) && userData.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to delete this forum"})
		return
	}

	// Hapus semua komentar dan balasannya
	for _, comment := range forum.Comments {
		for _, reply := range comment.Replies {
			if err := database.DB.Delete(&reply).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete reply", "details": err.Error()})
				return
			}
		}
		if err := database.DB.Delete(&comment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment", "details": err.Error()})
			return
		}
	}

	// Hapus forum
	if err := database.DB.Delete(&forum).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete forum", "details": err.Error()})
		return
	}

	// Kirim notifikasi ke pemilik forum jika dihapus oleh admin
	if userData.Role == "admin" && forum.UserID != uint(userData.ID) {
		notification := models.Notification{
			UserID:  forum.UserID,
			Content: "Forum Anda dengan judul '" + forum.Title + "' telah dihapus oleh admin karena melanggar kebijakan.",
			ForumID: forum.ID,
			IsRead:  false,
		}
		if err := database.DB.Create(&notification).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send notification", "details": err.Error()})
			return
		}
	}

	// Respons sukses
	c.JSON(http.StatusOK, gin.H{"message": "Forum deleted successfully"})
}

func GetForumStats(c *gin.Context) {
    var totalForums int64
    var weeklyForums int64

    // Total semua forum
    if err := database.DB.Model(&models.Forum{}).Count(&totalForums).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch total forums"})
        return
    }

    // Total forum yang dibuat minggu ini (mulai dari Senin)
    startOfWeek := time.Now().AddDate(0, 0, -int(time.Now().Weekday())+1).Truncate(24 * time.Hour)
    if err := database.DB.Model(&models.Forum{}).
        Where("created_at >= ?", startOfWeek).
        Count(&weeklyForums).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch weekly forums"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "total_forums":  totalForums,
        "weekly_forums": weeklyForums,
    })
}