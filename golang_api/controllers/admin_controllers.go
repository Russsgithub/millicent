package controllers

import (
	"strconv"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"github.com/gin-gonic/gin"

	"github.com/google/uuid"

	"work/golang_api/helpers"
	"work/golang_api/models"
	"work/golang_api/db"
)

func AdminUpload(c *gin.Context) {
	c.HTML(http.StatusOK, "upload.tmpl", gin.H{
		"title": "Upload files",
	})
}

func Admin(c *gin.Context) {
	filterField := "stream_2"
	filterValue := c.Query("stream")
	limitStr := c.Query("per_page")
	pageStr := c.Query("page")

	if filterValue == "all" {
		filterValue = ""
	}

	//Set defaults to return all content
	const defaultLimit = -1
	const defaultPage = 1

	// Convert strings to integers
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		fmt.Println("Invalid limit value:", err)
		limit = defaultLimit
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		fmt.Println("Invalid offset value:", err)
		page = defaultPage
	}

	offset := (page - 1) * limit

	var conf models.Conf
	result := db.DB.First(&conf)

	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// Get total content for count and include filter variables.
	totalContent, err := helpers.GetContent(defaultLimit, defaultPage, filterField, filterValue)
	if err != nil {
		fmt.Println("Error getting all content from db")
		return
	}
	totalPages := len(totalContent) / limit

	contents, err := helpers.GetContent(limit, offset, filterField, filterValue)
	if err != nil {
		fmt.Println("Error getting content from db")
		return
	}

	// Deal with pagination edges
	prevPage := page - 1
	if prevPage < 1 {
		prevPage = 1
	}

	nextPage := page + 1
	if nextPage > totalPages {
		nextPage = totalPages
	}

	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title":        "Admin",
	"data":         contents,
	"currentPage":  page,
	"nextPage":     nextPage,
	"previousPage": prevPage,
	})
}

// suggested search , needs improving and instigating on frontend
func SuggestedSearch(c *gin.Context) {
	searchString := c.Query("search")

	suggestions, err := fetchSuggestions(searchString)
	if err != nil {
		fmt.Println("Fetch Search Suggestions failed")
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

func fetchSuggestions(query string) ([]models.Content, error) {
	var items []models.Content

	result := db.DB.Where("title LIKE ?", "%"+query+"%").Find(&items)
	if result.Error != nil {
		return nil, result.Error
	}

	return items, nil
}


// / Add file upload end point which uploads to s3 and saves the file in ./upload so a cron job can run a file processing script later
func Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		return
	}

	// Generate a unique identifier for the file
	id := uuid.New().String()

	// Get the original filename
	fname := filepath.Base(file.Filename)

	// Modify the filename to UUID + original filename
	newFilename := id + "_" + fname

	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		log.Fatal(err)
	}

	tempFile := fmt.Sprintf("./uploads/%s", newFilename)
	if err := c.SaveUploadedFile(file, tempFile); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("Processing file")

	// Split this to extract one feature with one call to the py script.

	// New using goroutines
	contentChan := make(chan models.Content)
	errChan := make(chan error)

	go func() {
		content, err := helpers.ExtractAudioData(tempFile, fmt.Sprintf("%s", newFilename))
		if err != nil {
			errChan <- err
			return
		}
		contentChan <- content
	}()

	var content models.Content

	select {
		case content = <-contentChan:
			fmt.Println("\nNew file detatils:")
			helpers.PrettyPrintStruct(content)
		case err = <-errChan:
			fmt.Printf("Error %s while extracting audio data", err)
	}

	db.DB.Create(&content)

	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll("../archive", os.ModePerm); err != nil {
		log.Fatal(err)
	}

	newPath := fmt.Sprintf("../archive/%s", newFilename)

	err = os.Rename(tempFile, newPath)
	if err != nil {
		log.Fatal(err)
	}

	/// Add new content to db and move file from /uploads to ../../archive

	c.IndentedJSON(http.StatusCreated, gin.H{"message": "File uploaded", "data": content})
}

