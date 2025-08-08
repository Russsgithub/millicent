package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jmcvetta/randutil"

	//"github.com/aws/aws-sdk-go-v2/aws"
	//"github.com/aws/aws-sdk-go-v2/config"
	//"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"work/golang_api/models"

	"github.com/dhowden/tag"
)

// remove space from replay gain
// db content to only include needed columns (json keys) remove the others


var db *gorm.DB

func main() {
	// Set up the logger
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             200 * time.Millisecond, // Slow SQL threshold
			LogLevel:                  logger.Info,            // Log level
			IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,                   // Disable color
		},
	)

	var err error
	db, err = gorm.Open(sqlite.Open("users.db"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	if db == nil {
		log.Fatalf("db is nil")
	}

	fmt.Println("Database connection established")

	// Migrate one model at a time
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("Automigrate error for User: %v", err)
	}
	fmt.Println("User model migrated successfully")

	if err := db.AutoMigrate(&models.Content{}); err != nil {
		log.Fatalf("Automigrate error for Content: %v", err)
	}
	fmt.Println("Content model migrated successfully")

	if err := db.AutoMigrate(&models.Conf{}); err != nil {
		log.Fatalf("Automigrate error for Conf: %v", err)
	}
	fmt.Println("Conf model migrated successfully")

	fmt.Println("Database connected and migrated successfully")

	// Check if the User table has any records
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)

	// Check there is a default user
	if userCount == 0 {
		// No users found, create a default user
		defaultUser := models.User{
			Username: "control",
			Password: "m1ll1c3nt",
		}
		if err := db.Create(&defaultUser).Error; err != nil {
			log.Fatalf("failed to create default user: %v", err)
		}
		fmt.Println("Default user created")
	} else {
		fmt.Println("User table already has records")
	}

	var confCount int64
	db.Model(&models.Conf{}).Count(&confCount)
	if confCount > 1 {
		db.Where("id Not IN (?)", db.Model(&models.Conf{}).Select("id").Limit(1).Delete(&models.Conf{}))
		fmt.Println("Cleared extra entries in the conf table")
	} else if confCount == 0 {
		db.Create(&models.Conf{NoRepeatTime: "18"})
		fmt.Println("Added no repeat time default to conf table")
	}

	router := gin.Default()
	router.LoadHTMLGlob("templates/**/*")

	router.StaticFS("/static", gin.Dir("static", false))

	router.GET("/", home)

	router.GET("/admin", admin)

	router.GET("/adminUpload", BasicAuthMiddleware(adminUpload))

	router.GET("/content", getContents)
	router.GET("/content/:id", BasicAuthMiddleware(getContentByID))
	router.DELETE("/content/:id", BasicAuthMiddleware(deleteContentByID))
	router.POST("/content", BasicAuthMiddleware(postContent))
	router.PUT("/content/:id", BasicAuthMiddleware(patchContent))
	router.GET("/content/where", getContentWhere)

	router.GET("/playout", lastPlayed)

	router.GET("/search", suggestedSearch)

	router.GET("/users", BasicAuthMiddleware(getUsers))
	router.GET("/users/:id", BasicAuthMiddleware(getUserByID))
	router.POST("/users", BasicAuthMiddleware(postUsers))

	//	router.GET("/stats", stats )

	router.GET("/next", BasicAuthMiddleware(getNext))

	router.POST("/upload", BasicAuthMiddleware(upload))

	router.Run("0.0.0.0:8080")
}

// Slice containing the last 10 played music tracks
var recentlyPlayed []models.Content

func home(c *gin.Context) {
	c.HTML(http.StatusOK, "home.tmpl", gin.H{
		"title":   "millicent",
		"playout": recentlyPlayed,
	})
}

func adminUpload(c *gin.Context) {
	c.HTML(http.StatusOK, "upload.tmpl", gin.H{
		"title": "Upload files",
	})
}

