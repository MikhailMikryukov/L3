package handlers

import (
	"L3.7/internal/models"
	"L3.7/internal/repository/postgres"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

// HistoryHandler обработчик истории изменений
type HistoryHandler struct {
	db *postgres.DB
}

// NewHistoryHandler создание обработчика
func NewHistoryHandler(db *postgres.DB) *HistoryHandler {
	return &HistoryHandler{db: db}
}

// GetItemHistory получает историю изменений айтема
func (h *HistoryHandler) GetItemHistory(c *gin.Context) {
	idStr := c.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid id"})
		return
	}

	exists, err := h.db.Exists(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error checking the existence"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "no such item"})
		return
	}

	filter := models.Filter{
		ID: id,
	}

	history, err := h.db.GetItemHistory(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error getting history of the item"})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetAllHistory получает всю историю изменений
func (h *HistoryHandler) GetAllHistory(c *gin.Context) {
	var filter models.Filter
	err := c.ShouldBindQuery(&filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid filter data" + err.Error()})
		return
	}

	history, err := h.db.GetItemHistory(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error getting history"})
		return
	}

	c.JSON(http.StatusOK, history)
}
