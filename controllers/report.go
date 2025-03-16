package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"backend/database"
	"backend/models"
)

func CheckExistingReport(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	reportedID := c.Query("reported_id")
	if reportedID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reported ID diperlukan"})
		return
	}

	var existingReport models.Report
	if err := database.DB.Where("reporter_id = ? AND reported_id = ? AND status = 'pending'", userID, reportedID).First(&existingReport).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"exists": true})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exists": false})
}

func ReportUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.Report

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if strings.TrimSpace(req.Reason) == "" {
		req.Reason = "Melanggar aturan komunitas"
	}

	var reportedUser models.User
	if err := database.DB.First(&reportedUser, req.ReportedID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User yang dilaporkan tidak ditemukan"})
		return
	}

	var existingReport models.Report
	if err := database.DB.Where("reporter_id = ? AND reported_id = ? AND status = 'pending'", userID, req.ReportedID).First(&existingReport).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Laporan terhadap user ini sudah ada dan sedang diproses"})
		return
	}

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

	if err := database.DB.Preload("ReportedUser").Where("status = ?", "pending").Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports"})
		return
	}

	type ReportResponse struct {
		ID           uint        `json:"id"`
		ReporterID   uint        `json:"reporter_id"`
		ReportedID   uint        `json:"reported_id"`
		Reason       string      `json:"reason"`
		Status       string      `json:"status"`
		CreatedAt    time.Time   `json:"created_at"`
		ReportedUser models.User `json:"reported_user"`
	}

	var response []ReportResponse
	for _, report := range reports {
		response = append(response, ReportResponse{
			ID:           report.ID,
			ReporterID:   report.ReporterID,
			ReportedID:   report.ReportedID,
			Reason:       report.Reason,
			Status:       report.Status,
			CreatedAt:    report.CreatedAt,
			ReportedUser: report.ReportedUser,
		})
	}

	c.JSON(http.StatusOK, response)
}

func ReviewReport(c *gin.Context) {
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
		report.Status = "rejected"
		database.DB.Save(&report)
		c.JSON(http.StatusOK, gin.H{"message": "Report rejected"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, report.ReportedID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	suspendUntil := time.Now().Add(time.Duration(req.Days) * 24 * time.Hour)
	user.Status = "suspended"
	user.SuspendUntil = &suspendUntil
	user.SuspendDuration = req.Days
	report.Status = "approved"

	database.DB.Save(&user)
	database.DB.Save(&report)

	c.JSON(http.StatusOK, gin.H{
		"message":         "User suspended successfully",
		"user_id":         user.ID,
		"suspend_until":   suspendUntil.Format("2006-01-02 15:04:05"),
		"suspend_duration": req.Days,
	})
}


func CheckExistingForumReport(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	forumID := c.Query("forum_id")
	if forumID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Forum ID diperlukan"})
		return
	}

	var existingReport models.ForumReport
	if err := database.DB.Where("reporter_id = ? AND forum_id = ? AND status = 'pending'", userID, forumID).First(&existingReport).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"exists": true})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exists": false})
}

func ReportForumPost(c *gin.Context) {
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

	if err := database.DB.Preload("Forum.User").Where("status = ?", "pending").Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch forum reports"})
		return
	}

	type ForumReportResponse struct {
		ID         uint         `json:"id"`
		ReporterID uint         `json:"reporter_id"`
		ForumID    uint         `json:"forum_id"`
		Reason     string       `json:"reason"`
		Status     string       `json:"status"`
		CreatedAt  time.Time    `json:"created_at"`
		Forum      models.Forum `json:"forum"`
	}

	var response []ForumReportResponse
	for _, report := range reports {
		if report.Forum.ID == 0 {
			database.DB.Delete(&report)
			continue
		}

		response = append(response, ForumReportResponse{
			ID:         report.ID,
			ReporterID: report.ReporterID,
			ForumID:    report.ForumID,
			Reason:     report.Reason,
			Status:     report.Status,
			CreatedAt:  report.CreatedAt,
			Forum:      report.Forum,
		})
	}

	c.JSON(http.StatusOK, response)
}

func DeleteReport(c *gin.Context) {
	reportID := c.Param("id")

	var report models.ForumReport
	if err := database.DB.First(&report, reportID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Laporan tidak ditemukan"})
		return
	}

	if err := database.DB.Delete(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus laporan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Laporan berhasil dihapus",
		"report":  report,
	})
}