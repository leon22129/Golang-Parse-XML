package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	DB_TABLE_NAME             = "doc"         // Table name for SQLite
	DB_ID_FIELD_NAME          = "id"          // Field name for id in SQLite table
	DB_TITLE_FIELD_NAME       = "title"       // Field name for title in SQLite table
	DB_DESCRIPTION_FIELD_NAME = "description" // Field name for description in SQLite table
	DB_AUTHOR_FIELD_NAME      = "author"      // Field name for author in SQLite table
	DB_CREATEDAT_FIELD_NAME   = "created_at"  // Field name for created_at in SQLite table
	DB_XMLDATA_FIELD_NAME     = "xml_data"    // Field name for xml_data in SQLite table

	XML_FILES_PATH        = "./xml_files"    // XML file path to get all xml files in the storage
	XML_TITLE_PREFIX      = "<title>"        // XML tag prefix for title
	XML_DESCIPTION_PREFIX = "<description>"  // XML tag prefix for description
	XML_AUTHOR_PREFIX     = "<author>"       // XML tag prefix for author
	XML_CREATEDAT_PREFIX  = "<creationDate>" // XML tag prefix for creationDate

	SPLIT_XMLDATA_STR = "µ∜⨚Ť¿" // String to split and join XML data
)

// XML Document struct to hold parsed data
type XMLDoc struct {
	ID          string
	Title       string
	Description string
	Author      string
	CreatedAt   string
	XMLData     []string
}

// parseXML parses XML-formed string to array
// Array's order is the same with visiting tree by depth-order
func parseXML(data string) ([]string, error) {
	// XMLTag represents a parsed XML tag with its index
	type XMLTag struct {
		Tag   string // Tag represents the XML tag string ("<tag>" or "</tag>")
		Index int    // Index is the starting index of the tag in the original XML data string
	}

	var result []string   // The result which returned in this function
	var xmlTags []XMLTag  // Slice to hold parsed XML tags
	var currentTag XMLTag // current tag for cache
	inTag := false        // Flag to track if currently parsing inside a tag

	// Parse through the XML string character by character
	for i, char := range data {
		if char == '<' { // If it's a new start of a tag
			inTag = true
			if currentTag.Tag != "" {
				return nil, errors.New("tag pairing error") // Return error if tags are not properly paired
			}
			currentTag.Tag = "<"
			currentTag.Index = i
		} else if char == '>' { // If it's the end of a tag
			inTag = false
			currentTag.Tag += ">"
			xmlTags = append(xmlTags, currentTag)
			currentTag.Tag = ""
		} else { // If not, then add the letter to currentTag
			if inTag {
				currentTag.Tag += string(char)
			}
		}
	}

	var stack []XMLTag // Stack to manage nested tags
	index := 0         // Depth index counter

	// XMLData represents extracted XML data along with its depth
	type XMLData struct {
		Data  string // Data is the extracted XML data including its tags
		Depth int    // Depth represents the nested level of the XML data
	}
	var xmlDataArr []XMLData // Slice to hold final extracted XML data

	// Process each parsed XML tag
	for _, tag := range xmlTags {
		if strings.HasPrefix(tag.Tag, "</") { // If it's a closing tag
			if len(stack) == 0 {
				return nil, errors.New("no opening tag error: no opening tag") // Return error if no matching opening tag found
			}
			lastTag := stack[len(stack)-1] // Get the last opened tag from the stack

			if strings.Split(lastTag.Tag[1:len(lastTag.Tag)-1], " ")[0] == strings.Split(tag.Tag[2:len(tag.Tag)-1], " ")[0] { // Check if the closing tag matches the last opened tag ***split is needed if tag is like this: "<section id="1">"***
				data := XMLData{Data: data[lastTag.Index:tag.Index] + tag.Tag, Depth: index}
				xmlDataArr = append(xmlDataArr, data) // Add to xmlDataArr
				stack = stack[:len(stack)-1]
				index--
			} else {
				return nil, errors.New("unmatched closing tag error: " + lastTag.Tag + " " + tag.Tag) // Return error if closing tag doesn't match
			}
		} else {
			if strings.HasSuffix(tag.Tag, "/>") { // If self-closing tag
				data := XMLData{Data: tag.Tag, Depth: index}
				xmlDataArr = append(xmlDataArr, data)
			} else if !(strings.HasPrefix(tag.Tag, "<!--")) { // Check if it's a comment
				stack = append(stack, tag)
				index++
			}
		}
	}

	// Sort xmlDataArr by depth
	sort.Slice(xmlDataArr, func(i, j int) bool {
		return xmlDataArr[i].Depth < xmlDataArr[j].Depth
	})

	for _, data := range xmlDataArr {
		// Clean up unnecessary characters from data
		str := strings.ReplaceAll(data.Data, "\t", "")
		str = strings.ReplaceAll(str, "    ", "")
		str = strings.ReplaceAll(str, "\n", "")
		str = strings.ReplaceAll(str, "\r", "")

		result = append(result, str)
	}

	return result, nil
}

