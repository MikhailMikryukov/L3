package handlers

import (
	"L3.5/internal/entities"
	"L3.5/internal/repository/postgres"
	"L3.5/internal/services"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Response ответ в веб-интерфейс
type Response struct {
	Status   string `json:"status"`
	Response string `json:"response"`
}

// EventHandler обработчик запросов
type EventHandler struct {
	db  *postgres.DB
	qs  *services.QueueService
}

// New создает экземпляр EventHandler
func New(db *postgres.DB, qs *services.QueueService) *EventHandler {
	return &EventHandler{
		qs:  qs,
		db:  db,
	}
}

// NewRouter создание и настройка обработчика
func NewRouter(db *postgres.DB, qs *services.QueueService) *http.ServeMux {
	mux := http.NewServeMux()
	eventHandler := New( db, qs)

	// Страница для проверки работы приложения
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	})
	// Страница для проверки работы приложения
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/admin.html")
	})

	// Создание события
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		eventHandler.createEvent(w, r)
	})

	mux.HandleFunc("/events/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/events/")

		if r.Method == "GET" {
			// Получение всех событий
			if strings.HasPrefix(path, "all") {
				eventHandler.getAllEvents(w, r)
				return
			}
			// Получение информации о событии
			eventHandler.getEvent(w, r, path)
			return
		}

		if r.Method == "POST" {
			// Бронирование
			if strings.HasSuffix(path, "/book") {
				id := path[:strings.Index(path, "/")]
				eventHandler.book(w, r, id)
				return
			}

			// Подтверждение брони
			if strings.HasSuffix(path, "/confirm") {
				id := path[:strings.Index(path, "/")]
				eventHandler.confirm(w, r, id)
				return
			}
		}

		if r.Method != "DELETE" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Удаление события
		eventHandler.deleteEvent(w, r, path)

		return
	})

	// Получение всех броней пользователя
	mux.HandleFunc("/user_bookings", eventHandler.userBookings)

	// Получение всех броней
	mux.HandleFunc("/bookings/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/bookings/")

		if r.Method == "GET" {
			// Получение всех броней
			if strings.HasPrefix(path, "all") {
				eventHandler.bookings(w, r)
				return
			}

			return
		}

		// Отмена брони
		if r.Method == "POST" {
			if strings.HasSuffix(path, "cancel") {
				id := path[:strings.Index(path, "/")]
				eventHandler.cancelBooking(w, r, id)
			}
		}
	})

	return mux
}

func (h *EventHandler) createEvent(w http.ResponseWriter, r *http.Request) {
	eventDate := r.FormValue("date")
	eventName := r.FormValue("name")
	seatsStr := r.FormValue("seats")
	deadlineStr := r.FormValue("deadline")
	priceStr := r.FormValue("price")

	if eventDate == "" || eventName == "" || seatsStr == "" || deadlineStr == "" || priceStr == "" {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: "укажите все данные"})
		return
	}

	seats, err := strconv.Atoi(seatsStr)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: "некорректное значение количества мест"})
		return
	}

	price, err := strconv.Atoi(priceStr)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: "некорректное значение цены"})
		return
	}

	date, err := time.Parse("2006-01-02T15:04", eventDate)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: "некорректное значение даты события. формат - ГГГГ-ММ-ДДТчч:мм"})
		return
	}

	deadline, err := strconv.Atoi(deadlineStr)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: "некорректное значение срока оплаты брони"})
		return
	}

	id := uuid.New().String()

	event := entities.Event{
		ID:                     id,
		Name:                   eventName,
		Date:                   date,
		TotalSeats:             seats,
		AvailableSeats:         seats,
		Price:                  price,
		BookingDeadlineMinutes: deadline,
		Status:                 entities.EventStatusActive,
	}

	err = h.db.CreateEvent(event)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка создания события %s", err)})
		return
	}

	h.writeJSON(w, http.StatusOK, Response{
		Status:   "success",
		Response: event.ID,
	})
}

func (h *EventHandler) getEvent(w http.ResponseWriter, r *http.Request, id string) {

	event, err := h.db.GetEventByID(id)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("событие не найдено, id: %s", id)})
		return
	}
	h.writeJSON(w, http.StatusOK, event)
}

func (h *EventHandler) book(w http.ResponseWriter, r *http.Request, eventID string) {
	userName := r.FormValue("userName")

	if userName == "" {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("не указано имя пользователя ")})
		return
	}

	// Начинаем транзакцию
	tx, err := h.db.BeginTx()
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка начала транзакции %s", err)})
		return
	}
	defer tx.Rollback()

	// Получаем событие
	event, err := h.db.GetEventForUpdate(tx, eventID)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка получения события %s", err)})
		return
	}

	// Проверяем есть ли свободные места
	if event.AvailableSeats <= 0 {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("нет свободных мест")})
		return
	}

	// Убавляем свободные места
	err = h.db.DecreaseAvailableSeats(tx, event.ID, 1)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка уменьшения свободных мест %s", err)})
		return
	}

	// Создаем бронь
	paymentDeadline := time.Now().Add(time.Duration(event.BookingDeadlineMinutes) * time.Minute)

	if paymentDeadline.After(event.Date) {
		paymentDeadline = event.Date
	}

	bookingID := uuid.New().String()
	booking := entities.Booking{
		ID:              bookingID,
		UserName:        userName,
		EventID:         event.ID,
		EventName:       event.Name,
		Status:          entities.BookingStatusBooked,
		Price:           event.Price,
		CreatedAt:       time.Now(),
		PaymentDeadline: paymentDeadline,
	}

	// Записываем в бд бронь
	err = h.db.CreateBooking(tx, booking)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка бронирования %s", err)})
		return
	}

	// Завершаем транзакцию
	err = tx.Commit()
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка завершения транзакции %s", err)})
		return
	}

	// Отправляем бронь в очередь
	err = h.qs.ScheduleCancellation(booking)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка отправки брони в очередь %s", err)})
		return
	}

	h.writeJSON(w, http.StatusOK, booking)
}

