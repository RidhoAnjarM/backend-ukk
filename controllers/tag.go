package controllers

import (
	"backend/database"
	"backend/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetAllTags fetches tags from the database based on a search query
func GetAllTags(search string) ([]models.Tag, error) {
	var tags []models.Tag
	query := database.DB.Where("name ILIKE ?", search+"%").Limit(5).Find(&tags)
	if query.Error != nil {
		return nil, query.Error
	}
	return tags, nil
}

// CreateTag inserts a new tag into the database
func CreateTag(name string) (models.Tag, error) {
	newTag := models.Tag{Name: name}
	query := database.DB.Create(&newTag)
	if query.Error != nil {
		return models.Tag{}, query.Error
	}
	return newTag, nil
}

// TagExists checks if a tag already exists in the database
func TagExists(name string) (bool, error) {
	var count int64
	query := database.DB.Model(&models.Tag{}).Where("name = ?", name).Count(&count)
	if query.Error != nil {
		return false, query.Error
	}
	return count > 0, nil
}

// Handler untuk GET /api/tags
func GetTags(c *gin.Context) {
	search := c.Query("q")

	if search == "" {
		c.JSON(http.StatusOK, []models.Tag{})
		return
	}

	tags, err := GetAllTags(search)
	if err != nil {
		log.Println("Error querying database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching tags"})
		return
	}

	c.JSON(http.StatusOK, tags)
}

// Handler untuk POST /api/tags
func CreateTagHandler(c *gin.Context) {
	var request struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if request.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tag name cannot be empty"})
		return
	}

	exists, err := TagExists(request.Name)
	if err != nil {
		log.Println("Database error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Tag already exists"})
		return
	}

	tag, err := CreateTag(request.Name)
	if err != nil {
		log.Println("Error inserting tag:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tag created",
		"tag": tag,
	})
}

func GetTagsAll(c *gin.Context) {
	var tags []models.Tag
	if err := database.DB.Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	c.JSON(http.StatusOK, tags)
}

func GetPopularTags(c *gin.Context) {
	var tags []models.Tag

	if err := database.DB.
		Select("id, name, COALESCE(usage_count, 0) AS usage_count").
		Order("usage_count DESC").
		Limit(10).
		Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch popular tags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"popular_tags": tags})
}


// ResetTagUsage - Atur ulang usage_count menjadi 0 (jalankan setiap minggu)
func ResetTagUsage() {
	database.DB.Model(&models.Tag{}).Update("usage_count", 0)
}

// ScheduleWeeklyTagReset - Menjalankan reset setiap minggu (gunakan goroutine)
func ScheduleWeeklyTagReset() {
	ticker := time.NewTicker(7 * 24 * time.Hour) 
	go func() {
		for range ticker.C {
			ResetTagUsage()
		}
	}()
}