// Function to parse XML-formed string to XMLDoc struct
func parseDocument(data string) (*XMLDoc, error) {
	if data == "" {
		return nil, errors.New("no data for parsing")
	}

	// Get xmlDoc-formed data by calling parseXML
	xmlDataArr, err := parseXML(data)
	if err != nil {
		return nil, err
	}

	doc := XMLDoc{}

	for _, str := range xmlDataArr {
		// Check and parse specific elements if they match known prefixes

		if strings.HasPrefix(str, XML_TITLE_PREFIX) && doc.Title == "" {
			doc.Title = str[len(XML_TITLE_PREFIX) : len(str)-len(XML_TITLE_PREFIX)-1]
		}
		if strings.HasPrefix(str, XML_DESCIPTION_PREFIX) && doc.Description == "" {
			doc.Description = str[len(XML_DESCIPTION_PREFIX) : len(str)-len(XML_DESCIPTION_PREFIX)-1]
		}
		if strings.HasPrefix(str, XML_AUTHOR_PREFIX) && doc.Author == "" {
			doc.Author = str[len(XML_AUTHOR_PREFIX) : len(str)-len(XML_AUTHOR_PREFIX)-1]
		}
		if strings.HasPrefix(str, XML_CREATEDAT_PREFIX) && doc.CreatedAt == "" {
			doc.CreatedAt = str[len(XML_CREATEDAT_PREFIX) : len(str)-len(XML_CREATEDAT_PREFIX)-1]
		}
	}

	doc.XMLData = xmlDataArr

	return &doc, nil
}

// loadXMLFiles loads XML files from the specified directory, parses them, and inserts into the database
func loadXMLFiles(db *sql.DB, directory string) error {
	funcName := "loadXMLFiles"

	// Read all files in the directory
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return err
	}

	// Iterate over files and filter XML files
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".xml") {
			// Read XML file content
			filePath := filepath.Join(directory, file.Name())
			content, err := ioutil.ReadFile(filePath)
			if err != nil {
				log.Fatalf(funcName, "Error reading file %s: %v", filePath, err)
				continue
			}

			// Parse content to XMLDoc struct
			doc, err := parseDocument(string(content))
			if err != nil {
				log.Fatalf(funcName, err)
				continue
			}

			// Add doc to SQLite
			err = insertDocument(db, *doc)
			if err != nil {
				log.Fatalf(funcName, err)
			}
		}
	}

	return nil
}

// initDB initializes SQLite database and creates the necessary table if not exists
func initDB(db *sql.DB) {
	funcName := "initDB"

	// Create documents table if not exists
	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		"%s" INTEGER PRIMARY KEY,
		"%s" TEXT,
		"%s" TEXT,
		"%s" TEXT,
		"%s" TEXT,
		"%s" TEXT
	);
