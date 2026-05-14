package entities

import "time"

// URLData информация о ссылке
type URLData struct {
	UserAgent string
	Date      time.Time
	IP        string
}