func admin(c *gin.Context) {
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
	result := db.First(&conf)

	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// Get total content for count and include filter variables.
	totalContent, err := getContent(defaultLimit, defaultPage, filterField, filterValue)
	if err != nil {
		fmt.Println("Error getting all content from db")
		return
	}
	totalPages := len(totalContent) / limit

	contents, err := getContent(limit, offset, filterField, filterValue)
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

//func getLoudnessDistribution(data []Content) ([10]int, error) {
//	var distribution [10]int // Array to hold counts for loudness values 0-9
//
//	for _, item := range data {
//		if streamType, ok := item["stream_2"].(string); !ok || streamType != "music" {
//			continue
//		}
//		loudnessValue, ok := item["loudness_old"].(string)
//		if !ok {
//			continue
//		}
//
//		value, err := strconv.Atoi(loudnessValue)
//		if err != nil {
//			continue
//		}
//
//		if value < 0 || value > 9 {
//			return distribution, fmt.Errorf("loudness value out of range: %d", value)
//		}
//		distribution[value]++
//	}
//
//	return distribution, nil
//}
//
//func stats(c *gin.Context) {
//	contents, err := getContent()
//	if err != nil {
//		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//	}
//
//	loudnessDistribution, err := getLoudnessDistribution(contents)
//	if err != nil {
//		log.Printf("Error getting loudness distribution: %v", err)
//		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Failed to get loudness distribution"})
//		return
//	}
//
//	c.HTML(http.StatusOK, "stats.tmpl", gin.H{
//		"title": "Stats",
//		"data":  loudnessDistribution,
//	})
//
//}

// Audio analysis - fix

//func getEnergy(samples []float64) float64 {
//	start := time.Now()
//	// exrtact energy
//	var sum float64
//	for _, sample := range samples {
//		sum += sample * sample
//	}
//
//	averageEnergy := sum / float64(len(samples))
//
//	fmt.Printf("Energy: %.3f\n", averageEnergy)
//	fmt.Printf("Which took: %s seconds\n", time.Since(start).String())
//
//	return averageEnergy
//}
//
//// Find the next power of 2
//func nextPowerOf2(n int) int {
//	if n <= 0 {
//		return 1
//	}
//	return int(math.Pow(2, math.Ceil(math.Log2(float64(n)))))
//}
//
//// Pad array to the next power of 2
//func padToNextPowerOf2(samples []float64) []float64 {
//	paddedSize := nextPowerOf2(len(samples))
//	paddedSamples := make([]float64, paddedSize)
//	copy(paddedSamples, samples)
//	return paddedSamples
//}
//
//func getCentroid(samples []float64, sampleRate float64) float64 {
//	start := time.Now()
//
//    // Pad samples to the next power of 2
//    paddedSamples := padToNextPowerOf2(samples)
//
//	 // Convert samples to complex numbers
//	 complexSamples := gofft.Float64ToComplex128Array(paddedSamples)
//
//	// Perform FFT
//	err := gofft.FFT(complexSamples)
//	if err != nil {
//		fmt.Println("FFT error:", err)
//		return 0
//	}
//	// Debug: Print first few FFT results
//	//for i := 0; i < 10 && i < len(complexSamples); i++ {
//	//	fmt.Printf("FFT[%d]: %v\n", i, complexSamples[i])
//	//}
//    // Calculate magnitudes
//    magnitudes := make([]float64, len(complexSamples))
//    for i, v := range complexSamples {
//        magnitudes[i] = cmplx.Abs(v)
//    }
//
//    // Calculate the spectral centroid
//    var sumMagnitudes, weightedSum float64
//    for i, magnitude := range magnitudes {
//        frequency := float64(i) * float64(sampleRate) / float64(len(paddedSamples))
//        weightedSum += frequency * magnitude
//        sumMagnitudes += magnitude
//    }
//
//	if sumMagnitudes == 0 {
//		fmt.Println("Sum of magnitudes is zero, returning zero centroid")
//		return 0
//	}
//
//    spectralCentroid := weightedSum / sumMagnitudes
//
//	// Debug: Print magnitude and frequency
//	fmt.Printf("SumWeightedMagnitude = %f\n", weightedSum)
//	fmt.Printf("SumMagnitude = %f\n", sumMagnitudes)
//	fmt.Printf("Centroid: %.3f\n", spectralCentroid)
//	fmt.Printf("Which took: %s seconds\n", time.Since(start).String())
//
//	return spectralCentroid
//}
//
//func extractAudioEnergy(f *os.File) (string, string, error) {
//	decoder, err := mp3.NewDecoder(f)
//	if err != nil {
//		return "", "", err
//	}
//	buf := make([]byte, 1024)
//	var samples []float64
//	for {
//		n, err := decoder.Read(buf)
//		if err == io.EOF {
//			break
//		}
//		if err != nil {
//			return "", "", err
//		}
//		for i := 0; i < n; i += 2 {
//			sample := int16(buf[i]) | int16(buf[i+1])<<8
//			samples = append(samples, float64(sample)/32768.0)
//		}
//	}
//
//	sampleRate := float64(decoder.SampleRate())
//
//	averageEnergy := fmt.Sprintf("%.2f", getEnergy(samples))
//	spectralCentroid := fmt.Sprintf("%.2f", getCentroid(samples, sampleRate))
//
//	return averageEnergy, spectralCentroid, nil
//}

// suggested search , needs improving and instigating on frontend
func suggestedSearch(c *gin.Context) {
	searchString := c.Query("search")

	suggestions, err := fetchSuggestions(searchString)
	if err != nil {
		fmt.Println("Fetch Search Suggestions failed")
	}

	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

func fetchSuggestions(query string) ([]models.Content, error) {
	var items []models.Content

	result := db.Where("title LIKE ?", "%"+query+"%").Find(&items)
	if result.Error != nil {
		return nil, result.Error
	}

	return items, nil
}

func extractAudioData(fn string, newFilename string) (models.Content, error) {
	fmt.Println("Opening file")
	file, err := os.Open(fn)
	if err != nil {
		fmt.Printf("There was an error opening the file: %s", err)
		return models.Content{}, err
	}
	defer file.Close()

	// Extract metadata
	fmt.Println("Extracting metadata")
	metadata, err := tag.ReadFrom(file)
	if err != nil {
		fmt.Printf("There was an error opening the file: %s", err)
		return models.Content{}, err
	}

	title := metadata.Title()
	if title == "" {
		title = "Unknown"
	}

	artist := metadata.Artist()
	if artist == "" {
		artist = "Unknown"
	}

	var source_url string
	rawMetadata := metadata.Raw()
	for key, value := range rawMetadata {
		if key == "TXXX" {
			if comm, ok := value.(*tag.Comm); ok {
				if comm.Description == "URL" {
					source_url = comm.Text
				}
			}
		}
	}
	if source_url == "" {
		fmt.Println("TXXX tag not found")

		// add source url as a duckduckgo search
		a := strings.Replace(artist, " ", "+", -1)
		t := strings.Replace(title, " ", "+", -1)
		source_url = fmt.Sprintf("https://duckduckgo.com/?q=%s+%s&mute=1", a, t)
	}

	// Sanitize the url
	sanitized_source_url, err := url.Parse(source_url)
	if err != nil {
		fmt.Printf("There was an error sanitizing the url for source_url: %s", err)
	}

	source_url = sanitized_source_url.String()

	//var source_url string
	//
	//if metadata.SourceUrl() {
	//	source_url = metadata.SourceUrl()
	//} else {
	//	source_url = "https://duckduckgo.com/?q=" + artist + title
	//}

	// Run python script to extract audio data. (./tools/analyze.py)
	pythonPath := "./venv/bin/python" // Use "venv/bin/python" for Unix-based systems

	// Path to the Python script
	scriptPath := "./tools/analyze.py"

	fmt.Printf("Executing py script for %s %s\n", title, artist)

	cmd := exec.Command(pythonPath, scriptPath, fn)

	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error:", err)
		return models.Content{}, err
	}

	fmt.Printf("Outptut: %s", output)

	var content models.Content
	err = json.Unmarshal(output, &content)
	if err != nil {
		return models.Content{}, err
	}

	delimiter := "/"
	url_parts := strings.Split(fn, delimiter)
	url := url_parts[len(url_parts)-1]

	content.Title = title
	content.Artist = artist
	content.SourceUrl = source_url
	content.Url = url
	content.Processed = "1"
	content.Currated = "1"
	content.PlayCount = "0"

	return content, nil
}

// Pretty print struct data
func prettyPrintStruct(s interface{}) {
	v := reflect.ValueOf(s)
	typeOfS := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fmt.Printf("%s: %v\n", typeOfS.Field(i).Name, v.Field(i).Interface())
	}
}

