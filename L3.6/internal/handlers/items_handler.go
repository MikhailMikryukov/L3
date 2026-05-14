package handlers

import (
	"L3.6/internal/entities"
	"L3.6/internal/repository/postgres"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Response ответ
type Response struct {
	Status   string `json:"status"`
	Response string `json:"response"`
}

// ItemsHandler обработчик запросов
type ItemsHandler struct {
	db  *postgres.DB
}

// New создает экземпляр EventHandler
func New(db *postgres.DB) *ItemsHandler {
	return &ItemsHandler{
		db:  db,
	}
}

// NewRouter создание и настройка обработчика
func NewRouter(db *postgres.DB) *http.ServeMux {
	mux := http.NewServeMux()
	itemsHandler := New(db)

	// Страница для проверки работы приложения
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	})

	mux.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		// Создание данных
		if r.Method == "POST" {
			itemsHandler.createItem(w, r)
			return
		}

		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Получение данных
		itemsHandler.GetFiltered(w, r)
	})

	mux.HandleFunc("/items/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/items/")
		// Обновление данных
		if r.Method == "PUT" {
			itemsHandler.updateItem(w, r, id)
			return
		}

		if r.Method == "GET" {
			itemsHandler.GetItem(w, r, id)
			return
		}

		if r.Method != "DELETE" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Удаление данных
		itemsHandler.deleteItem(w, r, id)

		return
	})

	// Получить агрегированую аналитику
	mux.HandleFunc("/analytics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		itemsHandler.GetAnalytics(w, r)
	})

	return mux
}

func (h *ItemsHandler) createItem(w http.ResponseWriter, r *http.Request) {

	typeStr := r.FormValue("type")
	amountStr := r.FormValue("amount")
	dateStr := r.FormValue("date")
	category := r.FormValue("category")
	comment := r.FormValue("comment")

	if category == "" {
		h.writeJSON(w, http.StatusBadRequest, Response{
			Status:   "error",
			Response: "укажите категорию",
		})
		return
	}

	if typeStr == "" || (typeStr != "income" && typeStr != "expense") {
		h.writeJSON(w, http.StatusBadRequest, Response{
			Status:   "error",
			Response: "укажите тип Доход/Расход",
		})
		return
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, Response{
			Status:   "error",
			Response: "некорректное значение суммы",
		})
		return
	}

	if amount < 0 {
		amount = -amount
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, Response{
			Status:   "error",
			Response: "некорректное значение даты. формат YYYY-MM-DD",
		})
		return
	}
	id := uuid.New().String()

	item := entities.Item{
		ID:       id,
		Type:     typeStr,
		Amount:   amount,
		Date:     date,
		Category: category,
		Comment:  comment,
	}

	err = h.db.CreateItem(item)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{
			Status:   "error",
			Response: fmt.Sprintf("не удалось записать данные в бд: %v", err),
		})
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Status:   "success",
		Response: item.ID,
	})
}

func (h *ItemsHandler) updateItem(w http.ResponseWriter, r *http.Request, id string) {
	exists, err := h.db.Exists(id)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{
			Status:   "error",
			Response: fmt.Sprintf("не удалось получить данные: %v", err),
		})
	}

	if !exists {
		h.writeJSON(w, http.StatusNotFound, Response{
			Status:   "not found",
			Response: "нет таких данных по данному ID",
		})
	}

	typeStr := r.FormValue("type")
	amountStr := r.FormValue("amount")
	dateStr := r.FormValue("date")
	category := r.FormValue("category")
	comment := r.FormValue("comment")

	var updates = make(map[string]interface{})

	if typeStr != "" {
		updates["type"] = typeStr
	}

	if amountStr != "" {
		amount, err := strconv.Atoi(amountStr)
		if err != nil {
			h.writeJSON(w, http.StatusInternalServerError, Response{
				Status:   "error",
				Response: "некорректное значение суммы",
			})
			return
		}
		updates["amount"] = amount
	}

	if dateStr != "" {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			h.writeJSON(w, http.StatusInternalServerError, Response{
				Status:   "error",
				Response: "некорректное значение даты. формат YYYY-MM-DD",
			})
			return
		}
		updates["date"] = date
	}

	if category != "" {
		updates["category"] = category
	}

	if comment != "" {
		updates["comment"] = comment
	}

	err = h.db.UpdateItem(id, updates)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{
			Status:   "error",
			Response: fmt.Sprintf("не удалось обновить данные: %v", err),
		})
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ItemsHandler) deleteItem(w http.ResponseWriter, r *http.Request, id string) {
	exists, err := h.db.Exists(id)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{
			Status:   "error",
			Response: fmt.Sprintf("не удалось получить данные: %v", err),
		})
	}

	if !exists {
		h.writeJSON(w, http.StatusNotFound, Response{
			Status:   "not found",
			Response: fmt.Sprintf("нет таких данных: %v", err),
		})
	}

	err = h.db.DeleteItem(id)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{
			Status:   "error",
			Response: fmt.Sprintf("не удалось удалить запись: %v", err),
		})
	}
	return
}

