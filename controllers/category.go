package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"backend/models"
	"backend/database"
)

func CreateCategory(c *gin.Context) {
	var category models.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := database.DB.Create(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category created", "data": category})
}

func GetCategories(c *gin.Context) {
	var categories []models.Category
	if err := database.DB.Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	c.JSON(http.StatusOK, categories)
}

func GetCategoryByID(c *gin.Context) {
    categoryID := c.Param("id") 
    var category models.Category

    if err := database.DB.First(&category, categoryID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"data": category})
}

func UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	var category models.Category
	if err := database.DB.First(&category, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := database.DB.Save(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category updated", "data": category})
}

func DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	var category models.Category
	if err := database.DB.First(&category, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	if err := database.DB.Delete(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted"})
}

// GetPopularCategories - Ambil kategori berdasarkan jumlah penggunaan tertinggi
func GetPopularCategories(c *gin.Context) {
	var categories []models.Category

	if err := database.DB.
		Select("id, name, COALESCE(usage_count, 0) AS usage_count").
		Order("usage_count DESC").
		Limit(10).
		Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch popular categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"popular_categories": categories})
}

func ResetCategoryUsage() {
	database.DB.Model(&models.Category{}).Update("usage_count", 0)
}

// ScheduleWeeklyReset - Menjalankan reset setiap minggu (gunakan goroutine)
func ScheduleWeeklyReset() {
	ticker := time.NewTicker(7 * 24 * time.Hour) // Setiap 7 hari
	go func() {
		for range ticker.C {
			ResetCategoryUsage()
		}
	}()
}
