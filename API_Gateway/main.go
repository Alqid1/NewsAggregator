package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type NewsFullDetailed struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type NewsShortDetailed struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

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
	api.r.HandleFunc("/news/latest", api.getLatestNews).Methods(http.MethodGet)
	api.r.HandleFunc("/news/filter", api.filterNews).Methods(http.MethodGet)
	api.r.HandleFunc("/news/{id}", api.getNewsDetail).Methods(http.MethodGet)
	api.r.HandleFunc("/news/{id}/comments", api.getComments).Methods(http.MethodGet)
	api.r.HandleFunc("/news/{id}/comments", api.addComment).Methods(http.MethodPost)
}

// Заглушка данных
var newsList = []NewsShortDetailed{
	{"1", "Breaking News 1"},
	{"2", "Breaking News 2"},
	{"3", "Breaking News 3"},
	{"4", "Breaking News 4"},
	{"5", "Breaking News 5"},
	{"6", "Breaking News 6"},
	{"7", "Breaking News 7"},
	{"8", "Breaking News 8"},
	{"9", "Breaking News 9"},
	{"10", "Breaking News 10"},
}

var newsDetails = map[string]NewsFullDetailed{
	"1":  {"1", "Breaking News 1", "Detailed content of Breaking News 1"},
	"2":  {"2", "Breaking News 2", "Detailed content of Breaking News 2"},
	"3":  {"3", "Breaking News 3", "Detailed content of Breaking News 3"},
	"4":  {"4", "Breaking News 4", "Detailed content of Breaking News 4"},
	"5":  {"5", "Breaking News 5", "Detailed content of Breaking News 5"},
	"6":  {"6", "Breaking News 6", "Detailed content of Breaking News 6"},
	"7":  {"7", "Breaking News 7", "Detailed content of Breaking News 7"},
	"8":  {"8", "Breaking News 8", "Detailed content of Breaking News 8"},
	"9":  {"9", "Breaking News 9", "Detailed content of Breaking News 9"},
	"10": {"10", "Breaking News 10", "Detailed content of Breaking News 10"},
}

var comments = map[string][]Comment{
	"1": {
		{"1", "User1", "Comment for Breaking News 1"},
		{"2", "User2", "Another comment for Breaking News 1"},
	},
	"2": {
		{"1", "User3", "Comment for Breaking News 2"},
	},
}

func (api *API) getLatestNews(w http.ResponseWriter, r *http.Request) {
	// Извлекаем параметр page из строки запроса
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1" // по умолчанию устанавливаем страницу 1, если параметр не передан
	}

	// Преобразуем page в целое число
	pageNumber := 1
	if p, err := strconv.Atoi(page); err == nil {
		pageNumber = p
	}

	// Устанавливаем количество новостей на странице (например, 3 новости на странице)
	perPage := 3
	start := (pageNumber - 1) * perPage
	end := start + perPage
	if end > len(newsList) {
		end = len(newsList)
	}

	// Загружаем новости для текущей страницы
	pageNews := newsList[start:end]

	// Отправляем список новостей в формате JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(pageNews); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Обработчик для фильтрации новостей
func (api *API) filterNews(w http.ResponseWriter, r *http.Request) {
	// Извлекаем параметры фильтра из строки запроса
	title := r.URL.Query().Get("title")
	category := r.URL.Query().Get("category")

	// Фильтруем новости по названию и категории
	var filteredNews []NewsShortDetailed
	for _, news := range newsList {
		if (title == "" || strings.Contains(news.Title, title)) &&
			(category == "" || strings.Contains(news.Title, category)) {
			filteredNews = append(filteredNews, news)
		}
	}

	// Отправляем отфильтрованные новости в формате JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(filteredNews); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Обработчик для получения детальной информации о новости
func (api *API) getComments(w http.ResponseWriter, r *http.Request) {
	// Получаем параметр из URL
	params := mux.Vars(r)
	newsID := params["id"]

	// Создаем HTTP запрос к микросервису комментариев
	url := fmt.Sprintf("http://localhost:8081/comments/%s", newsID) // Микросервис комментариев на порту 8081

	// Выполняем GET запрос к микросервису
	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch comments: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Проверяем статус код ответа
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Failed to fetch comments: %s", resp.Status), resp.StatusCode)
		return
	}

	// Читаем тело ответа микросервиса комментариев
	bodyBytes, err := io.ReadAll(resp.Body) // Используем io.ReadAll, который заменяет ioutil.ReadAll в Go 1.16+
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading response body: %v", err), http.StatusInternalServerError)
		return
	}

	// Отправляем комментарии обратно пользователю
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(bodyBytes) // Отправляем содержимое в ответ
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
	}
}

// Обработчик для добавления комментария
func (api *API) addComment(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	newsID := params["id"]

	// Получаем данные комментария из тела запроса
	var newComment Comment
	if err := json.NewDecoder(r.Body).Decode(&newComment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Добавляем комментарий в список
	comments[newsID] = append(comments[newsID], newComment)

	// Отправляем добавленный комментарий в формате JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newComment); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Обработчик для получения детальной новости
func (api *API) getNewsDetail(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	newsID := params["id"]

	newsItem, exists := newsDetails[newsID]
	if !exists {
		http.Error(w, "News not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(newsItem); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	api := NewAPI()
	http.Handle("/", api.Router())
	fmt.Println("Server started at http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
