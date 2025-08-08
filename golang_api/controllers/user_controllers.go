package controllers

import (
	"github.com/gin-gonic/gin"
	"errors"
	"net/http"

	"work/golang_api/models"
	"work/golang_api/db"

	"gorm.io/gorm"
)

// getUsers respond with list of users in as json
func GetUsers(c *gin.Context) {
	var users []models.User // Define a slice to hold User structs

	result := db.DB.Find(&users)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, users)
}

// postUsers add user from json recieved in request body
func PostUsers(c *gin.Context) {
	var newUser models.User // Define a User struct

	if err := c.BindJSON(&newUser); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the user struct based on defined fields (optional)
	if err := newUser.Validate(); err != nil { // Define Validate method in User struct (if needed)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a new user record using GORM's Create method
	result := db.DB.Create(&newUser)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.IndentedJSON(http.StatusCreated, newUser)
}

// Get user by id
func GetUserByID(c *gin.Context) {
	id := c.Param("id")

	var user models.User // Define a User struct

	err := db.DB.First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "user not found"})
			return
		}
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

