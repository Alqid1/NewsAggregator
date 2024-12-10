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
	NewsID    int       `json:"news_id"` // ID новости, к которой относится комментарий
	Author    string    `json:"author"`
	Text      string    `json:"text"`       // Текст комментария
	ParentID  int       `json:"parent_id"`  // ID родительского комментария (если это ответ)
	CreatedAt time.Time `json:"created_at"` // Дата и время создания комментария
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

// Router возвращает маршрутизатор запросов.
func (api *API) Router() *mux.Router {
	return api.r
}

// Регистрация методов API в маршрутизаторе запросов.
func (api *API) endpoints() {
	// Обработчики для различных маршрутов
	api.r.HandleFunc("/comments/{NewsID}", api.getComments).Methods(http.MethodGet)
	api.r.HandleFunc("/comments/{NewsID}", api.addComment).Methods(http.MethodPost)
}

func initDB() *pgxpool.Pool {
	var err error
	// Строка подключения к PostgreSQL
	connString := "host=localhost port=5432 user=postgres password=postgres dbname=Comments sslmode=disable" // Замените username, password, dbname

	// Создаем пул соединений с базой данных
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
	WHERE id = $1;
	`, newsID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch comments: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment

		// Чтение данных из строки
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
	// Получаем параметр NewsID из URL
	params := mux.Vars(r)
	newsID := params["NewsID"]

	// Парсим JSON из тела запроса
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

	// Преобразуем NewsID в число
	newsIDInt, err := strconv.Atoi(newsID)
	if err != nil {
		http.Error(w, "Invalid NewsID", http.StatusBadRequest)
		return
	}

	// Устанавливаем ID новости и текущую дату
	comment.NewsID = newsIDInt
	comment.CreatedAt = time.Now()

	// Устанавливаем ParentID в 0, если оно не было передано
	if comment.ParentID == 0 {
		comment.ParentID = 0
	}

	// Вставляем новый комментарий в базу данных
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

	// Ответ успешного добавления комментария
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status": "success"}`))
}

func main() {
	// Подключение к базе данных
	db := initDB()
	defer db.Close()

	// Создаем новый API с подключением к базе данных
	api := NewAPI(db)

	// Запускаем HTTP-сервер
	log.Fatal(http.ListenAndServe(":8081", api.r))
}
