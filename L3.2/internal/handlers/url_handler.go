package handlers

import (
	"L3.2/internal/services"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type errorResponse struct {
	Error string `json:"error"`
}

// URLHandler обработчик запросов
type URLHandler struct {
	us *services.URLService
}

// New создание экземпляра URLHandler
func New(us *services.URLService) *URLHandler {
	return &URLHandler{us: us}
}

// NewRouter создание и настройка обработчика
func NewRouter(us *services.URLService) *http.ServeMux {
	mux := http.NewServeMux()
	notificationHandler := New(us)

	// Страница для проверки работы приложения
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	})

	// Обработка создание короткой ссылки
	mux.HandleFunc("/shorten", notificationHandler.MakeShortLink)
	// Обработка переход по короткой ссылке
	mux.HandleFunc("/s/", notificationHandler.GoToShortURL)

	// Обработка переход по короткой ссылке при проверке работы приложения из браузера
	mux.HandleFunc("/r/", notificationHandler.GoToShortURLBrowser)

	// Обработка получения аналитики
	mux.HandleFunc("/analytics/", notificationHandler.GetAnalytics)

	// Обработка получения аналитики
	mux.HandleFunc("/aggregated/", notificationHandler.GetAggregatedAnalytics)

	return mux
}

// MakeShortLink создание короткой ссылки
func (h *URLHandler) MakeShortLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	URL := r.FormValue("url")

	shortURL, err := h.us.MakeShortURL(URL)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, shortURL)
}

// GoToShortURL Переход по короткой ссылке
func (h *URLHandler) GoToShortURL(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	shortURL := strings.TrimPrefix(path, "/s/")

	originalURL, err := h.us.GetOriginalURL(shortURL)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ip := r.Header.Get("X-Real-IP")
	err = h.us.SaveClick(shortURL, time.Now(), r.UserAgent(), ip)
	if err != nil {
		log.Println(err)
	}

	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		originalURL = "https://" + originalURL
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

// GetAnalytics получение аналитики о короткой ссылке
func (h *URLHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Получаем URL
	path := r.URL.Path
	shortURL := strings.TrimPrefix(path, "/analytics/")
	shortURL = strings.TrimSuffix(shortURL, "/")

	analytics, err := h.us.GetAnalytics(shortURL, "all", "", "")
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, analytics)
}

// GetAggregatedAnalytics получение аналитики за выбранный период
func (h *URLHandler) GetAggregatedAnalytics(w http.ResponseWriter, r *http.Request) {
	URL := r.FormValue("url")
	periodType := r.FormValue("period_type")
	from := r.FormValue("from")
	to := r.FormValue("to")

	analytics, err := h.us.GetAnalytics(URL, periodType, from, to)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, analytics)
}

// GoToShortURLBrowser переход по короткой ссылке для демонстрации в браузере
func (h *URLHandler) GoToShortURLBrowser(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Получаем URL
	path := r.URL.Path
	shortURL := strings.TrimPrefix(path, "/r/")

	// Получаем оригинальный URL
	originalURL, err := h.us.GetOriginalURL(shortURL)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Сохраняем информацию о переходе по ссылке
	ip := r.Header.Get("X-Real-IP")
	err = h.us.SaveClick(shortURL, time.Now(), r.UserAgent(), ip)
	if err != nil {
		log.Println(err)
	}

	// Добавляем протокол если нужно
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		originalURL = "https://" + originalURL
	}

	// Возвращаем URL в JSON
	h.writeJSON(w, http.StatusOK, map[string]string{
		"url": originalURL,
	})
}

func (h *URLHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *URLHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, errorResponse{Error: message})
}
