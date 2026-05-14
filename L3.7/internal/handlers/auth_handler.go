package handlers

import (
	"L3.7/internal/models"
	"L3.7/internal/repository/postgres"
	"L3.7/internal/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

// AuthHandler обработчик аутентификации
type AuthHandler struct {
	db *postgres.DB
}

// LoginRequest запрос на логин
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
}

// RegisterRequest запрос на создание пользователя
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

// NewAuthHandler Новый обработчик
func NewAuthHandler(db *postgres.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

// Login логин
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.db.GetUser(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error getttin the user"})
		return
	}

	token, err := utils.GenerateToken(user.Username, user.Role)
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error generating the token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

// Register создание пользователя
func (h *AuthHandler) Register(c *gin.Context) {
	// Получаем данные из запроса
	var req RegisterRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := models.User{
		Username: req.Username,
		Role:     req.Role,
	}

	// Записываем в бд
	err = h.db.SaveUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error saving the user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
