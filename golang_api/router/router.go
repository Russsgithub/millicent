package router

import (
	"github.com/gin-gonic/gin"
	"work/golang_api/controllers"
	"work/golang_api/middleware"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.LoadHTMLGlob("templates/**/*")
	r.StaticFS("/static", gin.Dir("static", false))

	r.GET("/", controllers.Home)
	r.GET("/admin", controllers.Admin)
	r.GET("/adminUpload", middleware.BasicAuthMiddleware(controllers.AdminUpload))
	r.GET("/playout", controllers.LastPlayed)
	r.GET("/search", controllers.SuggestedSearch)
	r.GET("/next", middleware.BasicAuthMiddleware(controllers.GetNext))
	r.POST("/upload", middleware.BasicAuthMiddleware(controllers.Upload))

	// Create grouped routes for content and users:
	contentGroup := r.Group("/content")
	ContentRoutes(contentGroup)

	userGroup := r.Group("/users")
	UserRoutes(userGroup)

	return r
}