// GetAnalytics получить агрегированную аналитику
func (h *ItemsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	fromStr := r.FormValue("from")
	toStr := r.FormValue("to")

	var from time.Time
	var err error
	if fromStr != "" {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			h.writeJSON(w, http.StatusInternalServerError, Response{
				Status:   "error",
				Response: "некорректное значение даты. формат YYYY-MM-DD",
			})
			return
		}
	}

	var to time.Time
	if toStr != "" {
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			h.writeJSON(w, http.StatusInternalServerError, Response{
				Status:   "error",
				Response: "некорректное значение даты. формат YYYY-MM-DD",
			})
			return
		}
	}

	if fromStr == "" && toStr == "" {
		from = time.Time{}
		to = time.Now()
	}

	analytics, err := h.db.GetAnalytics(from, to)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{
			Status:   "error",
			Response: fmt.Sprintf("не удалось получить данные: %v", err),
		})
		return
	}

	h.writeJSON(w, http.StatusOK, analytics)
}

// GetFiltered получить фильтрованные данные
func (h *ItemsHandler) GetFiltered(w http.ResponseWriter, r *http.Request) {
	fromStr := r.FormValue("from")
	toStr := r.FormValue("to")
	typeStr := r.FormValue("type")
	category := r.FormValue("category")
	amountMinStr := r.FormValue("amountMin")
	amountMaxStr := r.FormValue("amountMax")
	search := r.FormValue("search")
	sortBy := r.FormValue("sortBy")
	sortOrder := r.FormValue("sortOrder")
	limitStr := r.FormValue("limit")
	pageStr := r.FormValue("page")

	filter := entities.Filter{}

	if fromStr != "" {
		from, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			h.writeJSON(w, http.StatusInternalServerError, Response{
				Status:   "error",
				Response: "некорректное значение даты. формат YYYY-MM-DD",
			})
			return
		}
		filter.From = from
	}

	if toStr != "" {
		to, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			h.writeJSON(w, http.StatusInternalServerError, Response{
				Status:   "error",
				Response: "некорректное значение даты. формат YYYY-MM-DD",
			})
			return
		}
		filter.From = to
	}

	if typeStr != "" && typeStr != "all" {
		filter.Type = typeStr
	}

	if category != "" && category != "all" {
		filter.Category = category
	}

	if amountMinStr != "" {
		amountMin, err := strconv.Atoi(amountMinStr)
		if err != nil {
			h.writeJSON(w, http.StatusInternalServerError, Response{
				Status:   "error",
				Response: "некорректное значение суммы",
			})
		}
		filter.AmountMin = amountMin
	}

	if amountMaxStr != "" {
		amountMax, err := strconv.Atoi(amountMaxStr)
		if err != nil {
			h.writeJSON(w, http.StatusInternalServerError, Response{
				Status:   "error",
				Response: "некорректное значение суммы",
			})
		}
		filter.AmountMax = amountMax
	}

	if search != "" {
		filter.Search = search
	}

	if sortBy != "" {
		filter.SortBy = sortBy
	}

	if sortOrder != "" {
		filter.SortOrder = sortOrder
	}

	var limit int
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			h.writeJSON(w, http.StatusInternalServerError, Response{
				Status:   "error",
				Response: "некорректное значение лимита",
			})
		}
		filter.Limit = limit
	}

	if pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			h.writeJSON(w, http.StatusInternalServerError, Response{
				Status:   "error",
				Response: "некорректное значение страницы",
			})
		}
		filter.Offset = (page - 1) * limit
	}

	items, err := h.db.GetFiltered(filter)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{
			Status:   "error",
			Response: fmt.Sprintf("не удалось получить данные: %s", err),
		})
	}

	totalPages, err := h.db.GetTotalPages(filter)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{
			Status:   "error",
			Response: fmt.Sprintf("не удалось получить данные: %s", err),
		})
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_pages": totalPages,
		"items":       items,
	})
}

// GetItem получить один айтем
func (h *ItemsHandler) GetItem(w http.ResponseWriter, r *http.Request, id string) {
	item, err := h.db.GetItem(id)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{
			Status:   "error",
			Response: fmt.Sprintf("не удалось получить данные: %s", err),
		})
	}

	json.NewEncoder(w).Encode(item)
}

func (h *ItemsHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
