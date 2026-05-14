package entities

import "time"

// Comment структура комментария
type Comment struct {
	ID        int
	Author    string
	Parent    int
	Text      string
	CreatedAt time.Time
}
