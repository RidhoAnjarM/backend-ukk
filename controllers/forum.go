package controllers

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"backend/database"
	"backend/models"
	"backend/utils"
)

func CreateForum(c *gin.Context) {
    var forum models.Forum

    // Validasi user
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

    // Upload file photo
    file, err := c.FormFile("photo")
    if err != nil {
        forum.Photo = ""
    } else {
        uploadPath := fmt.Sprintf("./uploads/%s", file.Filename)
        if _, err := os.Stat("./uploads"); os.IsNotExist(err) {
            os.MkdirAll("./uploads", os.ModePerm)
        }
        c.SaveUploadedFile(file, uploadPath)
        forum.Photo = fmt.Sprintf("/uploads/%s", file.Filename)
    }

    // Validasi title
    forum.Title = c.PostForm("title")
    if forum.Title == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Title is required"})
        return
    }

    // Validasi category_id (opsional)
    categoryIDStr := c.PostForm("category_id")
    if categoryIDStr != "" {
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil || categoryID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Category ID"})
			return
		}
	
		// Periksa apakah category_id valid di database
		var category models.Category
		if err := database.DB.First(&category, categoryID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}
	
		forum.CategoryID = &category.ID
	} else {
		forum.CategoryID = nil // Jika tidak diisi, set ke NULL
	}
	

    // Simpan forum ke database
    if err := database.DB.Create(&forum).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create forum", "details": err.Error()})
        return
    }

    // Load data forum dengan relasi
    database.DB.Preload("User").Preload("Category").First(&forum, forum.ID)

    response := models.ForumCreateResponse{
        ID:           uint(forum.ID),
        Title:        forum.Title,
        Photo:        forum.Photo,
        Username:     forum.User.Username,
        CategoryName: forum.Category.Name,
        CreatedAt:    forum.CreatedAt.Format("2006-01-02 15:04:05"),
        RelativeTime: utils.TimeAgo(forum.CreatedAt),
    }

    c.JSON(http.StatusCreated, gin.H{
        "message": "Forum created successfully",
        "forum":   response,
    })
}


func GetAllForums(c *gin.Context) {
	var forums []models.Forum
	// Preload user, category, comments, dan replies beserta user yang membuat komentar dan reply
	if err := database.DB.Preload("User").Preload("Category").Preload("Comments.User").Preload("Comments.Replies.User").Order("created_at desc").Find(&forums).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch forums"})
		return
	}

	// Shuffle the forums except the latest ones
	if len(forums) > 1 {
		latestForum := forums[0]
		remainingForums := forums[1:]
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(remainingForums), func(i, j int) {
			remainingForums[i], remainingForums[j] = remainingForums[j], remainingForums[i]
		})
		forums = append([]models.Forum{latestForum}, remainingForums...)
	}

	var response []gin.H
	for _, forum := range forums {
		// Membuat daftar komentar dengan reply
		var comments []gin.H
		for _, comment := range forum.Comments {
			// Menyusun daftar reply untuk komentar ini
			var replies []gin.H
			for _, reply := range comment.Replies {
				replies = append(replies, gin.H{
					"id":           reply.ID,
					"content":      reply.Content,
					"user_id":      reply.UserID,
					"username":     reply.User.Username,
					"profile":      reply.User.Profile,
					"created_at":   reply.CreatedAt.Format("2006-01-02 15:04:05"),
					"relative_time": utils.TimeAgo(reply.CreatedAt),
				})
			}

			// Menyusun data komentar beserta reply-nya
			comments = append(comments, gin.H{
				"id":            comment.ID,
				"content":       comment.Content,
				"user_id":       comment.UserID,
				"username":      comment.User.Username,
				"profile":       comment.User.Profile,
				"created_at":    comment.CreatedAt.Format("2006-01-02 15:04:05"),
				"relative_time": utils.TimeAgo(comment.CreatedAt),
				"replies":       replies,  // Menambahkan reply pada komentar
			})
		}

		// Menyusun data forum
		response = append(response, gin.H{
			"id":            forum.ID,
			"title":         forum.Title,
			"photo":         forum.Photo,
			"user_id":       forum.UserID,
			"username":      forum.User.Username,
			"profile":       forum.User.Profile,
			"category_id":   forum.CategoryID,
			"category_name": forum.Category.Name,
			"relative_time": utils.TimeAgo(forum.CreatedAt),
			"comments":      comments,  // Menambahkan komentar beserta reply-nya
		})
	}

	c.JSON(http.StatusOK, response)
}

func GetForumByID(c *gin.Context) {
	id := c.Param("id")
	var forum models.Forum

	// Preload user, category, comments, dan replies beserta user yang membuat komentar dan reply
	if err := database.DB.Preload("User").Preload("Category").Preload("Comments.User").Preload("Comments.Replies.User").First(&forum, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Forum not found"})
		return
	}

	// Membuat daftar komentar dengan reply
	var comments []gin.H
	for _, comment := range forum.Comments {
		// Menyusun daftar reply untuk komentar ini
		var replies []gin.H
		for _, reply := range comment.Replies {
			replies = append(replies, gin.H{
				"id":        reply.ID,
				"content":   reply.Content,
				"user_id":   reply.UserID,
				"username":  reply.User.Username,
				"profile":   reply.User.Profile,
				"created_at": reply.CreatedAt.Format("2006-01-02 15:04:05"),
				"relative_time": utils.TimeAgo(reply.CreatedAt),
			})
		}

		// Menyusun data komentar beserta reply-nya
		comments = append(comments, gin.H{
			"id":            comment.ID,
			"content":       comment.Content,
			"user_id":       comment.UserID,
			"username":      comment.User.Username,
			"profile":       comment.User.Profile,
			"created_at":    comment.CreatedAt.Format("2006-01-02 15:04:05"),
			"relative_time": utils.TimeAgo(comment.CreatedAt),
			"replies":       replies,  // Menambahkan reply pada komentar
		})
	}

	// Menyusun data forum
	response := gin.H{
		"id":            forum.ID,
		"title":         forum.Title,
		"photo":         forum.Photo,
		"user_id":       forum.UserID,
		"username":      forum.User.Username,
		"profile":       forum.User.Profile,
		"category_id":   forum.CategoryID,
		"category_name": forum.Category.Name,
		"relative_time": utils.TimeAgo(forum.CreatedAt),
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
	if err := database.DB.Preload("Comments.Replies").First(&forum, forumID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Forum not found"})
		return
	}

	// Pastikan hanya pembuat forum atau admin yang dapat menghapus
	if forum.UserID != uint(userData.ID) && userData.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own forum"})
		return
	}

	// Hapus semua komentar dan reply terkait forum
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

	if err := database.DB.Delete(&forum).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete forum", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Forum deleted successfully"})
}