package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

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
	api.r.HandleFunc("/orders", api.myHttpHandler).Methods(http.MethodGet)
	api.r.HandleFunc("/news/filter", filterNews).Methods("GET")
	api.r.HandleFunc("/news/{id:[0-9]+}", getDetailedNews).Methods("GET")
	api.r.HandleFunc("/comments", addComment).Methods("POST")
}

func (api *API) myHttpHandler(w http.ResponseWriter, r *http.Request) {
	// если параметр был передан, вернется строка со значением.
	// Если не был - в переменной будет пустая строка
	pageParam := r.URL.Query().Get("page")
	// параметр page - это число, поэтому нужно сконвертировать
	// строку в число при помощи пакета strconv
	page, err := strconv.Atoi(pageParam)
	if err != nil {
		// обработка ошибки
	}
	fmt.Println(page)
}

func main() {
	r := mux.NewRouter()

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}
