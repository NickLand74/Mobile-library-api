package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/songs", getSongs)
	r.GET("/songs/:id/text", getSongText)
	r.DELETE("/songs/:id", deleteSong)
	r.PUT("/songs/:id", updateSong)
	r.POST("/songs", createSong)
	return r
}

func TestGetSongs(t *testing.T) {
	// Setup
	r := setupRouter()
	req, _ := http.NewRequest(http.MethodGet, "/songs?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d; got %d", http.StatusOK, w.Code)
	}
}

func TestCreateSong(t *testing.T) {
	// Setup
	r := setupRouter()
	jsonStr := []byte(`{"title":"Test Song", "artist":"Test Artist", "releaseDate":"2022-01-01", "text":"Test lyrics", "link":"test.com"}`)
	req, _ := http.NewRequest(http.MethodPost, "/songs", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d; got %d", http.StatusCreated, w.Code)
	}

	// Проверка содержимого ответа
	var song Song
	err := json.Unmarshal(w.Body.Bytes(), &song)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if song.Title != "Test Song" {
		t.Errorf("Expected song title to be 'Test Song'; got '%s'", song.Title)
	}
}

func TestGetSongText(t *testing.T) {
	// Setup
	r := setupRouter()
	req, _ := http.NewRequest(http.MethodGet, "/songs/1/text?page=1&limit=2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d; got %d", http.StatusOK, w.Code)
	}

	var verses []string
	err := json.Unmarshal(w.Body.Bytes(), &verses)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
}

func TestUpdateSong(t *testing.T) {
	// Setup
	r := setupRouter()
	jsonStr := []byte(`{"title":"Updated Song", "artist":"Updated Artist", "releaseDate":"2022-02-01", "text":"Updated lyrics", "link":"updated.com"}`)
	req, _ := http.NewRequest(http.MethodPut, "/songs/1", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d; got %d", http.StatusOK, w.Code)
	}
}

func TestDeleteSong(t *testing.T) {
	// Setup
	r := setupRouter()
	req, _ := http.NewRequest(http.MethodDelete, "/songs/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d; got %d", http.StatusOK, w.Code)
	}
}

func TestMain(m *testing.M) {
	if err := godotenv.Load("./.env"); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