// / Add file upload end point which uploads to s3 and saves the file in ./upload so a cron job can run a file processing script later
func upload(c *gin.Context) {
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
		content, err := extractAudioData(tempFile, fmt.Sprintf("%s", newFilename))
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
		prettyPrintStruct(content)
	case err = <-errChan:
		fmt.Printf("Error %s while extracting audio data", err)
	}

	db.Create(&content)

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

/// Add delete file on delete content by id

// getContentByID returns an entry from an id as json
func getContentByID(c *gin.Context) {
	id := c.Param("id")

	var content models.Content
	err := db.First(&content, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "content not found"})
			return
		}
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.IndentedJSON(http.StatusOK, content)
}

func getItemfromDb(id string) (models.Content, error) {
	var content models.Content
	err := db.First(&content, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Content{}, err
		}
		return models.Content{}, err
	}

	return content, nil
}

// deleteContentByID deletes an entry in the content db from its id
// add file delete to this
func deleteContentByID(c *gin.Context) {
	id := c.Param("id")

	item, err := getItemfromDb(id)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := db.Delete(&models.Content{}, id)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error})
		return
	}

	if result.RowsAffected == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "content not found"})
		return
	}

	filePath := fmt.Sprintf("../archive/%s", item.Url)

	err = os.Remove(filePath)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "content deleted successfully"})
}

