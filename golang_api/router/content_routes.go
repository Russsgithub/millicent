package router

import (
	"github.com/gin-gonic/gin"

	"work/golang_api/controllers"
	"work/golang_api/middleware"
)

func ContentRoutes(rg *gin.RouterGroup) {
	rg.GET("/", controllers.GetContents)
	rg.GET("/:id", middleware.BasicAuthMiddleware(controllers.GetContentByID))
	rg.DELETE("/:id", middleware.BasicAuthMiddleware(controllers.DeleteContentByID))
	rg.POST("/", middleware.BasicAuthMiddleware(controllers.PostContent))
	rg.PUT("/:id", middleware.BasicAuthMiddleware(controllers.PatchContent))
	rg.GET("/where", controllers.GetContentWhere)
}
