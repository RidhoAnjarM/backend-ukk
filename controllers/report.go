package controllers

import(
	"net/http"
	"time"
	"strings"

	"github.com/gin-gonic/gin"

	"backend/models"
	"backend/database"
)

func ReportUser(c *gin.Context) {
	// Ambil ID user dari token
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		ReportedID uint   `json:"reported_id"`
		Reason     string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Jika alasan kosong, isi default "Melanggar aturan komunitas"
	if strings.TrimSpace(req.Reason) == "" {
		req.Reason = "Melanggar aturan komunitas"
	}

	// Periksa apakah user yang dilaporkan ada
	var reportedUser models.User
	if err := database.DB.First(&reportedUser, req.ReportedID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User yang dilaporkan tidak ditemukan"})
		return
	}

	// Cek apakah sudah ada laporan pending terhadap user yang sama
	var existingReport models.Report
	if err := database.DB.Where("reporter_id = ? AND reported_id = ? AND status = 'pending'", userID, req.ReportedID).First(&existingReport).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Laporan terhadap user ini sudah ada dan sedang diproses"})
		return
	}

	// Simpan laporan baru
	report := models.Report{
		ReporterID: userID.(uint),
		ReportedID: req.ReportedID,
		Reason:     req.Reason,
		Status:     "pending",
	}

	if err := database.DB.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim laporan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Laporan berhasil dikirim dan menunggu review admin.",
		"report":  report,
	})
}


func GetPendingReports(c *gin.Context) {
	var reports []models.Report
	if err := database.DB.Where("status = ?", "pending").Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports"})
		return
	}
	c.JSON(http.StatusOK, reports)
}


func ReviewReport(c *gin.Context) {
	// Ambil role dari token
	role, exists := c.Get("role")
	if !exists || role.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		ReportID uint   `json:"report_id"`
		Action   string `json:"action"` // "approve" atau "reject"
		Days     int    `json:"days"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var report models.Report
	if err := database.DB.First(&report, req.ReportID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	if req.Action == "reject" {
		// Jika laporan ditolak, update status jadi "rejected"
		report.Status = "rejected"
		database.DB.Save(&report)
		c.JSON(http.StatusOK, gin.H{"message": "Report rejected"})
		return
	}

	// Jika laporan disetujui, suspend akun
	var user models.User
	if err := database.DB.First(&user, report.ReportedID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	suspendUntil := time.Now().Add(time.Duration(req.Days) * 24 * time.Hour)
	user.Status = "suspended"
	user.SuspendUntil = &suspendUntil
	report.Status = "approved"

	database.DB.Save(&user)
	database.DB.Save(&report)

	c.JSON(http.StatusOK, gin.H{
		"message":       "User suspended successfully",
		"user_id":       user.ID,
		"suspend_until": suspendUntil.Format("2006-01-02 15:04:05"),
	})
}

func ReportForumPost(c *gin.Context) {
	// Ambil ID user dari token
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		ForumID uint   `json:"forum_id"`
		Reason  string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if strings.TrimSpace(req.Reason) == "" {
		req.Reason = "Melanggar aturan komunitas"
	}

	var forum models.Forum
	if err := database.DB.First(&forum, req.ForumID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Forum post tidak ditemukan"})
		return
	}

	var existingReport models.ForumReport
	if err := database.DB.Where("reporter_id = ? AND forum_id = ? AND status = 'pending'", userID, req.ForumID).First(&existingReport).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Laporan terhadap postingan ini sudah ada dan sedang diproses"})
		return
	}

	// Simpan laporan baru
	report := models.ForumReport{
		ReporterID: userID.(uint),
		ForumID:    req.ForumID,
		Reason:     req.Reason,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	if err := database.DB.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim laporan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Laporan berhasil dikirim dan menunggu review admin.",
		"report":  report,
	})
}

func GetPendingForumReports(c *gin.Context) {
	var reports []models.ForumReport
	if err := database.DB.Where("status = ?", "pending").Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports"})
		return
	}
	c.JSON(http.StatusOK, reports)
}