func getContents(c *gin.Context) {
	filterField := "stream2"
	filterValue := c.Query("stream")
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	const defaultLimit = -1
	const defaultOffset = 0

	// Convert strings to integers
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		fmt.Println("Invalid limit value:", err)
		limit = defaultLimit
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		fmt.Println("Invalid offset value:", err)
		offset = defaultOffset
	}

	contents, err := getContent(limit, offset, filterField, filterValue)
	if err != nil {
		log.Fatalf("Error getting content: %v", err)
	}

	c.JSON(http.StatusOK, contents)
}

// getContent gets the content from the content database as json
// TODO return all columns not just 4
func getContent(limit int, offset int, filterField string, filterValue string) ([]models.Content, error) {
	var contents []models.Content
	query := db.Limit(limit).Offset(offset) // Use := for variable initialization

	// Apply filtering only if filterField and filterValue are provided
	if filterField != "" && filterValue != "" {
		query = query.Where(fmt.Sprintf("%s = ?", filterField), filterValue)
	}

	result := query.Find(&contents) // Execute the query
	if result.Error != nil {
		return nil, result.Error
	}

	return contents, nil
}

// / get content with where clause call as get ...../content?column_name=value&column_name=value%201 where %20 is the space character
func getContentWhere(c *gin.Context) {
	query := map[string]interface{}{}

	for key, value := range c.Request.URL.Query() {
		// Validate allowed query parameters
		if !isValidQueryParam(key) { // Define a function to check allowed columns
			c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "invalid query parameter"})
			return
		}
		query[key] = value[0]
	}

	var contents []models.Content

	// Use GORM's Where method with the query map
	result := db.Where(query).Find(&contents)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK,
		contents)
}

func isValidQueryParam(param string) bool {
	// Define allowed query parameters here (e.g., "title", "artist")
	allowedParams := map[string]struct{}{
		"title":    {},
		"artist":   {},
		"stream":   {},
		"style":    {},
		"stream_2": {},
		// ... other allowed parameters
	}
	_, ok := allowedParams[param]
	return ok
}

