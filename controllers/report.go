package controllers

import (
	"log"
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

    // Cek jika user mencoba mereport dirinya sendiri
    reporterID := userID.(uint)
    if reporterID == req.ReportedID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak dapat melaporkan diri sendiri"})
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
    if err := database.DB.Where("reporter_id = ? AND reported_id = ? AND status = 'pending'", reporterID, req.ReportedID).First(&existingReport).Error; err == nil {
        c.JSON(http.StatusConflict, gin.H{"error": "Laporan terhadap user ini sudah ada dan sedang diproses"})
        return
    }

    report := models.Report{
        ReporterID: reporterID,
        ReportedID: req.ReportedID,
        Reason:     req.Reason,
        Status:     "pending",
    }

    if err := database.DB.Create(&report).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim laporan"})
        return
    }

    // Buat notifikasi untuk reporter
    notification := models.Notification{
        UserID:  reporterID,
        Content: "Laporan Anda terhadap akun @" + reportedUser.Username + " telah berhasil dikirim dan sedang diproses oleh admin.",
        IsRead:  false,
        CreatedAt: time.Now(),
    }
    if err := database.DB.Create(&notification).Error; err != nil {
        // Log error tapi tidak gagalakan response utama
        log.Printf("Gagal membuat notifikasi: %v", err)
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Laporan berhasil dikirim dan menunggu review admin.",
        "report":  report,
    })
}

func GetPendingReports(c *gin.Context) {
    var reports []models.Report
    if err := database.DB.Preload("ReportedUser").Find(&reports).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports"})
        return
    }

    type ReportResponse struct {
        ID            uint        `json:"id"`
        ReporterID    uint        `json:"reporter_id"`
        ReportedID    uint        `json:"reported_id"`
        Reason        string      `json:"reason"`
        Status        string      `json:"status"`
        CreatedAt     time.Time   `json:"created_at"`
        ReportedUser  models.User `json:"reported_user"`
        SuspendUntil  *time.Time  `json:"suspend_until,omitempty"`
        SuspendCount  int         `json:"suspend_count"`
    }

    var pendingReports []ReportResponse
    var pendingSuspendedReports []ReportResponse
    for _, report := range reports {
        resp := ReportResponse{
            ID:           report.ID,
            ReporterID:   report.ReporterID,
            ReportedID:   report.ReportedID,
            Reason:       report.Reason,
            Status:       report.Status,
            CreatedAt:    report.CreatedAt,
            ReportedUser: report.ReportedUser,
            SuspendUntil: report.ReportedUser.SuspendUntil,
            SuspendCount: report.ReportedUser.SuspendCount,
        }
        if report.Status == "pending" {
            pendingReports = append(pendingReports, resp)
        } else if report.Status == "pending_suspended" {
            pendingSuspendedReports = append(pendingSuspendedReports, resp)
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "pending":           pendingReports,
        "pending_suspended": pendingSuspendedReports,
    })
}