func (h *EventHandler) confirm(w http.ResponseWriter, r *http.Request, eventID string) {
	userName := r.FormValue("userName")
	bookingId := r.FormValue("booking_id")

	if userName == "" {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("не указано имя пользователя ")})
		return
	}

	// Начинаем транзакцию
	tx, err := h.db.BeginTx()
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка бронирования %s", err)})
		return
	}
	defer tx.Rollback()

	// Получаем бронь
	booking, err := h.db.GetBookingByID(tx, bookingId)
	if err != nil {
		h.writeJSON(w, http.StatusNotFound, Response{Status: "error", Response: fmt.Sprintf("нет брони на данное событие и имя пользователя %s", err.Error())})
		return
	}

	// Проверяем нужно ли оплачивать бронь
	if booking.Price <= 0 {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("событие не нужно оплачивать, id: %s", eventID)})
		return
	}

	// Проверяем статус брони
	switch booking.Status {
	// Оплачено
	case entities.BookingStatusPaid:
		h.writeJSON(w, http.StatusBadRequest, Response{Status: "error", Response: "бронь уже оплачена"})
		return
	// Отменено
	case entities.BookingStatusCancelled:
		h.writeJSON(w, http.StatusBadRequest, Response{Status: "error", Response: "бронь отменена"})
		return
	// Забронировано
	case entities.BookingStatusBooked:
		// Если время больше чем дедлайн
		if time.Now().After(booking.PaymentDeadline) {
			// Отменяем бронь
			err = h.db.UpdateBookingStatus(tx, booking.ID, entities.BookingStatusCancelled)
			if err != nil {
				log.Println(err)
				return
			}
			// Возвращаем место
			err = h.db.IncreaseAvailableSeats(tx, booking.ID, 1)
			if err != nil {
				log.Println(err)
			}
			h.writeJSON(w, http.StatusBadRequest, Response{Status: "error", Response: "срок оплаты истек"})
			return
		}
	default:
		h.writeJSON(w, http.StatusBadRequest, Response{Status: "error", Response: "неизвестный статус брони"})
		return
	}

	// Имитация оплаты
	payment, err := processPayment(booking.ID, booking.Price)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: "ошибка обработки платежа"})
		return
	}

	// Записываем оплату в бд
	err = h.db.CreatePayment(tx, payment)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: "ошибка сохранения платежа"})
		return
	}

	// Завершаем транзакцию
	err = tx.Commit()
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка бронирования %s", err)})
		return
	}

	h.writeJSON(w, http.StatusOK, payment)
}

func processPayment(bookingID string, amount int) (entities.Payment, error) {
	time.Sleep(3 * time.Second)

	paymentID := uuid.New().String()
	p := entities.Payment{
		ID:          paymentID,
		BookingID:   bookingID,
		Amount:      amount,
		Status:      entities.PaymentStatusSuccess,
		ProcessedAt: time.Now(),
	}

	return p, nil
}

func (h *EventHandler) getAllEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.db.GetAllEvents(nil)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка получения событий %s", err)})
		return
	}

	h.writeJSON(w, http.StatusOK, events)
}

func (h *EventHandler) userBookings(w http.ResponseWriter, r *http.Request) {
	userName := r.FormValue("userName")
	if userName == "" {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("укажите имя пользователя")})
		return
	}

	bookings, err := h.db.GetAllBookingsByUserName(nil, userName)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка получения броней %s", err)})
		return
	}

	h.writeJSON(w, http.StatusOK, bookings)
}

func (h *EventHandler) bookings(w http.ResponseWriter, r *http.Request) {
	bookings, err := h.db.GetAllBookings(nil)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка получения броней %s", err)})
		return
	}

	h.writeJSON(w, http.StatusOK, bookings)
}

func (h *EventHandler) cancelBooking(w http.ResponseWriter, r *http.Request, id string) {
	// Начинаем транзакцию
	tx, err := h.db.BeginTx()
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка начала транзакции %s", err)})
		return
	}
	defer tx.Rollback()

	// Получаем бронь
	booking, err := h.db.GetBookingByID(tx, id)
	if err != nil {
		h.writeJSON(w, http.StatusNotFound, Response{Status: "error", Response: fmt.Sprintf("ошибка получения брони %s", err)})
	}

	// Если уже отменена
	if booking.Status == entities.BookingStatusCancelled {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("бронь уже отменена %s", err)})
		return
	}

	// Обновляем статус
	err = h.db.UpdateBookingStatus(tx, id, entities.BookingStatusCancelled)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка отмена брони %s", id)})
		return
	}
	// Возвращаем свободное место
	err = h.db.IncreaseAvailableSeats(tx, booking.EventID, 1)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка возврата свободных мест %s", id)})
		return
	}

	// Закрываем транзакцию
	tx.Commit()

	w.WriteHeader(http.StatusOK)
}

func (h *EventHandler) deleteEvent(w http.ResponseWriter, r *http.Request, id string) {
	// Начинаем транзакцию
	tx, err := h.db.BeginTx()
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка начала транзакции %s", err)})
		return
	}
	defer tx.Rollback()

	// Удаляем событие
	err = h.db.DeleteEvent(tx, id)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, Response{Status: "error", Response: fmt.Sprintf("ошибка удаления события %s", err)})
		return
	}

	tx.Commit()
	w.WriteHeader(http.StatusOK)
}

func (h *EventHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