// get_next returns a random entry that meets the condition of the where clause.
// get content with where clause call as get ...../content?column_name=value&column_name=value%201 where %20 is the space character
func getNext(c *gin.Context) {
	// Define query parameters with defaults
	params := map[string]interface{}{
		"stream_2":            c.Request.URL.Query().Get("stream"),
		"norm_centroid":       c.Request.URL.Query().Get("norm_centroid"), // Assuming loudness is a range
		"norm_energy":         c.Request.URL.Query().Get("norm_energy"),
		"spec_bandwidth_norm": c.Request.URL.Query().Get("bandwidth_norm"),
		"mix_type":            c.Request.URL.Query().Get("mix_type"),
		"style":               c.Request.URL.Query().Get("genre"),
		"duration_lt":         c.Request.URL.Query().Get("duration"), // Assuming duration is less than
		"currated":            "1",
	}

	// consider reducing the length of fx fiels to below 10 - 15 secs

	// Only get files longer than 30 seconds on vocal and field recordings streams
	if params["stream_2"] == "vocal" || params["stream_2"] == "noise" {
		params["duration_gt"] = "30"
	}

	// Only get files shorter than 30 seconds files on the vocal_fx stream
	if params["stream_2"] == "vocal_fx" {
		params["stream_2"] = "vocal"
		params["duration_lt"] = "20"
		params["duration_gt"] = ""
	}

	// Build GORM query with conditions
	// Assuming a configured GORM instance
	query := db.Model(&models.Content{})

	for key, value := range params {
		if valueStr, ok := value.(string); ok && valueStr != "" {
			switch key {
			case "norm_centroid", "norm_energy":
				val, err := strconv.Atoi(valueStr)
				if err != nil {
					log.Fatal("norm_x str conversion error")
				}

				if val >= 2 && val <= 8 {
					query = query.Where(key+" BETWEEN ? AND ?", val-1, val+1)
				} else if val < 1 {
					query = query.Where(key+" BETWEEN ? AND ?", 0, 3)
				} else if val > 8 {
					query = query.Where(key+" BETWEEN ? AND ?", 7, 9)
				}
// Include current centroid val.
				// query = query.Where(key+" != ?", val)

			case "duration_lt":
				val, err := strconv.Atoi(valueStr)
				if err != nil {
					c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err})
				} else {
					query = query.Where("CAST(duration AS REAL) < ?", val)
				}
			case "duration_gt":
				val, err := strconv.Atoi(valueStr)
				if err != nil {
					c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err})
				} else {
					query = query.Where("CAST(duration AS REAL) > ?", val)
				}
			default:
				query = query.Where(key+" = ?", value)
			}
		}
	}

	// Additional condition for "vocal_fx" to include "noise"
	if c.Request.URL.Query().Get("stream") == "vocal_fx" {
		query = query.Or("stream_2 = ?", "noise")
	}

	// Get no repeat time from conf table
	var conf models.Conf
	result := db.First(&conf)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	noRepeatTime, err := strconv.Atoi(conf.NoRepeatTime)
	if err != nil {
		fmt.Println("Error convertng no repeat string")
	}
	noRepeatTime = -1 * noRepeatTime

	// Filter for content not played recently
	pastTime := time.Now().Add(time.Duration(noRepeatTime) * time.Hour)
	query = query.Where("last_played < ?", pastTime.In(time.Local).Format("2006-01-02T15:04:05.000Z"))

	// Ensure proper grouping of conditions - WHY ?
	//query = query.Where(
	//	db.Where("stream_2 = ? AND CAST(duration AS REAL) < ? AND currated = ?", params["stream_2"], //params["duration_lt"], "1").
		//Or("stream_2 = ? AND last_played < ?", params["stream_2"], //pastTime.Format("2006-01-02T15:04:05.000Z")),
	//)


	// Apply weighted randomization logic (using a separate library like "github.com/cespare/weightedrand")
	// fix this
	var contents []models.Content
	result = query.Order("RANDOM()").Find(&contents) // Assuming a way to populate choices with weighted content
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	var maxPlaycountInDb int
	db.Model(&models.Content{}).Select("MAX(play_count)").Scan(&maxPlaycountInDb)

	rand.Seed(time.Now().UnixNano())

	var choices []randutil.Choice
	for _, content := range contents {
		playCount, err := strconv.Atoi(content.PlayCount)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "Invalid playcount value"})
			return
		}

		// Avoid log(0), favor lower playCount but reduce patterning
		adjusted := math.Log(float64(playCount+2)) + rand.Float64()*0.5 // Add a bit of noise
		weight := int(1000 / adjusted)

		if weight < 1 {
			weight = 1
		}

		choices = append(choices, randutil.Choice{Item: content, Weight: weight})
	}

	// Debug print to check choices and weights
	//for i, choice := range choices {
	//	fmt.Printf("Choice %d: %+v\n", i, choice)
	//}

	msg := fmt.Sprintf("There are %v files to choose from in this call\n", len(choices))
	fmt.Println(msg)

	chooser, err := randutil.WeightedChoice(choices)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"weighted choice error": err.Error()})
		return
	}

	choice := chooser.Item.(models.Content)

	if choice.LastPlayed.After(pastTime) {
		fmt.Printf("\033[31mFile last picked on: %s\033[0m\n\n", choice.LastPlayed.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("File last picked on: %s\n\n", choice.LastPlayed.Format("2006-01-02 15:04:05"))
	}

	// If call is to 'vocal_fx' return stream is 'vocal_fx' - revert changes for db call made above.
	// checked on existance of duration_lt in params.
	if params["duration_lt"] == "30" {
		choice.Stream_2 = "vocal_fx"
	}

	// Build the formatted result string
	base_url := "home/ubuntu/archive"
	formattedResult := fmt.Sprintf("annotate:id=\"%d\",title=\"%s\",artist=\"%s\",duration=\"%s\",mix_type=\"%s\",source_url=\"%s\",liq_on_offset=\"%s\",play_count=\"%s\",style=\"%s\",stream=\"%s\",norm_energy=\"%s\",norm_centroid=\"%s\",replaygain_track_gain=\"%s\":file:///%s/%s",
		choice.ID, choice.Title, choice.Artist, choice.Duration, choice.MixType, choice.SourceUrl, choice.Offset, choice.PlayCount, choice.Style, choice.Stream_2, choice.NormEnergy, choice.NormCentroid, choice.ReplaygainTrackGain, base_url, choice.Url)

	c.String(http.StatusOK, formattedResult)
}

