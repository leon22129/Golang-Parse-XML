package main

import (
	"database/sql"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// Utility function to initialize an in-memory SQLite database for testing
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	initDB(db)

	// Cleanup function to close the database connection
	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func compareDoc(doc1 XMLDoc, doc2 XMLDoc) bool {
	if doc1.Title != doc2.Title {
		return false
	} else if doc1.Description != doc2.Description {
		return false
	} else if doc1.Author != doc2.Author {
		return false
	} else if doc1.CreatedAt != doc2.CreatedAt {
		return false
	} else if len(doc1.XMLData) != len(doc2.XMLData) {
		return false
	}
	for i, _ := range doc1.XMLData {
		if doc1.XMLData[i] != doc2.XMLData[i] {
			return false
		}
	}
	return true
}

// Test the XML parsing function with malformed XML
func TestParseXML(t *testing.T) {
	tests := []struct {
		desc             string
		msg              string
		expectedResponse []string
		err              error
	}{
		{
			desc: "valid parsing",
			msg: `<document>
			<title>Test Title</title>
			<description>Test Description</description>
			<author>Test Author</author>
			<creationDate>2024-07-09</creationDate>
		</document>`,
			expectedResponse: []string{
				"<document><title>Test Title</title><description>Test Description</description><author>Test Author</author><creationDate>2024-07-09</creationDate></document>",
				"<title>Test Title</title>",
				"<description>Test Description</description>",
				"<author>Test Author</author>",
				"<creationDate>2024-07-09</creationDate>",
			},
			err: nil,
		}, {
			desc: "invalid pairing",
			msg:  `<document><title</description></document>`,
			err:  errors.New("tag pairing error"),
		}, {
			desc: "no opening tag",
			msg: `</document>
			<title>Test Title</title>
			<description>Test Description</description>
			<author>Test Author</author>
			<creationDate>2024-07-09</creationDate>
		</document>`,
			err: errors.New("no opening tag error: no opening tag"),
		}, {
			desc: "unmatched closing tag error",
			msg: `<document>
			Test Title</title>
			<description>Test Description</description>
			<author>Test Author</author>
			<creationDate>2024-07-09</creationDate>
		</document>`,
			err: errors.New("unmatched closing tag error: <document> </title>"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			response, err := parseXML(tt.msg)
			if tt.err != nil {
				require.EqualValues(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.EqualValues(t, tt.expectedResponse, response)
			}
		})
	}
}

// Test the document parsing function with valid data
func TestParseDocument(t *testing.T) {
	tests := []struct {
		desc             string
		msg              string
		expectedResponse XMLDoc
		err              error
	}{
		{
			desc: "valid parsing",
			msg: `<document>
			<title>Test Title</title>
			<description>Test Description</description>
			<author>Test Author</author>
			<creationDate>2024-07-09</creationDate>
		</document>`,
			expectedResponse: XMLDoc{
				Title:       "Test Title",
				Description: "Test Description",
				Author:      "Test Author",
				CreatedAt:   "2024-07-09",
				XMLData: []string{
					"<document><title>Test Title</title><description>Test Description</description><author>Test Author</author><creationDate>2024-07-09</creationDate></document>",
					"<title>Test Title</title>",
					"<description>Test Description</description>",
					"<author>Test Author</author>",
					"<creationDate>2024-07-09</creationDate>",
				},
			},
			err: nil,
		}, {
			desc: "empty data",
			msg:  ``,
			err:  errors.New("no data for parsing"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			response, err := parseDocument(tt.msg)
			if tt.err != nil {
				require.EqualValues(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.EqualValues(t, &tt.expectedResponse, response)
			}
		})
	}
}

// Test inserting a document to the database
func TestInsertDocument(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	doc := XMLDoc{
		Title:       "Test Title",
		Description: "Test Description",
		Author:      "Test Author",
		CreatedAt:   "2024-07-09",
		XMLData: []string{
			"<title>Test Title</title>",
			"<description>Test Description</description>",
			"<author>Test Author</author>",
			"<creationDate>2024-07-09</creationDate>",
		},
	}

	err := insertDocument(db, doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	retrievedDoc, err := getDocumentByID(db, "1")
	if err != nil {
		t.Fatalf("Failed to retrieve document: %v", err)
	}

	if !compareDoc(doc, *retrievedDoc) {
		t.Errorf("Expected %#v, got %#v", doc, retrievedDoc)
	}
}

// Test handling /document requests
func TestHandleDocumentRequest(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	doc := XMLDoc{
		Title:       "Test Title",
		Description: "Test Description",
		Author:      "Test Author",
		CreatedAt:   "2024-07-09",
		XMLData: []string{
			"<title>Test Title</title>",
			"<description>Test Description</description>",
			"<author>Test Author</author>",
			"<creationDate>2024-07-09</creationDate>",
		},
	}

	err := insertDocument(db, doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	req := httptest.NewRequest("GET", "/document?id=1", nil)
	w := httptest.NewRecorder()

	handleRequest(db, w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	retrievedDoc, err := parseDocument(string(body))
	if err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if compareDoc(*retrievedDoc, doc) {
		t.Errorf("Expected %#v, got %#v", doc, retrievedDoc)
	}
}

// Test handling /add requests
func TestHandleAddRequest(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	xmlData := `<document>
		<title>Test Title</title>
		<description>Test Description</description>
		<author>Test Author</author>
		<creationDate>2024-07-09</creationDate>
	</document>`

	req := httptest.NewRequest("POST", "/add", strings.NewReader(xmlData))
	w := httptest.NewRecorder()

	handleRequest(db, w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	// Verify that the document was inserted
	retrievedDoc, err := getDocumentByID(db, "1")
	if err != nil {
		t.Fatalf("Failed to retrieve document: %v", err)
	}

	expectedDoc := XMLDoc{
		Title:       "Test Title",
		Description: "Test Description",
		Author:      "Test Author",
		CreatedAt:   "2024-07-09",
		XMLData: []string{
			"<title>Test Title</title>",
			"<description>Test Description</description>",
			"<author>Test Author</author>",
			"<creationDate>2024-07-09</creationDate>",
		},
	}

	if compareDoc(*retrievedDoc, expectedDoc) {
		t.Errorf("Expected %#v, got %#v", expectedDoc, retrievedDoc)
	}
}

// Test handling /del requests
func TestHandleDeleteRequest(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	doc := XMLDoc{
		Title:       "Test Title",
		Description: "Test Description",
		Author:      "Test Author",
		CreatedAt:   "2024-07-09",
		XMLData: []string{
			"<title>Test Title</title>",
			"<description>Test Description</description>",
			"<author>Test Author</author>",
			"<creationDate>2024-07-09</creationDate>",
		},
	}

	err := insertDocument(db, doc)
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	req := httptest.NewRequest("DELETE", "/del?id=1", nil)
	w := httptest.NewRecorder()

	handleRequest(db, w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Verify that the document was deleted
	_, err = getDocumentByID(db, "1")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Expected error %v, got %v", sql.ErrNoRows, err)
	}
}

// Test handling an invalid path
func TestHandleRequestInvalidPath(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/invalid", nil)
	w := httptest.NewRecorder()

	handleRequest(db, w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}
