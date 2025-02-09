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
		api.GET("/users/:id", controllers.GetUserByID)
		api.PUT("/users/:id", controllers.UpdateUser)
		api.DELETE("/users/:id", controllers.DeleteUser)
		api.GET("/profile", middlewares.AuthMiddleware(), controllers.GetUsername)

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
			forum.GET("/", controllers.GetAllForums)
			forum.GET("/:id", controllers.GetForumByID)
			forum.PUT("/:id", middlewares.AuthMiddleware(), controllers.UpdateForum)
			forum.DELETE("/:id", middlewares.AuthMiddleware(), controllers.DeleteForum)
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
	}
}
