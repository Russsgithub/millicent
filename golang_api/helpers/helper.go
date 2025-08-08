package helpers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"strings"

	"github.com/dhowden/tag"

	"gorm.io/gorm"

	"work/golang_api/db"
	"work/golang_api/models"
)

func ExtractAudioData(fn string, newFilename string) (models.Content, error) {
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
func PrettyPrintStruct(s interface{}) {
	v := reflect.ValueOf(s)
	typeOfS := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fmt.Printf("%s: %v\n", typeOfS.Field(i).Name, v.Field(i).Interface())
	}
}

/// Add delete file on delete content by id



func GetItemfromDb(id string) (models.Content, error) {
	var content models.Content
	err := db.DB.First(&content, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Content{}, err
		}
		return models.Content{}, err
	}

	return content, nil
}


func IsValidQueryParam(param string) bool {
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

// Function to check if a column exists in the table // not working
func ColumnExistsInTable(db *sql.DB, tableName string, columnName string) bool {
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
func AddColumnToTable(db *sql.DB, tableName string, columnName string, columnType string) {
	log.Println("Adding columns.")
	alterQuery := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, columnType)
	_, err := db.Exec(alterQuery)
	if err != nil {
		log.Fatal(err)
	}
}

// Helper function to check if the entry already exists in the list
func Contains(entries []models.Content, entry models.Content) bool {
	for _, e := range entries {
		if e.ID == entry.ID { // Assuming Content has an ID field
			return true
		}
	}
	return false
}

// getContent gets the content from the content database as json
// TODO return all columns not just 4
func GetContent(limit int, offset int, filterField string, filterValue string) ([]models.Content, error) {
	var contents []models.Content
	query := db.DB.Limit(limit).Offset(offset) // Use := for variable initialization

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
