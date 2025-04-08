package controllers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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

	form, err := c.MultipartForm()
	if err != nil && err != http.ErrNotMultipart {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	var photoPaths []string
	if form != nil {
		photos := form.File["photos"]
		if len(photos) > 5 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 5 photos allowed"})
			return
		}

		for _, file := range photos {
			f, err := file.Open()
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open file"})
				return
			}
			defer f.Close()

			mime, err := mimetype.DetectReader(f)
			if err != nil || !strings.HasPrefix(mime.String(), "image/") {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type, only images allowed"})
				return
			}
			f.Seek(0, 0)
			uploadPath := fmt.Sprintf("./uploads/%s", file.Filename)
			if err := c.SaveUploadedFile(file, uploadPath); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save uploaded photo"})
				return
			}
			photoPaths = append(photoPaths, fmt.Sprintf("/uploads/%s", file.Filename))
		}
	}

	// Konversi photoPaths ke JSON
	if len(photoPaths) > 0 {
		photosJSON, err := json.Marshal(photoPaths)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal photos"})
			return
		}
		forum.Photos = photosJSON
	} else {
		forum.Photos = json.RawMessage("[]") 
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

	// Handle tags
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

	for _, tag := range tags {
		if err := tx.Model(&models.Tag{}).Where("id = ?", tag.ID).Update("usage_count", gorm.Expr("usage_count + 1")).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tag usage count"})
			return
		}
	}

	tx.Commit()

	database.DB.Preload("User").Preload("Tags").First(&forum, forum.ID)

	var photosResponse []string
	if err := json.Unmarshal(forum.Photos, &photosResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal photos for response"})
		return
	}

	response := gin.H{
		"id":            forum.ID,
		"title":         forum.Title,
		"description":   forum.Description,
		"photos":        photosResponse,  
		"username":      forum.User.Username,
		"name":          forum.User.Name,
		"profile":       forum.User.Profile,
		"tags":          forum.Tags,
		"created_at":    forum.CreatedAt.Format("2006-01-02 15:04:05"),
		"relative_time": utils.TimeAgo(forum.CreatedAt),
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
    if err := database.DB.Preload("User").Preload("Comments.User").Preload("Comments.Replies.User").Preload("Tags").Order("created_at desc").Find(&forums).Error; err != nil {
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
        // Cek status suspend user
        if forum.User.Status == "suspended" && forum.User.SuspendUntil != nil && time.Now().Before(*forum.User.SuspendUntil) {
            continue // Skip forum ini kalo user-nya lagi disuspend
        }

        // Update status user kalo suspend-nya udah selesai
        if forum.User.Status == "suspended" && forum.User.SuspendUntil != nil && time.Now().After(*forum.User.SuspendUntil) {
            forum.User.Status = "active"
            forum.User.SuspendUntil = nil
            forum.User.SuspendDuration = 0
            database.DB.Save(&forum.User)
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
            "photos":        forum.Photos,
            "photo":         forum.Photo,
            "user_id":       forum.UserID,
            "username":      forum.User.Username,
            "name":          forum.User.Name,
            "profile":       forum.User.Profile,
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

	if err := database.DB.Preload("User").
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
				"image_url":     reply.ImageURL,
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
			"image_url":     comment.ImageURL,
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
		"photos":        forum.Photos,
		"photo":         forum.Photo,
		"user_id":       forum.UserID,
		"username":      forum.User.Username,
		"name":          forum.User.Name,
		"profile":       forum.User.Profile,
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

	if forum.UserID != uint(userData.ID) && userData.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own forum"})
		return
	}

	timeSinceCreation := time.Since(forum.CreatedAt)
	if timeSinceCreation > 30*time.Minute {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forum can only be updated within 30 minutes of creation"})
		return
	}

	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update title
	newTitle := c.PostForm("title")
	if newTitle != "" {
		forum.Title = newTitle
	}

	// Update photos (optional)
	form, err := c.MultipartForm()
	if err == nil {
		photos := form.File["photos"]
		if len(photos) > 5 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 5 photos allowed"})
			return
		}

		var photoPaths []string
		if len(photos) > 0 {
			for _, file := range photos {
				f, err := file.Open()
				if err != nil {
					tx.Rollback()
					c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open file"})
					return
				}
				defer f.Close()

				mime, err := mimetype.DetectReader(f)
				if err != nil || !strings.HasPrefix(mime.String(), "image/") {
					tx.Rollback()
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type, only images allowed"})
					return
				}
				f.Seek(0, 0)

				uploadPath := fmt.Sprintf("./uploads/%d-%s", time.Now().UnixNano(), file.Filename) // Nama unik
				if _, err := os.Stat("./uploads"); os.IsNotExist(err) {
					os.MkdirAll("./uploads", os.ModePerm)
				}
				if err := c.SaveUploadedFile(file, uploadPath); err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save uploaded photo"})
					return
				}
				photoPaths = append(photoPaths, uploadPath)
			}

			// Konversi photoPaths ke JSON
			photosJSON, err := json.Marshal(photoPaths)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal photos"})
				return
			}
			forum.Photos = photosJSON // Hanya update jika ada foto baru
		}
	} else if err != http.ErrNotMultipart {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	if err := tx.Save(&forum).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update forum", "details": err.Error()})
		return
	}

	tx.Commit()

	database.DB.Preload("User").Preload("Tags").First(&forum, forum.ID)

	// Parse Photos kembali ke []string untuk response
	var photosResponse []string
	if err := json.Unmarshal(forum.Photos, &photosResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal photos for response"})
		return
	}

	response := gin.H{
		"id":            forum.ID,
		"title":         forum.Title,
		"description":   forum.Description,
		"photos":        photosResponse,
		"user_id":       forum.UserID,
		"username":      forum.User.Username,
		"name":          forum.User.Name,
		"profile":       forum.User.Profile,
		"relative_time": utils.TimeAgo(forum.CreatedAt),
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

func GetAllForumsNoLogin(c *gin.Context) {
	var forums []models.Forum
	if err := database.DB.Preload("User").Preload("Comments.User").Preload("Comments.Replies.User").Preload("Tags").Order("created_at desc").Find(&forums).Error; err != nil {
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
            "photos":        forum.Photos,
			"user_id":       forum.UserID,
			"username":      forum.User.Username,
			"name":          forum.User.Name,
			"profile":       forum.User.Profile,
			"relative_time": utils.TimeAgo(forum.CreatedAt),
			"like":          forum.LikesCount,
			"comments":      comments,
			"tags":          tags,
			"createAt":      forum.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}

func GetForumByIDNoLogin(c *gin.Context) {
	id := c.Param("id")
	var forum models.Forum

	// Ambil forum beserta relasi yang diperlukan
	if err := database.DB.Preload("User").
		Preload("Comments.User").
		Preload("Comments.Replies.User").
		Preload("Tags").
		First(&forum, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Forum not found"})
		return
	}

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
        "photos":        forum.Photos,
		"user_id":       forum.UserID,
		"username":      forum.User.Username,
		"name":          forum.User.Name,
		"profile":       forum.User.Profile,
		"like":          forum.LikesCount,
		"relative_time": utils.TimeAgo(forum.CreatedAt),
		"tags":          tags,
		"comments":      comments,
	}

	c.JSON(http.StatusOK, response)
}
