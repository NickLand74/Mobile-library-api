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

// NullString обертка для sql.NullString с аннотацией Swagger
// @Description NullString is a string that can be null
type NullString struct {
	String string `json:"string"`
	Valid  bool   `json:"valid"` // Valid если не пустое
}

// SuccessResponse представляет структуру для успешных ответов API
// @Description Структура для успешного ответа в API
type SuccessResponse struct {
	Message string `json:"message"`
}

// Song представляет модель песни
// @Description Структура для представления информации о песне
type Song struct {
	// ID песни
	// @example 1
	ID int `json:"id" db:"id"`

	// Название песни
	// @example "Song Title"
	Title string `json:"title" db:"title" binding:"required"`

	// Исполнитель песни
	// @example "Artist Name"
	Artist string `json:"artist" db:"artist" binding:"required"`

	// Дата выпуска песни
	// @example "2023-11-26"
	ReleaseDate string `json:"releaseDate" db:"release_date" binding:"required"`

	// Текст песни
	// @example "This is the text of the song."
	Text string `json:"text" db:"text" binding:"required"`

	// Ссылка на песню
	// @example "https://example.com/song-link"
	Link string `json:"link" db:"link" binding:"required"`

	// Название группы (может быть null)
	// @example "Group Name"
	GroupName NullString `json:"groupName" db:"group_name"`
}

// Pagination представляет модель пагинации
type Pagination struct {
	Limit int `form:"limit"`
	Page  int `form:"page"`
}

// ErrorResponse представляет структуру для ошибок API
// @Description Структура для представления ошибок в API
type ErrorResponse struct {
	Error string `json:"error"`
}

// @title Music API
// @version 1.0
// @description This is a simple music management API.
// @host localhost:8080
// @basePath /
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

// @Summary Get Songs
// @Description Retrieve a list of songs with pagination
// @Param limit query int false "Limit the number of songs" default(10)
// @Param page query int false "Page number" default(1)
// @Produce json
// @Success 200 {array} Song
// @Failure 400 {object} ErrorResponse "Ошибка в запросе"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /songs [get]
func getSongs(c *gin.Context) {
	var songs []Song
	var pagination Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	limit := pagination.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := (pagination.Page - 1) * limit

	query := "SELECT * FROM songs ORDER BY id LIMIT $1 OFFSET $2"
	err := db.Select(&songs, query, limit, offset)
	if err != nil {
		log.Printf("SQL Error: %s", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal Server Error"})
		return
	}
	c.JSON(http.StatusOK, songs)
}

// @Summary Получить текст песни
// @Description Возвращает текст песни по ID с поддержкой пагинации
// @Accept json
// @Produce json
// @Param id path string true "ID песни"
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество куплетов на странице" default(10)
// @Success 200 {array} string "Успешный ответ"
// @Failure 400 {object} ErrorResponse "Ошибка в запросе"
// @Failure 404 {object} ErrorResponse "Песня не найдена"
// @Router /songs/{id}/text [get]
func getSongText(c *gin.Context) {
	id := c.Param("id")
	log.Printf("Received request to get song text for ID: %s", id)

	var text string
	var pagination Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		log.Printf("Pagination binding error: %s", err.Error())
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	query := "SELECT text FROM songs WHERE id = $1"
	err := db.Get(&text, query, id)
	if err != nil {
		log.Printf("Database error: %s", err.Error())
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Song not found"})
		return
	}

	if text == "" {
		log.Println("No text found for this song")
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "No text found for this song"})
		return
	}

	verses := strings.Split(text, "\n\n")
	start := (pagination.Page - 1) * pagination.Limit
	end := start + pagination.Limit
	log.Printf("Pagination range: start=%d, end=%d", start, end)

	if start >= len(verses) {
		c.JSON(http.StatusOK, []string{})
		return
	}

	if end > len(verses) {
		end = len(verses)
	}

	log.Printf("Returning verses: %v", verses[start:end])
	c.JSON(http.StatusOK, verses[start:end])
}

// @Summary Удалить песню
// @Description Удаляет песню по ID
// @Accept json
// @Produce json
// @Param id path string true "ID песни"
// @Success 200 {object} SuccessResponse "Успешный ответ"
// @Failure 404 {object} ErrorResponse "Песня не найдена"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /songs/{id} [delete]
func deleteSong(c *gin.Context) {
	id := c.Param("id")
	result, err := db.Exec("DELETE FROM songs WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Song not found"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "Song deleted successfully"})
}

// @Summary Обновить информацию о песне
// @Description Обновляет данные песни по ID
// @Accept json
// @Produce json
// @Param id path string true "ID песни"
// @Param song body Song true "Данные для обновления песни"
// @Success 200 {object} SuccessResponse "Успешный ответ"
// @Failure 400 {object} ErrorResponse "Ошибка в запросе"
// @Failure 404 {object} ErrorResponse "Песня не найдена"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /songs/{id} [put]
func updateSong(c *gin.Context) {
	id := c.Param("id")
	var song Song
	if err := c.ShouldBindJSON(&song); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	result, err := db.Exec("UPDATE songs SET title = $1, artist = $2, release_date = $3, text = $4, link = $5 WHERE id = $6",
		song.Title, song.Artist, song.ReleaseDate, song.Text, song.Link, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal server error"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Song not found"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Message: "Song updated successfully"})
}

// @Summary Создать новую песню
// @Description Создает новую песню в базе данных
// @Accept json
// @Produce json
// @Param song body Song true "Данные для новой песни"
// @Success 201 {object} Song "Успешный ответ с созданной песней"
// @Failure 400 {object} ErrorResponse "Ошибка в запросе"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /songs [post]
func createSong(c *gin.Context) {
	var song Song
	log.Println("Received request to create song")

	if err := c.ShouldBindJSON(&song); err != nil {
		log.Printf("Binding error: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	log.Printf("Created song struct: %+v", song)

	err := db.QueryRow("INSERT INTO songs (title, artist, release_date, text, link) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		song.Title, song.Artist, song.ReleaseDate, song.Text, song.Link).Scan(&song.ID)

	if err != nil {
		log.Printf("Database insert error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	log.Printf("Successfully created song with ID: %d", song.ID)
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
