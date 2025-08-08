package middleware

import (
	"github.com/gin-gonic/gin"
	"errors"
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"work/golang_api/models"
	"work/golang_api/db"
)
// Basic auth func
func BasicAuthMiddleware(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, pass, ok := c.Request.BasicAuth()
		if !ok || !checkUsernameAndPassword(user, pass) {
			c.Header("WWW-Authenticate", `Basic realm="Please enter your username and password for this site"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		handler(c)
	}
}

func checkUsernameAndPassword(username, password string) bool {
	var user models.User // Define a User struct

	err := db.DB.Where("username = ? AND password = ?", username, password).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // Check for record not found specifically
			return false // User not found (valid login attempt but no match)
		}
		fmt.Println("Failed Login !")
		return false // Other errors
	}

	return true // User found and credentials match
}
