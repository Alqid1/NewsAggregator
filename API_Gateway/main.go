package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type NewsFullDetailed struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type NewsShortDetailed struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

type Comment struct {
	ID        int       `json:"id"`
	NewsID    int       `json:"news_id"`
	Author    string    `json:"author"`
	Text      string    `json:"text"`
	ParentID  int       `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
}

type API struct {
	r *mux.Router
}

func NewAPI() *API {
	api := &API{
		r: mux.NewRouter(),
	}
	api.endpoints()
	return api
}

func (api *API) Router() *mux.Router {
	return api.r
}

func (api *API) endpoints() {
	api.r.HandleFunc("/news", api.getNews).Methods(http.MethodGet)
	api.r.HandleFunc("/news/{id}", api.getComments).Methods(http.MethodGet)
	api.r.HandleFunc("/news/{id}", api.getSoloNews).Methods(http.MethodGet)
	api.r.HandleFunc("/news/{id}/comments", api.addComment).Methods(http.MethodPost)
}

func (api *API) getNews(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		requestID = generateRequestID()
	}
	page := r.URL.Query().Get("page")
	s := r.URL.Query().Get("s")

	// Создаем HTTP запрос к микросервису новостей
	url := fmt.Sprintf("http://localhost:8082/news?request_id=%s&page=%s&s=%s", requestID, page, s) // Микросервис комментариев на порту 8082

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

func (api *API) getSoloNews(w http.ResponseWriter, r *http.Request) {
	param := mux.Vars(r)
	id := param["id"]
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		requestID = generateRequestID()
	}

	resultCh := make(chan interface{}, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	// Горутина для запроса новости
	go func() {
		defer wg.Done()
		url := fmt.Sprintf("http://localhost:8082/news/%s?request_id=%s", id, requestID)
		resp, err := http.Get(url)
		if err != nil {
			resultCh <- fmt.Errorf("Failed to fetch news: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			resultCh <- fmt.Errorf("Failed to fetch news: %s", resp.Status)
			return
		}

		var news NewsFullDetailed
		if err := json.NewDecoder(resp.Body).Decode(&news); err != nil {
			resultCh <- fmt.Errorf("Error decoding news response: %v", err)
			return
		}

		resultCh <- news
	}()

	// Горутина для запроса комментариев
	go func() {
		defer wg.Done()
		url := fmt.Sprintf("http://localhost:8081/comments/%s?request_id=%s", id, requestID)
		resp, err := http.Get(url)
		if err != nil {
			resultCh <- fmt.Errorf("Failed to fetch comments: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			resultCh <- fmt.Errorf("Failed to fetch comments: %s", resp.Status)
			return
		}

		var comments []Comment
		if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
			resultCh <- fmt.Errorf("Error decoding comments response: %v", err)
			return
		}

		resultCh <- comments
	}()

	// Ожидаем завершения всех горутин
	wg.Wait()

	var newsData interface{}
	var commentData interface{}

	// Получаем данные и ошибки из канала
	for i := 0; i < 2; i++ {
		result := <-resultCh
		switch data := result.(type) {
		case error:
			http.Error(w, data.Error(), http.StatusInternalServerError)
			return
		case NewsFullDetailed:
			newsData = data
		case []Comment:
			commentData = data
		}
	}

	var finalResponse string
	if newsData != nil {
		news := newsData.(NewsFullDetailed)
		finalResponse = fmt.Sprintf("{\"news\": {\"id\": %d, \"title\": \"%s\", \"author\": \"%s\", \"created_at\": \"%s\"},",
			news.ID, news.Title, news.Author, news.CreatedAt)
	} else {
		http.Error(w, "No news data found", http.StatusInternalServerError)
		return
	}

	if commentData != nil {
		comments := commentData.([]Comment)
		finalResponse += "\"comments\": ["
		for i, comment := range comments {
			finalResponse += fmt.Sprintf("{\"id\": %d, \"author\": \"%s\", \"content\": \"%s\", \"created_at\": \"%s\"}",
				comment.ID, comment.Author, comment.Text, comment.CreatedAt)
			if i < len(comments)-1 {
				finalResponse += ","
			}
		}
		finalResponse += "]}"
	} else {
		http.Error(w, "No comment data found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(finalResponse))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
	}
}

func (api *API) getComments(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		requestID = generateRequestID()
	}

	params := mux.Vars(r)
	newsID := params["id"]
	fmt.Println("testtttttttt")
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

	// Создаем HTTP POST запрос к сервису цензурирования
	censorURL := "http://localhost:8083/censor"
	req, err := http.NewRequest(http.MethodPost, censorURL, bytes.NewBuffer(commentJSON))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request to censor service: %v", err), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Выполняем POST запрос в сервис цензурирования
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to send request to censor service: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Если статус 400, то комментарий не прошел цензуру
	if resp.StatusCode == http.StatusBadRequest {
		http.Error(w, "Comment contains forbidden words", http.StatusBadRequest)
		return
	} else if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Error from censor service: %s", resp.Status), http.StatusInternalServerError)
		return
	}

	// Если цензура прошла успешно, отправляем запрос на создание комментария в сервис комментариев
	url := fmt.Sprintf("http://localhost:8081/comments/%s", newsIDStr)
	req, err = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(commentJSON))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request to comment service: %v", err), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Выполняем POST запрос
	resp, err = client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to send request to comment service: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		// Отправляем успешный ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(newComment); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}
	} else {
		http.Error(w, fmt.Sprintf("Error from comment service: %s", resp.Status), http.StatusInternalServerError)
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
