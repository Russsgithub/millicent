package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"

	"work/golang_api/globals"
)


func Home(c *gin.Context) {
	c.HTML(http.StatusOK, "home.tmpl", gin.H{
		"title":   "millicent",
	"playout": globals.RecentlyPlayed,
	})
}
