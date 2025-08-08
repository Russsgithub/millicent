package models

import (
	"errors"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string `json:"username"`
	Password string `json:"password"`
}


// Validate User item
func (c User) Validate() error {
	// Add your validation rules here
	if c.Username == "" {
		return errors.New("title is required")
	}
	if c.Password == "" {
		return errors.New("artist is required")
	}
	return nil
}
