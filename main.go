package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sqlx.DB

type Song struct {
	ID          int    `json:"id" db:"id"`
	Title       string `json:"title" db:"title" binding:"required"`
	Artist      string `json:"artist" db:"artist" binding:"required"`
	ReleaseDate string `json:"releaseDate" db:"release_date" binding:"required"`
	Text        string `json:"text" db:"text" binding:"required"`
	Link        string `json:"link" db:"link" binding:"required"`
}

type Pagination struct {
	Limit int `form:"limit"`
	Page  int `form:"page"`
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	connStr := "host=" + os.Getenv("DB_HOST") +
		" port=" + os.Getenv("DB_PORT") +
		" user=" + os.Getenv("DB_USER") +
		" password=" + os.Getenv("DB_PASSWORD") +
		" dbname=" + os.Getenv("DB_NAME") +
		" sslmode=disable"

	db, err = sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
}
func getSongs(c *gin.Context) {
	var songs []Song
	var pagination Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := pagination.Limit
	if limit == 0 {
		limit = 10 // значение по умолчанию
	}
	offset := (pagination.Page - 1) * limit

	query := `SELECT * FROM songs ORDER BY id LIMIT $1 OFFSET $2`
	err := db.Select(&songs, query, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, songs)
}
func getSongText(c *gin.Context) {
	id := c.Param("id")
	var text string
	var pagination Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Для простоты текст песен делится на куплеты через разделитель "\n\n"
	query := `SELECT text FROM songs WHERE id = $1`
	err := db.Get(&text, query, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}

	verses := strings.Split(text, "\n\n") // Разделяем текст на куплеты
	start := (pagination.Page - 1) * pagination.Limit
	end := start + pagination.Limit
	if start >= len(verses) {
		c.JSON(http.StatusOK, []string{})
		return
	}
	if end > len(verses) {
		end = len(verses)
	}
	c.JSON(http.StatusOK, verses[start:end])
}
func deleteSong(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec(`DELETE FROM songs WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Song deleted successfully"})
}
func updateSong(c *gin.Context) {
	id := c.Param("id")
	var song Song
	if err := c.ShouldBindJSON(&song); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`UPDATE songs SET title = $1, artist = $2, release_date = $3, text = $4, link = $5 WHERE id = $6`,
		song.Title, song.Artist, song.ReleaseDate, song.Text, song.Link, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Song not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Song updated successfully"})
}
func createSong(c *gin.Context) {
	var song Song
	if err := c.ShouldBindJSON(&song); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`INSERT INTO songs (title, artist, release_date, text, link) VALUES ($1, $2, $3, $4, $5)`,
		song.Title, song.Artist, song.ReleaseDate, song.Text, song.Link)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, song)
}
func main() {
	router := gin.Default()

	router.GET("/songs", getSongs)
	router.GET("/songs/:id/text", getSongText)
	router.DELETE("/songs/:id", deleteSong)
	router.PUT("/songs/:id", updateSong)
	router.POST("/songs", createSong)

	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
