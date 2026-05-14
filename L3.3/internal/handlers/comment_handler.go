package handlers

import (
	"L3.3/internal/entities"
	"L3.3/internal/services"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type errorResponse struct {
	Error string `json:"error"`
}

type commentResponse struct {
	ID        int       `json:"id"`
	Author    string    `json:"author"`
	Parent    int       `json:"parent_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// CommentHandler обработчик запросов
type CommentHandler struct {
	s *services.CommentService
}

// New создает экземпляр CommentHandler
func New(cs *services.CommentService) *CommentHandler {
	return &CommentHandler{s: cs}
}

// NewRouter создание и настройка обработчика
func NewRouter(cs *services.CommentService) *http.ServeMux {
	mux := http.NewServeMux()
	commentHandler := New(cs)

	// Страница для проверки работы приложения
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	})
	// Загрузка комментариев
	mux.HandleFunc("/comments/load", commentHandler.loadComments)
	mux.HandleFunc("/comments/flat", commentHandler.getCommentsForParentOption)

	// Обработка запроса основной функции
	mux.HandleFunc("/comments", commentHandler.handle)

	return mux
}

func (h *CommentHandler) handle(w http.ResponseWriter, r *http.Request) {
	// Создание комментария
	if r.Method == "POST" {
		author := r.FormValue("author")
		parent := r.FormValue("parent")
		text := r.FormValue("text")

		err := h.s.CreateComment(author, parent, text)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Формируем ответ
		h.writeJSON(w, http.StatusOK, map[string]bool{
			"success": true,
		})
		return
	}

	// Получение комментария и всех вложенных
	if r.Method == "GET" {
		parent := r.FormValue("parent")
		if parent == "" {
			return
		}
		comments, err := h.s.GetCommentAndChild(parent)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		response := h.convertCommentToResponse(comments)

		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"comments": response,
		})
		return
	}

	// Удаление комментария
	if r.Method == "DELETE" {
		id := r.FormValue("id")
		err := h.s.DeleteComment(id)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		h.writeJSON(w, http.StatusOK, map[string]bool{
			"success": true,
		})
		return
	}
}

// getCommentsForParentOption получение комментариев для выбора в качестве родительского
func (h *CommentHandler) getCommentsForParentOption(w http.ResponseWriter, r *http.Request) {
	limitStr := r.FormValue("limit")

	if limitStr != "" {

		limit, _ := strconv.Atoi(limitStr)
		//comments, err := h.s.GetAllComments(limit)
		comments, err := h.s.GetComments(limit, -1, "", "")
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		response := h.convertCommentToResponse(comments)

		h.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"comments": response,
		})
		return
	}
}

// loadComments загрузка комментариев на страницу
func (h *CommentHandler) loadComments(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры из URL
	pageStr := r.FormValue("page")
	limitStr := r.FormValue("limit")
	sortBy := r.FormValue("sort_by")
	searchQuery := r.FormValue("search")

	// Преобразуем строки в числа
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(limitStr)
	if limit < 1 || limit > 100 {
		limit = 10 // ограничиваем максимальное количество
	}

	// Вычисляем OFFSET для SQL
	offset := (page - 1) * limit

	comments, err := h.s.GetComments(limit, offset, sortBy, searchQuery)
	if err != nil {
		fmt.Println(err)
	}

	response := h.convertCommentToResponse(comments)

	// Вычисляем общее количество страниц
	commentsPerPage := 10
	totalCount := len(comments)

	totalPages := (limit + commentsPerPage - 1) / commentsPerPage

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"page":        page,
		"total_pages": totalPages,
		"total_count": totalCount,
		"comments":    response,
	})
}

func (h *CommentHandler) convertCommentToResponse(comments []entities.Comment) []commentResponse {
	result := make([]commentResponse, 0, len(comments))

	for i, _ := range comments {
		var comment commentResponse
		comment.Text = comments[i].Text
		comment.Author = comments[i].Author
		comment.ID = comments[i].ID
		comment.CreatedAt = comments[i].CreatedAt
		comment.Parent = comments[i].Parent
		result = append(result, comment)
	}

	return result
}

func (h *CommentHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *CommentHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, errorResponse{Error: message})
}