`, DB_TABLE_NAME, DB_ID_FIELD_NAME, DB_TITLE_FIELD_NAME, DB_DESCRIPTION_FIELD_NAME, DB_AUTHOR_FIELD_NAME, DB_CREATEDAT_FIELD_NAME, DB_XMLDATA_FIELD_NAME)

	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf(funcName, "Failed to create table: %v", err)
	}

	// Add document from files
	// err = loadXMLFiles(db, XML_FILES_PATH)
	// if err != nil {
	// 	log.Fatalf(funcName, "Failed to load XML files: %v", err)
	// }
}

// insertDocument inserts a document into the database
func insertDocument(db *sql.DB, doc XMLDoc) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s, %s, %s)
		VALUES (?, ?, ?, ?, ?)
	`, DB_TABLE_NAME, DB_TITLE_FIELD_NAME, DB_DESCRIPTION_FIELD_NAME, DB_AUTHOR_FIELD_NAME, DB_CREATEDAT_FIELD_NAME, DB_XMLDATA_FIELD_NAME)
	_, err := db.Exec(query, doc.Title, doc.Description, doc.Author, doc.CreatedAt, strings.Join(doc.XMLData, SPLIT_XMLDATA_STR))
	return err
}

func deleteDocumentByID(db *sql.DB, id string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s WHERE %s=?
	`, DB_TABLE_NAME, DB_ID_FIELD_NAME)
	_, err := db.Exec(query, id)
	return err
}

// getDocumentByID retrieves a document from the database by its ID
func getDocumentByID(db *sql.DB, id string) (*XMLDoc, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s, %s FROM %s WHERE %s=?
	`, DB_TITLE_FIELD_NAME, DB_DESCRIPTION_FIELD_NAME, DB_AUTHOR_FIELD_NAME, DB_CREATEDAT_FIELD_NAME, DB_XMLDATA_FIELD_NAME, DB_TABLE_NAME, DB_ID_FIELD_NAME)
	var title, description, author, createdAt, xmlDataStr string
	err := db.QueryRow(query, id).Scan(&title, &description, &author, &createdAt, &xmlDataStr)
	if err != nil {
		return nil, err
	}

	xmlData := strings.Split(xmlDataStr, SPLIT_XMLDATA_STR)
	return &XMLDoc{
		ID:          id,
		Title:       title,
		Description: description,
		Author:      author,
		CreatedAt:   createdAt,
		XMLData:     xmlData,
	}, nil
}

func handleRequest(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/document":
		handleDocumentRequest(db, w, r)
	case "/add":
		handleAddRequest(db, w, r)
	case "/del":
		handleDeleteRequest(db, w, r)
	default:
		http.Error(w, "404 Not Found", http.StatusNotFound)
	}
}

func handleDocumentRequest(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	doc, err := getDocumentByID(db, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch document with ID %s: %v", id, err), http.StatusInternalServerError)
		return
	}

	// Convert to JSON and send response
	response, err := json.Marshal(doc)
	if err != nil {
		http.Error(w, "Failed to marshal JSON response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func handleAddRequest(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	// Parse request body
	xmlData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse XML data into XMLDoc struct
	doc, err := parseDocument(string(xmlData))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse document: %v", err), http.StatusInternalServerError)
		return
	}

	// Insert document into database
	err = insertDocument(db, *doc)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to insert document into database: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func handleDeleteRequest(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	err := deleteDocumentByID(db, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete document with ID %s: %v", id, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	docDB, err := sql.Open("sqlite3", "./documents.db")
	if err != nil {
		log.Fatal("Failed to open database", err)
	}
	defer docDB.Close()

	initDB(docDB)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRequest(docDB, w, r)
	})

	log.Println("Server listening on :3456")
	log.Fatal(http.ListenAndServe(":3456", nil))
}
