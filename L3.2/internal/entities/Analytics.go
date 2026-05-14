package entities

import "time"

// Analytics сборная аналитика
type Analytics struct {
	StatDate   time.Time     `json:"stat_date"`
	Summary    SummaryClicks `json:"summary_clicks"`
	UserAgents []UserAgent   `json:"user_agents_stat"`
	ClicksTime ClicksTime    `json:"clicks_time"`
}

// SummaryClicks информация о количестве переходов
type SummaryClicks struct {
	Total     int `json:"total"`
	Today     int `json:"today"`
	LastWeek  int `json:"last_week"`
	LastMonth int `json:"last_month"`
}

// UserAgent информация с каких user-agent были переходы
type UserAgent struct {
	Name   string `json:"user_agent"`
	Clicks int    `json:"clicks"`
}

// ClicksTime информация в какие часы были переходы
type ClicksTime struct {
	NightHours   int `json:"night"`
	DayHours     int `json:"day"`
	EveningHours int `json:"evening"`
}
