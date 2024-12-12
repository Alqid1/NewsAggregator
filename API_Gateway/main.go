package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type NewsFullDetailed struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type NewsShortDetailed struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

type Comment struct {
	ID        int       `json:"id"`
	NewsID    int       `json:"news_id"` // ID новости, к которой относится комментарий
	Author    string    `json:"author"`
	Text      string    `json:"text"`       // Текст комментария
	ParentID  int       `json:"parent_id"`  // ID родительского комментария (если это ответ)
	CreatedAt time.Time `json:"created_at"` // Дата и время создания комментария
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

func (api *API) Router() *mux.Router {
	return api.r
}

func (api *API) endpoints() {
	api.r.HandleFunc("/news", api.getNews).Methods(http.MethodGet)
	api.r.HandleFunc("/news/filter", api.filterNews).Methods(http.MethodGet)
	api.r.HandleFunc("/news/{id}/comments", api.getComments).Methods(http.MethodGet)
	api.r.HandleFunc("/news/{id}/comments", api.addComment).Methods(http.MethodPost)
}

// Заглушка данных
var newsList = []NewsShortDetailed{}

var newsDetails = map[string]NewsFullDetailed{}

func (api *API) getNews(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		requestID = generateRequestID()
	}
	page := r.URL.Query().Get("page")
	s := r.URL.Query().Get("s")

	// Создаем HTTP запрос к микросервису комментариев
	url := fmt.Sprintf("http://localhost:8082/news?request_id=%s&page=%s&s=%s", requestID, page, s) // Микросервис комментариев на порту 8081

	// Выполняем GET запрос к микросервису
	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch comments: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Failed to fetch comments: %s", resp.Status), resp.StatusCode)
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading response body: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(bodyBytes) // Отправляем содержимое в ответ
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
	}
}

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

func (api *API) getComments(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		requestID = generateRequestID()
	}

	params := mux.Vars(r)
	newsID := params["id"]

	// Создаем HTTP запрос к микросервису комментариев
	url := fmt.Sprintf("http://localhost:8081/comments/%s?request_id=%s", newsID, requestID) // Микросервис комментариев на порту 8081

	// Выполняем GET запрос к микросервису
	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch comments: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Failed to fetch comments: %s", resp.Status), resp.StatusCode)
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading response body: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(bodyBytes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
	}
}

func (api *API) addComment(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	newsIDStr := params["id"]

	newsID, err := strconv.Atoi(newsIDStr)
	if err != nil {
		http.Error(w, "Invalid NewsID", http.StatusBadRequest)
		return
	}

	var newComment Comment
	if err := json.NewDecoder(r.Body).Decode(&newComment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newComment.NewsID = newsID

	commentJSON, err := json.Marshal(newComment)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal comment: %v", err), http.StatusInternalServerError)
		return
	}

	// Создаем HTTP POST запрос к микросервису комментариев
	url := fmt.Sprintf("http://localhost:8081/comments/%s", newsIDStr)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(commentJSON))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Выполняем POST запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to send request to microservice: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		http.Error(w, fmt.Sprintf("Microservice error: %s", resp.Status), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ клиенту
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newComment); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader записывает HTTP-статус и сохраняет его в переменную
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Write записывает тело ответа
func (rw *responseWriter) Write(p []byte) (n int, err error) {
	return rw.ResponseWriter.Write(p)
}

func HeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.URL.Query().Get("request_id")
		if requestID == "" {
			requestID = generateRequestID()
		}

		wrappedWriter := &responseWriter{ResponseWriter: w}

		next.ServeHTTP(wrappedWriter, r)

		log.Printf(
			"Request ID: %s | Time: %s | IP: %s | Status: %d",
			requestID,
			time.Now().Format(time.RFC3339),
			r.RemoteAddr,
			wrappedWriter.statusCode,
		)

	})
}

func main() {
	api := NewAPI()
	api.Router().Use(HeadersMiddleware)
	http.Handle("/", api.Router())
	fmt.Println("Server started at http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
