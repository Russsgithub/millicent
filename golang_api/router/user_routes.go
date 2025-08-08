package router

import (
	"work/golang_api/controllers"
	"work/golang_api/middleware"

	"github.com/gin-gonic/gin"
)

func UserRoutes(rg *gin.RouterGroup) {
	rg.GET("/", middleware.BasicAuthMiddleware(controllers.GetUsers))
	rg.GET("/:id", middleware.BasicAuthMiddleware(controllers.GetUserByID))
	rg.POST("/", middleware.BasicAuthMiddleware(controllers.PostUsers))
}
