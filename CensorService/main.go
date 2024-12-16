package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var forbiddenWords = []string{"qwerty", "йцукен", "zxvbnm"}

type Comment struct {
	ID        int       `json:"id"`
	NewsID    int       `json:"news_id"`
	Author    string    `json:"author"`
	Text      string    `json:"text"`
	ParentID  int       `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
}

type API struct {
	r *mux.Router // маршрутизатор запросов
}

func NewAPI() *API {
	api := &API{
		r: mux.NewRouter(),
	}
	api.endpoints() // Настройка маршрутов
	return api
}

func (api *API) Router() *mux.Router {
	return api.r
}

func (api *API) endpoints() {
	// Обработчики для различных маршрутов
	api.r.HandleFunc("/censor", api.censorComment).Methods(http.MethodPost)
}

func (api *API) censorComment(w http.ResponseWriter, r *http.Request) {
	// Чтение тела запроса
	var comment Comment
	err := json.NewDecoder(r.Body).Decode(&comment)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверка на наличие запрещенных слов
	if containsForbiddenWords(comment.Text) {
		http.Error(w, "Comment contains forbidden words", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Comment approved"))
}

func containsForbiddenWords(text string) bool {
	for _, word := range forbiddenWords {
		if strings.Contains(strings.ToLower(text), word) {
			return true
		}
	}
	return false
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
	fmt.Println("Server started at http://localhost:8083/")
	log.Fatal(http.ListenAndServe(":8083", api.r))
}
