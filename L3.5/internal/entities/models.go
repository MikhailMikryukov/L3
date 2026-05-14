package entities

import "time"

var (
	// EventStatusActive статус события активно
	EventStatusActive = "active"
	// EventStatusEnded статус события закончено
	EventStatusEnded = "ended"
	// EventStatusCancelled статус события отменено
	EventStatusCancelled = "cancelled"
	// BookingStatusBooked статус брони забронировано
	BookingStatusBooked = "booked"
	// BookingStatusPaid статус брони оплачено
	BookingStatusPaid = "paid"
	// BookingStatusCancelled статус брони отменено
	BookingStatusCancelled = "cancelled"
	// PaymentStatusSuccess статус оплаты успешно
	PaymentStatusSuccess = "success"
	// PaymentStatusProcessing статус оплаты в процессе
	PaymentStatusProcessing = "processing"
	// PaymentStatusError статус оплаты ошибка
	PaymentStatusError = "error"
)

// Event событие
type Event struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Date                   time.Time `json:"event_date"`
	TotalSeats             int       `json:"total_seats"`
	AvailableSeats         int       `json:"available_seats"`
	Price                  int       `json:"price"`
	BookingDeadlineMinutes int       `json:"booking_deadline_minutes"`
	Status                 string    `json:"status"`
}

// Booking бронь события
type Booking struct {
	ID              string    `json:"id"`
	UserName        string    `json:"user_name"`
	EventID         string    `json:"event_id"`
	EventName       string    `json:"event_name"`
	Status          string    `json:"status"`
	Price           int       `json:"price"`
	CreatedAt       time.Time `json:"created_at"`
	PaymentDeadline time.Time `json:"payment_deadline"`
	PaidAt          time.Time `json:"paid_at"`
}

// Payment платеж брони
type Payment struct {
	ID          string    `json:"id"`
	BookingID   string    `json:"booking_id"`
	Amount      int       `json:"amount"`
	Status      string    `json:"status"`
	ProcessedAt time.Time `json:"processed_at"`
}