// / postContent take JSON in the body of a request and adds it to the content table in the db
// / TODO remove duplicate keys in json inside the go code raher than out sourcing to the python migrate script
// / TODO Use uuid to create id don't use increment
func postContent(c *gin.Context) {
	var newContent models.Content // Use the Content struct

	if err := c.ShouldBindJSON(&newContent); err != nil {
		fmt.Println("Json bind error")
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the content struct based on defined fields (optional)
	if err := newContent.Validate(); err != nil { // Define Validate method in Content struct (if needed)
		fmt.Println("Json validation error")
		fmt.Println(err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a new content record using GORM's Create method
	result := db.Create(&newContent)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.IndentedJSON(http.StatusCreated,
		newContent)
}

// Function to check if a column exists in the table // not working
func columnExistsInTable(db *sql.DB, tableName string, columnName string) bool {
	var count int
	query := `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?`
	err := db.QueryRow(query, tableName, columnName).Scan(&count)
	if err != nil {
		log.Println(err)
		return false
	}
	return count > 0
}

// Function to add a column to the table
func addColumnToTable(db *sql.DB, tableName string, columnName string, columnType string) {
	log.Println("Adding columns.")
	alterQuery := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, columnType)
	_, err := db.Exec(alterQuery)
	if err != nil {
		log.Fatal(err)
	}
}

func patchContent(c *gin.Context) {
	var updatedContent map[string]interface{}
	if err := c.BindJSON(&updatedContent); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		fmt.Println(err.Error())
		return
	}

	// Remove keys with null values
	for k, v := range updatedContent {
		if v == nil {
			delete(updatedContent, k)
		}
	}

	id := c.Param("id")

	// Convert last_played to time.Time if it exists in the updatedContent
	if lastPlayed, exists := updatedContent["last_played"]; exists {
		if lastPlayedStr, ok := lastPlayed.(string); ok {
			parsedTime, err := time.Parse(time.RFC3339, lastPlayedStr)
			if err != nil {
				c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "invalid last_played format"})
				return
			}
			updatedContent["last_played"] = parsedTime
		}
	}

	// Use GORM's Model and Updates methods
	result := db.Model(&models.Content{}).Where("id = ?", id).Updates(updatedContent)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "content not found"})
		return
	}
	// Retrieve item from the database
	var updatedEntry models.Content
	if err := db.First(&updatedEntry, id).Error; err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve the updated entry"})
		return
	}

	// Prepend music and vocal tracks to recentlyPlayed slice by adding old to the end of new slice
	if !contains(recentlyPlayed, updatedEntry) && (updatedEntry.Stream_2 == "music" || updatedEntry.Stream_2 == "vocal") {
		recentlyPlayed = append([]models.Content{updatedEntry}, recentlyPlayed...)
	}

	// Keep list at 10 entries max
	if len(recentlyPlayed) > 10 {
		recentlyPlayed = recentlyPlayed[:len(recentlyPlayed)-1]
	}

	c.IndentedJSON(http.StatusOK, updatedContent)
}

func lastPlayed(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, recentlyPlayed)
}

// Helper function to check if the entry already exists in the list
func contains(entries []models.Content, entry models.Content) bool {
	for _, e := range entries {
		if e.ID == entry.ID { // Assuming Content has an ID field
			return true
		}
	}
	return false
}

// getUsers respond with list of users in as json
func getUsers(c *gin.Context) {
	var users []models.User // Define a slice to hold User structs

	result := db.Find(&users)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, users)
}

// postUsers add user from json recieved in request body
func postUsers(c *gin.Context) {
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
	result := db.Create(&newUser)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.IndentedJSON(http.StatusCreated, newUser)
}

// Get user by id
func getUserByID(c *gin.Context) {
	id := c.Param("id")

	var user models.User // Define a User struct

	err := db.First(&user, id).Error
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

	err := db.Where("username = ? AND password = ?", username, password).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // Check for record not found specifically
			return false // User not found (valid login attempt but no match)
		}
		fmt.Println("Failed Login !")
		return false // Other errors
	}

	return true // User found and credentials match
}
