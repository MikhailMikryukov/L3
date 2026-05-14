package models

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

// User пользователь
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Item данные
type Item struct {
	ID        int       `json:"id"`
	Name      string    `json:"name" binding:"omitempty"`
	Quantity  int       `json:"quantity" binding:"omitempty,min=0"`
	Price     float64   `json:"price" binding:"omitempty,min=0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ItemHistory история изменений
type ItemHistory struct {
	ID        int                    `json:"id"`
	ItemID    int                    `json:"item_id"`
	Action    string                 `json:"action"`
	OldData   map[string]interface{} `json:"old_data,omitempty"`
	NewData   map[string]interface{} `json:"new_data,omitempty"`
	ChangedBy string                 `json:"changed_by"`
	ChangedAt time.Time              `json:"changed_at"`
}

// Filter фильтрация истории изменений
type Filter struct {
	Action   string    `form:"action"`
	User     string    `form:"user"`
	ID       int       `form:"id"`
	DateFrom time.Time `form:"date_from" time_format:"2006-01-02T15:04"`
	DateTo   time.Time `form:"date_to" time_format:"2006-01-02T15:04"`
}

// Claims заголовки и информация для токена
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}
