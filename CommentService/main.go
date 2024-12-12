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

type Comment struct {
	ID        int       `json:"id"`
	NewsID    int       `json:"news_id"`
	Author    string    `json:"author"`
	Text      string    `json:"text"`
	ParentID  int       `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
}

type API struct {
	r  *mux.Router // маршрутизатор запросов
	db *pgxpool.Pool
}

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
	api.r.HandleFunc("/comments/{NewsID}", api.getComments).Methods(http.MethodGet)
	api.r.HandleFunc("/comments/{NewsID}", api.addComment).Methods(http.MethodPost)
}

func initDB() *pgxpool.Pool {
	var err error

	connString := "host=localhost port=5432 user=postgres password=postgres dbname=Comments sslmode=disable" // Замените username, password, dbname

	db, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	return db
}

func (api *API) getComments(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	newsID := params["NewsID"]

	rows, err := api.db.Query(context.Background(), `
	SELECT * FROM comments
	WHERE news_id = $1
	ORDER BY created_at DESC;
	`, newsID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch comments: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment

		if err := rows.Scan(&comment.ID, &comment.NewsID, &comment.Author, &comment.Text, &comment.ParentID, &comment.CreatedAt); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning comment: %v", err), http.StatusInternalServerError)
			return
		}

		comments = append(comments, comment)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(comments); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

func (api *API) addComment(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	newsID := params["NewsID"]

	var comment Comment
	err := json.NewDecoder(r.Body).Decode(&comment)
	if err != nil {
		fmt.Println(comment)
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}
	log.Println(comment)

	// Проверка обязательных полей
	if comment.Text == "" || comment.Author == "" {
		http.Error(w, "Text and Author are required", http.StatusBadRequest)
		return
	}

	newsIDInt, err := strconv.Atoi(newsID)
	if err != nil {
		http.Error(w, "Invalid NewsID", http.StatusBadRequest)
		return
	}

	comment.NewsID = newsIDInt
	comment.CreatedAt = time.Now()

	if comment.ParentID == 0 {
		comment.ParentID = 0
	}

	_, err = api.db.Exec(
		context.Background(),
		`INSERT INTO comments (news_id, text, parent_id, created_at, author) 
         VALUES ($1, $2, $3, $4, $5)`,
		comment.NewsID, comment.Text, comment.ParentID, comment.CreatedAt, comment.Author,
	)
	if err != nil {
		http.Error(w, "Failed to insert comment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status": "success"}`))
}

func main() {
	db := initDB()
	defer db.Close()

	api := NewAPI(db)

	log.Fatal(http.ListenAndServe(":8081", api.r))
}
