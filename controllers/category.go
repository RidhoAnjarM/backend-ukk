package controllers

import (
	"net/http"
	"time"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"

	"backend/models"
	"backend/database"
)

func CreateCategory(c *gin.Context) {
    var category models.Category

    // Ambil data dari form-data
    category.Name = c.PostForm("name")

    // Handle file upload
    file, err := c.FormFile("photo")
    if err == nil {
        // Buat folder uploads jika belum ada
        uploadPath := fmt.Sprintf("./uploads/%s", file.Filename)
        if _, err := os.Stat("./uploads"); os.IsNotExist(err) {
            os.MkdirAll("./uploads", os.ModePerm)
        }

        // Simpan file yang diunggah ke folder uploads
        if err := c.SaveUploadedFile(file, uploadPath); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save uploaded file"})
            return
        }

        // Simpan path foto ke dalam database
        category.Photo = fmt.Sprintf("/uploads/%s", file.Filename)
    }

    // Simpan data kategori ke database
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

    // Cari kategori berdasarkan ID
    if err := database.DB.First(&category, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
        return
    }

    // Ambil data dari form-data
    if name := c.PostForm("name"); name != "" {
        category.Name = name
    }

    // Handle file upload
    file, err := c.FormFile("photo")
    if err == nil {
        // Buat folder uploads jika belum ada
        uploadPath := fmt.Sprintf("./uploads/%s", file.Filename)
        if _, err := os.Stat("./uploads"); os.IsNotExist(err) {
            os.MkdirAll("./uploads", os.ModePerm)
        }

        // Simpan file yang diunggah ke folder uploads
        if err := c.SaveUploadedFile(file, uploadPath); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save uploaded file"})
            return
        }

        // Simpan path foto ke dalam database
        category.Photo = fmt.Sprintf("/uploads/%s", file.Filename)
    }

    // Update data kategori di database
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
		Select("id, name, photo, COALESCE(usage_count, 0) AS usage_count").
		Order("usage_count DESC").
		Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch popular categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"popular_categories": categories})
}

func ResetCategoryUsage() {
	database.DB.Model(&models.Category{}).Update("usage_count", 0)
}

func ScheduleWeeklyReset() {
	ticker := time.NewTicker(7 * 24 * time.Hour) 
	go func() {
		for range ticker.C {
			ResetCategoryUsage()
		}
	}()
}
