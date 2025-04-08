package controllers

import (
	"backend/database"
	"backend/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)


func GetAllTags(search string) ([]models.Tag, error) {
	var tags []models.Tag
	query := database.DB.Where("name ILIKE ?", search+"%").Limit(5).Find(&tags)
	if query.Error != nil {
		return nil, query.Error
	}
	return tags, nil
}


func CreateTag(name string) (models.Tag, error) {
	newTag := models.Tag{Name: name}
	query := database.DB.Create(&newTag)
	if query.Error != nil {
		return models.Tag{}, query.Error
	}
	return newTag, nil
}


func TagExists(name string) (bool, error) {
	var count int64
	query := database.DB.Model(&models.Tag{}).Where("name = ?", name).Count(&count)
	if query.Error != nil {
		return false, query.Error
	}
	return count > 0, nil
}


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
        Select("id, name, COALESCE(usage_count, 0) AS usage_count, created_at").
        Where("usage_count > 0").  
        Order("usage_count DESC").
        Limit(10).
        Find(&tags).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch popular tags"})
        return
    }

    if len(tags) == 0 {
        c.JSON(http.StatusOK, gin.H{"message": "Tidak ada tag populer di minggu ini"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"popular_tags": tags})
}


func ResetTagUsage() {
    log.Println("Running updated ResetTagUsage with Updates method")
	result := database.DB.Exec("UPDATE tags SET usage_count = 0")
    if result.Error != nil {
        log.Println("Error resetting tag usage:", result.Error)
        return
    }
    log.Printf("Successfully reset usage_count for %d tags", result.RowsAffected)

    var resetLog models.ResetLog
    if err := database.DB.First(&resetLog).Error; err != nil {
        resetLog = models.ResetLog{LastResetAt: time.Now()}
        database.DB.Create(&resetLog)
    } else {
        resetLog.LastResetAt = time.Now()
        database.DB.Save(&resetLog)
    }
    log.Println("Reset time updated to:", resetLog.LastResetAt)
}

// Fungsi untuk cek apakah perlu reset
func ShouldReset(lastResetTime time.Time, interval time.Duration) bool {
    return time.Since(lastResetTime) >= interval
}

// Fungsi untuk cek dan reset saat start
func CheckAndResetTags() {
    var resetLog models.ResetLog
    interval := 7 * 24 * time.Hour 

    if err := database.DB.First(&resetLog).Error; err != nil {
        ResetTagUsage()
        return
    }

    if ShouldReset(resetLog.LastResetAt, interval) {
        ResetTagUsage()
    } else {
        log.Println("No reset needed yet. Last reset was at:", resetLog.LastResetAt)
    }
}

func CheckAndResetTagsPeriodically() {
    interval := 7 * 24 * time.Hour  
    ticker := time.NewTicker(5 * time.Minute) 
    go func() {
        for range ticker.C {
            var resetLog models.ResetLog
            if err := database.DB.First(&resetLog).Error; err != nil {
                ResetTagUsage()
                continue
            }
            if ShouldReset(resetLog.LastResetAt, interval) {
                ResetTagUsage()
            }
        }
    }()
}

func ResetTagsManual(c *gin.Context) {
    ResetTagUsage()
    c.JSON(http.StatusOK, gin.H{"message": "Tag usage reset manually"})
}