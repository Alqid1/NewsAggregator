package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4/pgxpool"
)

var db *pgxpool.Pool

type Comment struct {
	ID      string `json:"id"`
	Author  string `json:"author"`
	Content string `json:"content"`
}

type API struct {
	r *mux.Router // маршрутизатор запросов
}

func NewAPI() *API {
	api := &API{
		r: mux.NewRouter(), // Инициализация маршрутизатора
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

func initDB() {
	var err error
	// Строка подключения к PostgreSQL
	connString := "host=localhost port=5432 user=postgres password=postgres dbname=Comments sslmode=disable" // Замените username, password, dbname

	// Создаем пул соединений с базой данных
	db, err = pgxpool.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

}

func (api *API) getComments(w http.ResponseWriter, r *http.Request) {

}
