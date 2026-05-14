package entities

import "time"

// Notification структура уведомлений
type Notification struct {
	ID     string
	SendAt time.Time
	Event  string
	Status string
}
