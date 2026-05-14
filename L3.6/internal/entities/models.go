package entities

import (
	"time"
)

// Item модель сохранения в бд
type Item struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"` // Доход/расход
	Amount   int       `json:"amount"`
	Date     time.Time `json:"date"`
	Category string    `json:"category"`
	Comment  string    `json:"comment"`
}

// Analytics для агрегированной аналитики
type Analytics struct {
	Category     string  `json:"category"`
	Sum          int     `json:"sum"`
	Avg          float64 `json:"avg"`
	Count        int     `json:"count"`
	Median       float64 `json:"median"`
	Percentile90 float64 `json:"percentile90"`
}

// Filter для получения данных из бд с фильтрами
type Filter struct {
	From      time.Time
	To        time.Time
	Type      string // Доход/расход
	Category  string
	AmountMin int
	AmountMax int
	Search    string
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
}
