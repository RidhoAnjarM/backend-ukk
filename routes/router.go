package routes

import (
	"github.com/gin-gonic/gin"

	"backend/controllers"
	"backend/middlewares"
)

func SetupRouter(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/register", controllers.Register)
		api.POST("/login", controllers.Login)

		api.GET("/users", controllers.GetAllUsers)
		api.GET("/users-stats", controllers.GetUserStats)
		api.GET("/users/:id", middlewares.AuthMiddleware(), controllers.GetUserByID)
		api.PUT("/users/:id", controllers.UpdateUser)
		api.DELETE("/users/:id", controllers.DeleteUser)
		api.GET("/profile", middlewares.AuthMiddleware(), controllers.GetUsername)

		report := api.Group("/report")
		{
			report.GET("/akun", controllers.GetPendingReports) 
			report.POST("/akun/review", middlewares.AuthMiddleware(), controllers.ReviewReport)
			report.GET("/forum", controllers.GetPendingForumReports)
			report.DELETE("/forum/:id", middlewares.AuthMiddleware(), controllers.DeleteReport) // Bukan "/forum" saja
			
			report.POST("/akun", middlewares.AuthMiddleware(), controllers.ReportUser)
			report.POST("/forum", middlewares.AuthMiddleware(), controllers.ReportForumPost)

			report.GET("/akun/check", middlewares.AuthMiddleware(), controllers.CheckExistingReport)
			report.GET("/forum/check", middlewares.AuthMiddleware(), controllers.CheckExistingForumReport)
		}

		category := api.Group("/category")
		{
			category.POST("/", controllers.CreateCategory)
			category.GET("/", controllers.GetCategories)
			category.GET("/:id", controllers.GetCategoryByID)
			category.PUT("/:id", controllers.UpdateCategory)
			category.DELETE("/:id", controllers.DeleteCategory)
		}

		forum := api.Group("/forum")
		{
			forum.POST("/", middlewares.AuthMiddleware(), controllers.CreateForum)
			forum.GET("/", middlewares.AuthMiddleware(), controllers.GetAllForums)
			forum.GET("/:id", middlewares.AuthMiddleware(), controllers.GetForumByID)
			forum.PUT("/:id", middlewares.AuthMiddleware(), controllers.UpdateForum)
			forum.DELETE("/:id", middlewares.AuthMiddleware(), controllers.DeleteForum)
			forum.GET("/stats", controllers.GetForumStats)
		}

		comment := api.Group("/comment")
		{
			comment.POST("/", middlewares.AuthMiddleware(), controllers.AddComment)
			comment.GET("/", controllers.GetAllComments)
			comment.GET("/:id", controllers.GetCommentByID)
			comment.DELETE("/:id", middlewares.AuthMiddleware(), controllers.DeleteComment)
			comment.POST("/reply", middlewares.AuthMiddleware(), controllers.ReplyComment)
			comment.DELETE("/reply/:id", middlewares.AuthMiddleware(), controllers.DeleteReply)
		}

		notification := api.Group("/notification")
		{
			notification.GET("/", middlewares.AuthMiddleware(), controllers.GetNotifications)
			notification.PUT("/:id/read", middlewares.AuthMiddleware(), controllers.MarkNotificationAsRead)
			notification.PUT("/readall", middlewares.AuthMiddleware(), controllers.MarkAllNotificationsAsRead)
			notification.DELETE("/:id", middlewares.AuthMiddleware(), controllers.DeleteNotification)
			notification.DELETE("/all", middlewares.AuthMiddleware(), controllers.DeleteAllNotifications)
		}

		profile := api.Group("/profil")
		{
			profile.GET("/", middlewares.AuthMiddleware(), controllers.GetProfile)
		}

		tags := api.Group("/tags")
		{
			tags.POST("/", controllers.CreateTagHandler)
			tags.GET("/", controllers.GetTags)
			tags.GET("/all", controllers.GetTagsAll)
		}

		populer := api.Group("/populer")
		{
			populer.GET("/tag", controllers.GetPopularTags)
			populer.GET("/category", controllers.GetPopularCategories)

		}

		like := api.Group("/like")
		{
			like.POST("/", middlewares.AuthMiddleware(), controllers.LikeForum)
			like.GET("/", controllers.GetForumLikesCount)
		}
	}
}
