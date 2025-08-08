package models

import (
	"gorm.io/gorm"
)

type Conf struct {
	gorm.Model
	NoRepeatTime string `json:"no_repeat_time"`
}