func ReviewReport(c *gin.Context) {
    role, exists := c.Get("role")
    if !exists || role.(string) != "admin" {
        c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized"})
        return
    }

    var req struct {
        ReportID uint   `json:"report_id"`
        Action   string `json:"action"` // "approve", "reject", "extend"
        Days     int    `json:"days"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    var report models.Report
    if err := database.DB.First(&report, req.ReportID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Laporan tidak ditemukan"})
        return
    }

    var user models.User
    if err := database.DB.First(&user, report.ReportedID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Pegguna yang dilaporkan tidak ditemukan"})
        return
    }

    if req.Action == "reject" {
        report.Status = "rejected"
        database.DB.Save(&report)
        c.JSON(http.StatusOK, gin.H{"message": "Laporan ditolak"})
        return
    }

    if req.Action == "approve" {
        if user.Status == "suspended" && user.SuspendUntil != nil && time.Now().Before(*user.SuspendUntil) {
            report.Status = "pending_suspended"
            database.DB.Save(&report)
            c.JSON(http.StatusOK, gin.H{
                "message": "Akun sedang disuspend, laporan ditunda hingga masa suspend selesai.",
            })
            return
        }

        // Hitung semua laporan pending untuk user ini
        var pendingReports []models.Report
        if err := database.DB.Where("reported_id = ? AND status = 'pending'", report.ReportedID).Find(&pendingReports).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung laporan pending"})
            return
        }

        suspendUntil := time.Now().Add(time.Duration(req.Days) * 24 * time.Hour)
        user.Status = "suspended"
        user.SuspendUntil = &suspendUntil
        user.SuspendDuration = req.Days
        user.SuspendCount++ // Tambah jumlah suspend

        // Update semua laporan pending menjadi approved
        for _, pendingReport := range pendingReports {
            pendingReport.Status = "approved"
            database.DB.Save(&pendingReport)
        }

        database.DB.Save(&user)

        c.JSON(http.StatusOK, gin.H{
            "message":         "Penggguna disuspend",
            "user_id":         user.ID,
            "suspend_until":   suspendUntil.Format("2006-01-02 15:04:05"),
            "suspend_duration": req.Days,
            "suspend_count":   user.SuspendCount,
            "resolved_reports": len(pendingReports),
        })
        return
    }

    if req.Action == "extend" && user.Status == "suspended" && user.SuspendUntil != nil {
        newSuspendUntil := user.SuspendUntil.Add(time.Duration(req.Days) * 24 * time.Hour)
        user.SuspendUntil = &newSuspendUntil
        user.SuspendDuration += req.Days
        user.SuspendCount++  
        report.Status = "approved"

        database.DB.Save(&user)
        database.DB.Save(&report)

        c.JSON(http.StatusOK, gin.H{
            "message":         "Suspend diperpanjang",
            "user_id":         user.ID,
            "suspend_until":   newSuspendUntil.Format("2006-01-02 15:04:05"),
            "suspend_duration": user.SuspendDuration,
            "suspend_count":   user.SuspendCount,
        })
        return
    }

    c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal memproses laporan"})
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

    var forum models.Forum
    if err := database.DB.First(&forum, req.ForumID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Forum post tidak ditemukan"})
        return
    }

    reporterID := userID.(uint)
    if forum.UserID == reporterID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak dapat melaporkan postingan Anda sendiri"})
        return
    }

    if strings.TrimSpace(req.Reason) == "" {
        req.Reason = "Melanggar aturan komunitas"
    }

    var existingReport models.ForumReport
    if err := database.DB.Where("reporter_id = ? AND forum_id = ? AND status = 'pending'", reporterID, req.ForumID).First(&existingReport).Error; err == nil {
        c.JSON(http.StatusConflict, gin.H{"error": "Laporan terhadap postingan ini sudah ada dan sedang diproses"})
        return
    }

    report := models.ForumReport{
        ReporterID: reporterID,
        ForumID:    req.ForumID,
        Reason:     req.Reason,
        Status:     "pending",
        CreatedAt:  time.Now(),
    }

    if err := database.DB.Create(&report).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim laporan"})
        return
    }

    // Notifikasi ke pelapor: Sedang Diproses
    notification := models.Notification{
        UserID:    reporterID,
        ForumID:   req.ForumID,
        Content:   "Laporan Anda untuk forum '" + forum.Title + "' telah diterima dan sedang diproses.",
        IsRead:    false,
        CreatedAt: time.Now(),
    }
    if err := database.DB.Create(&notification).Error; err != nil {
        log.Printf("Gagal membuat notifikasi: %v", err)
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

func HandleForumReport(c *gin.Context) {
    // Cek autentikasi admin
    user, exists := c.Get("user")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    userData, ok := user.(models.User)
    if !ok || userData.Role != "admin" {
        c.JSON(http.StatusForbidden, gin.H{"error": "Hanya admin yang dapat mengelola laporan"})
        return
    }

    reportID := c.Param("id")
    var req struct {
        Action string `json:"action"` // "accept" atau "reject"
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
        return
    }

    var report models.ForumReport
    if err := database.DB.Preload("Forum").First(&report, reportID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Laporan tidak ditemukan"})
        return
    }

    if report.Status != "pending" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Laporan ini sudah diproses"})
        return
    }

    switch req.Action {
    case "accept":
        // Update status laporan
        report.Status = "accepted"
        if err := database.DB.Save(&report).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status laporan"})
            return
        }

        // Hapus forum
        var forum models.Forum
        if err := database.DB.Preload("Comments.Replies").First(&forum, report.ForumID).Error; err != nil {
            c.JSON(http.StatusNotFound, gin.H{"error": "Forum tidak ditemukan"})
            return
        }

        for _, comment := range forum.Comments {
            for _, reply := range comment.Replies {
                database.DB.Delete(&reply)
            }
            database.DB.Delete(&comment)
        }
        if err := database.DB.Delete(&forum).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus forum"})
            return
        }

        // Notifikasi ke pelapor
        notificationReporter := models.Notification{
            UserID:    report.ReporterID,
            ForumID:   report.ForumID,
            Content:   "Laporan Anda untuk forum '" + forum.Title + "' diterima. Forum telah dihapus karena melanggar aturan.",
            IsRead:    false,
            CreatedAt: time.Now(),
        }
        if err := database.DB.Create(&notificationReporter).Error; err != nil {
            log.Printf("Gagal membuat notifikasi untuk pelapor: %v", err)
        }

        // Notifikasi ke pemilik forum (sudah ada logika serupa di DeleteForum)
        notificationOwner := models.Notification{
            UserID:    forum.UserID,
            ForumID:   forum.ID,
            Content:   "Forum Anda '" + forum.Title + "' telah dihapus oleh admin karena melanggar kebijakan.",
            IsRead:    false,
            CreatedAt: time.Now(),
        }
        if err := database.DB.Create(&notificationOwner).Error; err != nil {
            log.Printf("Gagal membuat notifikasi untuk pemilik forum: %v", err)
        }

        c.JSON(http.StatusOK, gin.H{"message": "Laporan diterima dan forum dihapus"})

    case "reject":
        // Update status laporan
        report.Status = "rejected"
        if err := database.DB.Save(&report).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status laporan"})
            return
        }

        // Notifikasi ke pelapor
        notification := models.Notification{
            UserID:    report.ReporterID,
            ForumID:   report.ForumID,
            Content:   "Laporan Anda untuk forum '" + report.Forum.Title + "' ditolak. Forum tidak melanggar aturan.",
            IsRead:    false,
            CreatedAt: time.Now(),
        }
        if err := database.DB.Create(&notification).Error; err != nil {
            log.Printf("Gagal membuat notifikasi untuk pelapor: %v", err)
        }

        c.JSON(http.StatusOK, gin.H{"message": "Laporan ditolak, forum tetap aktif"})

    default:
        c.JSON(http.StatusBadRequest, gin.H{"error": "Aksi tidak valid, gunakan 'accept' atau 'reject'"})
    }
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