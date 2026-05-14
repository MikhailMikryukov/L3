package handlers

import (
	"L3.7/internal/middleware"
	"L3.7/internal/models"
	"L3.7/internal/repository/postgres"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

// ItemHandler обработчик айтемов
type ItemHandler struct {
	db *postgres.DB
}

// NewItemHandler создает обработчик
func NewItemHandler(db *postgres.DB) *ItemHandler {
	return &ItemHandler{db: db}
}

// CreateItem создает айтем
func (h *ItemHandler) CreateItem(c *gin.Context) {

	// Получаем айтем из запроса
	var item models.Item
	err := c.ShouldBindJSON(&item)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	item.CreatedAt = time.Now()

	// Получаем текущего пользователя
	user := middleware.GetUserFromContext(c)

	// Устанавливаем текущего пользователя в бд
	err = h.db.SetCurrentUser(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error setting current user"})
	}
	defer h.db.SetCurrentUser("")

	// Сохраняем в бд
	err = h.db.SaveItem(item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error saving the item "})
	}

	c.JSON(http.StatusCreated, item)
}

// GetItem получает айтем
func (h *ItemHandler) GetItem(c *gin.Context) {
	idStr := c.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid id"})
		return
	}

	item, err := h.db.GetItem(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error getting the item"})
	}

	c.JSON(http.StatusOK, item)
}

// GetAllItems получает все айтемы
func (h *ItemHandler) GetAllItems(c *gin.Context) {
	items, err := h.db.GetAllItems()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error getting the item"})
	}

	c.JSON(http.StatusOK, items)
}

// UpdateItem обновляет айтем
func (h *ItemHandler) UpdateItem(c *gin.Context) {
	idStr := c.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid id"})
		return
	}

	// Проверяем существует ли айтем
	exists, err := h.db.Exists(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error getting the item"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "no such item"})
		return
	}

	// Получаем обновленный айтем из запроса
	var updatedItem models.Item
	err = c.ShouldBindJSON(&updatedItem)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Получаем из бд айтем
	oldItem, err := h.db.GetItem(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error getting the item"})
		return
	}

	// Записываем поля, которые нужно обновить
	updates := make(map[string]interface{})

	if updatedItem.Name != oldItem.Name {
		updates["name"] = updatedItem.Name
	}

	if updatedItem.Quantity != oldItem.Quantity {
		updates["quantity"] = updatedItem.Quantity
	}

	if updatedItem.Price != oldItem.Price {
		updates["price"] = updatedItem.Price
	}

	// Получаем текущего пользователя
	user := middleware.GetUserFromContext(c)

	// Устанавливаем текущего пользователя в бд
	err = h.db.SetCurrentUser(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error setting current user"})
	}
	defer h.db.SetCurrentUser("")

	// Записываем новые данные в бд
	err = h.db.UpdateItem(id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error updating the item"})
	}

	c.JSON(http.StatusOK, gin.H{"response": fmt.Sprintf("item %s updated", id)})
}

// DeleteItem удаляет айтем
func (h *ItemHandler) DeleteItem(c *gin.Context) {
	idStr := c.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid id"})
		return
	}

	// Проверяем существует ли айтем
	exists, err := h.db.Exists(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error getting the item"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "no such item"})
		return
	}

	// Получаем текущего пользователя
	user := middleware.GetUserFromContext(c)

	// Устанавливаем текущего пользователя в бд
	err = h.db.SetCurrentUser(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error setting current user"})
	}
	defer h.db.SetCurrentUser("")

	// Удаляем из бд
	err = h.db.DeleteItem(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error deleting the item"})
	}

	c.JSON(http.StatusOK, gin.H{"response": fmt.Sprintf("item %s deleted", id)})
}
