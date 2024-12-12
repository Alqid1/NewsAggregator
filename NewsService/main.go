package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
)

/* type NewsFullDetailed struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Author string `json:"author"`
	Content string `json:"content"`
	CreatedAt time.Time `json:"created_at"`
} */

type NewsShortDetailed struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

type Pagination struct {
	TotalPages  int `json:"totalPages"`
	CurrentPage int `json:"currentPage"`
	PageSize    int `json:"pageSize"`
}

type API struct {
	r  *mux.Router // маршрутизатор запросов
	db *pgxpool.Pool
}

const pageSize = 15

func NewAPI(db *pgxpool.Pool) *API {
	api := &API{
		r:  mux.NewRouter(),
		db: db,
	}
	api.endpoints() // Настройка маршрутов
	return api
}

func (api *API) Router() *mux.Router {
	return api.r
}

func (api *API) endpoints() {
	// Обработчики для различных маршрутов
	api.r.HandleFunc("/news", api.getNews).Methods(http.MethodGet)
}

func initDB() *pgxpool.Pool {
	var err error

	connString := "host=localhost port=5432 user=postgres password=postgres dbname=NewsService sslmode=disable"

	db, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	return db
}

func (api *API) getNews(w http.ResponseWriter, r *http.Request) {
	s := r.URL.Query().Get("s")
	pageParam := r.URL.Query().Get("page")
	if pageParam == "" {
		pageParam = "1" // Если параметр не передан, то по умолчанию первая страница
	}

	page, err := strconv.Atoi(pageParam)
	if err != nil || page <= 0 {
		http.Error(w, "Invalid page parameter", http.StatusBadRequest)
		return
	}

	var totalCount int
	err = api.db.QueryRow(context.Background(), `
	SELECT COUNT(*) FROM news
	WHERE title ILIKE $1;
	`, "%"+s+"%").Scan(&totalCount)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch total count: %v", err), http.StatusInternalServerError)
		return
	}

	totalPages := (totalCount + pageSize - 1) / pageSize

	pagination := Pagination{
		TotalPages:  totalPages,
		CurrentPage: page,
		PageSize:    pageSize,
	}

	offset := (page - 1) * pageSize
	rows, err := api.db.Query(context.Background(), `
		SELECT * FROM news
		WHERE title ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
		`, "%"+s+"%", pageSize, offset)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch comments: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var news []NewsShortDetailed
	for rows.Next() {
		var soloNews NewsShortDetailed

		// Чтение данных из строки
		if err := rows.Scan(&soloNews.ID, &soloNews.Title, &soloNews.Author, &soloNews.CreatedAt); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning comment: %v", err), http.StatusInternalServerError)
			return
		}

		news = append(news, soloNews)
	}

	response := struct {
		News       []NewsShortDetailed `json:"news"`
		Pagination interface{}         `json:"pagination"`
	}{
		News:       news,
		Pagination: pagination,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

func main() {
	db := initDB()
	defer db.Close()

	api := NewAPI(db)

	log.Fatal(http.ListenAndServe(":8082", api.r))
}
