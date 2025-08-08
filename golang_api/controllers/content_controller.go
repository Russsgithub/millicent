package controllers

import (
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/jmcvetta/randutil"

	"gorm.io/gorm"

	"work/golang_api/db"
	"work/golang_api/globals"
	"work/golang_api/models"
	"work/golang_api/helpers"
)

// getContentByID returns an entry from an id as json
func GetContentByID(c *gin.Context) {
	id := c.Param("id")

	var content models.Content
	err := db.DB.First(&content, id).Error

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

// deleteContentByID deletes an entry in the content db from its id
// add file delete to this
func DeleteContentByID(c *gin.Context) {
	id := c.Param("id")

	item, err := helpers.GetItemfromDb(id)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := db.DB.Delete(&models.Content{}, id)
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

func GetContents(c *gin.Context) {
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

	contents, err := helpers.GetContent(limit, offset, filterField, filterValue)
	if err != nil {
		log.Fatalf("Error getting content: %v", err)
	}

	c.JSON(http.StatusOK, contents)
}

// / get content with where clause call as get ...../content?column_name=value&column_name=value%201 where %20 is the space character
func GetContentWhere(c *gin.Context) {
	query := map[string]interface{}{}

	for key, value := range c.Request.URL.Query() {
		// Validate allowed query parameters
		if !helpers.IsValidQueryParam(key) { // Define a function to check allowed columns
			c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "invalid query parameter"})
			return
		}
		query[key] = value[0]
	}

	var contents []models.Content

	// Use GORM's Where method with the query map
	result := db.DB.Where(query).Find(&contents)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK,
		       contents)
}


// get_next returns a random entry that meets the condition of the where clause.
// get content with where clause call as get ...../content?column_name=value&column_name=value%201 where %20 is the space character
func GetNext(c *gin.Context) {
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
	query := db.DB.Model(&models.Content{})

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
	result := db.DB.First(&conf)
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
	db.DB.Model(&models.Content{}).Select("MAX(play_count)").Scan(&maxPlaycountInDb)

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
	// Editted for testing purposes
	base_url := "/run/media/russ/FE7B-F6D2/work/millicent_archive"
	//base_url := "home/ubuntu/archive"
	formattedResult := fmt.Sprintf("annotate:id=\"%d\",title=\"%s\",artist=\"%s\",duration=\"%s\",mix_type=\"%s\",source_url=\"%s\",liq_on_offset=\"%s\",play_count=\"%s\",style=\"%s\",stream=\"%s\",norm_energy=\"%s\",norm_centroid=\"%s\",replaygain_track_gain=\"%s\":file:///%s/%s",
		choice.ID, choice.Title, choice.Artist, choice.Duration, choice.MixType, choice.SourceUrl, choice.Offset, choice.PlayCount, choice.Style, choice.Stream_2, choice.NormEnergy, choice.NormCentroid, choice.ReplaygainTrackGain, base_url, choice.Url)

		c.String(http.StatusOK, formattedResult)
}

// / postContent take JSON in the body of a request and adds it to the content table in the db
// / TODO remove duplicate keys in json inside the go code raher than out sourcing to the python migrate script
// / TODO Use uuid to create id don't use increment
func PostContent(c *gin.Context) {
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
	result := db.DB.Create(&newContent)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.IndentedJSON(http.StatusCreated,
		       newContent)
}


func PatchContent(c *gin.Context) {
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
	result := db.DB.Model(&models.Content{}).Where("id = ?", id).Updates(updatedContent)
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
	if err := db.DB.First(&updatedEntry, id).Error; err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve the updated entry"})
		return
	}

	// Prepend music and vocal tracks to recentlyPlayed slice by adding old to the end of new slice
	if !helpers.Contains(globals.RecentlyPlayed, updatedEntry) && (updatedEntry.Stream_2 == "music" || updatedEntry.Stream_2 == "vocal") {
		globals.RecentlyPlayed = append([]models.Content{updatedEntry}, globals.RecentlyPlayed...)
	}

	// Keep list at 10 entries max
	if len(globals.RecentlyPlayed) > 10 {
		globals.RecentlyPlayed = globals.RecentlyPlayed[:len(globals.RecentlyPlayed)-1]
	}

	c.IndentedJSON(http.StatusOK, updatedContent)
}


func LastPlayed(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, globals.RecentlyPlayed)
}
