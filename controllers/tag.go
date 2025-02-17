package controllers

import (
	"backend/database"
	"backend/models"
	"log"
	"net/http"

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

	// Pastikan selalu return array, bahkan jika search kosong
	if search == "" {
		c.JSON(http.StatusOK, []models.Tag{})
		return
	}

	// Query database
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

	// Bind JSON request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Validasi nama tag
	if request.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tag name cannot be empty"})
		return
	}

	// Cek apakah tag sudah ada
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

	// Buat tag baru
	tag, err := CreateTag(request.Name)
	if err != nil {
		log.Println("Error inserting tag:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag"})
		return
	}

	// Response dengan data tag yang baru dibuat
	c.JSON(http.StatusCreated, gin.H{
		"message": "Tag created",
		"tag": tag,
	})
}
