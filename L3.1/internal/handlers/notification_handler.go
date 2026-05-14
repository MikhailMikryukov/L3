package handlers

import (
	"L3.1/internal/services"
	"encoding/json"
	"net/http"
	"strings"
)

type errorResponse struct {
	Error string `json:"error"`
}

// NotificationHandler обработчик запросов
type NotificationHandler struct {
	us *services.NotificationsService
}

// NewNotificationHandler создание экземпляра UserHandler
func NewNotificationHandler(userService *services.NotificationsService) *NotificationHandler {
	return &NotificationHandler{us: userService}
}

// NewRouter создание и настройка обработчика
func NewRouter(notificationService *services.NotificationsService) *http.ServeMux {
	mux := http.NewServeMux()
	notificationHandler := NewNotificationHandler(notificationService)

	// Страница для проверки работы приложения
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	})

	// Обработка получения всех уведомлений
	mux.HandleFunc("/all_notifications", notificationHandler.GetAllNotifications)
	// Обработка запроса на создание уведомления
	mux.HandleFunc("/notify", notificationHandler.CreateNotification)
	// Обработка запросов по ID
	mux.HandleFunc("/notify/", notificationHandler.RequestByNotificationID)

	return mux
}

// CreateNotification создание уведомления
func (h *NotificationHandler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}
	date := r.FormValue("date")
	event := r.FormValue("event")
	err := h.us.CreateNotification(date, event)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

// RequestByNotificationID обработка запроса по id
func (h *NotificationHandler) RequestByNotificationID(w http.ResponseWriter, r *http.Request) {
	// Получаем ID
	path := r.URL.Path
	id := strings.TrimPrefix(path, "/notify/")
	id = strings.TrimSuffix(id, "/")

	// Получаем статус уведомления
	if r.Method == "GET" {
		h.getNotificationStatus(w, r, id)
	}
	// Отменяем уведомление
	if r.Method == "DELETE" {
		h.deleteNotification(w, r, id)
	}
}

func (h *NotificationHandler) getNotificationStatus(w http.ResponseWriter, r *http.Request, id string) {
	status, err := h.us.GetNotificationStatus(id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]string{"status": status})
}
func (h *NotificationHandler) deleteNotification(w http.ResponseWriter, r *http.Request, id string) {
	err := h.us.DeleteNotification(id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetAllNotifications получить все уведомления (функция для отображения в браузере)
func (h *NotificationHandler) GetAllNotifications(w http.ResponseWriter, r *http.Request) {
	result, err := h.us.GetAllNotifications()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *NotificationHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *NotificationHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, errorResponse{Error: message})
}
