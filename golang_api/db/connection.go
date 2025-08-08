package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"work/golang_api/models"
)

var DB *gorm.DB

func Init() {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
				logger.Config{
					SlowThreshold:             200 * time.Millisecond,
					LogLevel:                  logger.Info,
					IgnoreRecordNotFoundError: true,
					Colorful:                  true,
				},
	)

	var err error
	DB, err = gorm.Open(sqlite.Open("users.db"), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil || DB == nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	fmt.Println("Database connection established")

	// Migrate models
	autoMigrate()
	ensureDefaultRecords()
}

func autoMigrate() {
	if err := DB.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("AutoMigrate User error: %v", err)
	}
	fmt.Println("User model migrated")

	if err := DB.AutoMigrate(&models.Content{}); err != nil {
		log.Fatalf("AutoMigrate Content error: %v", err)
	}
	fmt.Println("Content model migrated")

	if err := DB.AutoMigrate(&models.Conf{}); err != nil {
		log.Fatalf("AutoMigrate Conf error: %v", err)
	}
	fmt.Println("Conf model migrated")
}

func ensureDefaultRecords() {
	var userCount int64
	DB.Model(&models.User{}).Count(&userCount)
	if userCount == 0 {
		DB.Create(&models.User{Username: "control", Password: "m1ll1c3nt"})
		fmt.Println("Default user created")
	} else {
		fmt.Println("User table already has records")
	}

	var confCount int64
	DB.Model(&models.Conf{}).Count(&confCount)
	if confCount > 1 {
		DB.Where("id NOT IN (?)", DB.Model(&models.Conf{}).Select("id").Limit(1)).Delete(&models.Conf{})
		fmt.Println("Cleared extra entries in the conf table")
	} else if confCount == 0 {
		DB.Create(&models.Conf{NoRepeatTime: "18"})
		fmt.Println("Added default conf record")
	}
}
