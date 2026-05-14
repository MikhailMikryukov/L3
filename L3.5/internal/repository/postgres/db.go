package postgres

import (
	"L3.5/internal/entities"
	"context"
	"database/sql"
	"fmt"
	"github.com/wb-go/wbf/dbpg"
	"log"
	"time"
)

// DB обертка над DB wbf
type DB struct {
	db  *dbpg.DB
}

// New создание экземпляра
func New(conn string) (*DB, error) {
	opts := &dbpg.Options{MaxOpenConns: 10, MaxIdleConns: 5}
	db, err := dbpg.New(conn, nil, opts)
	if err != nil {
		return nil, err
	}

	// Создаем таблицу если нужно
	err = createTable(db)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &DB{
		db:  db,
	}, nil
}

// BeginTx начинает транзакцию
func (d *DB) BeginTx() (*sql.Tx, error) {
	return d.db.BeginTx(context.Background(), &sql.TxOptions{})
}

// CreateEvent записывает информацию о событии в бд
func (d *DB) CreateEvent(e entities.Event) error {
	ctx := context.Background()
	query := `INSERT INTO events (id, event_date, event_name, total_seats, price, available_seats, booking_deadline_minutes) VALUES ($1,$2,$3,$4, $5, $6, $7)`

	_, err := d.db.ExecContext(ctx, query, e.ID, e.Date, e.Name, e.TotalSeats, e.Price, e.AvailableSeats, e.BookingDeadlineMinutes)

	return err
}

// GetEventForUpdate получает событие по ID из бд для обновления данных
func (d *DB) GetEventForUpdate(tx *sql.Tx, eventID string) (*entities.Event, error) {
	ctx := context.Background()
	query := `SELECT * FROM events WHERE id = $1 FOR UPDATE`
	var row *sql.Row
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, eventID)
	} else {
		row = d.db.QueryRowContext(ctx, query, eventID)
	}

	var e entities.Event
	err := row.Scan(&e.ID, &e.Name, &e.Date, &e.TotalSeats, &e.AvailableSeats, &e.Price, &e.BookingDeadlineMinutes, &e.Status)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

// GetAllEvents получает все события из бд
func (d *DB) GetAllEvents(tx *sql.Tx) ([]entities.Event, error) {
	ctx := context.Background()
	query := `SELECT * FROM events`
	var rows *sql.Rows
	var err error
	if tx != nil {
		rows, err = tx.QueryContext(ctx, query)
	} else {
		rows, err = d.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, err
	}

	var events []entities.Event
	for rows.Next() {
		var e entities.Event
		err = rows.Scan(&e.ID, &e.Name, &e.Date, &e.TotalSeats, &e.AvailableSeats, &e.Price, &e.BookingDeadlineMinutes, &e.Status)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil
}

// CreateBooking записывает информацию о брони в бд
func (d *DB) CreateBooking(tx *sql.Tx, b entities.Booking) error {
	ctx := context.Background()
	query := `INSERT INTO bookings (id, user_name, event_id, event_name, status, price, created_at, payment_deadline) VALUES ($1,$2,$3,$4, $5, $6, $7, $8)`
	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, b.ID, b.UserName, b.EventID, b.EventName, b.Status, b.Price, b.CreatedAt, b.PaymentDeadline)
	} else {
		_, err = d.db.ExecContext(ctx, query, b.ID, b.UserName, b.EventID, b.EventName, b.Status, b.Price, b.CreatedAt, b.PaymentDeadline)
	}
	if err != nil {
		return fmt.Errorf("ошибка создания брони %s", err)
	}
	return nil
}

// CreatePayment записывает информацию о платеже в бд
func (d *DB) CreatePayment(tx *sql.Tx, p entities.Payment) error {
	ctx := context.Background()
	query := `INSERT INTO payments (id, booking_id, amount, status, processed_at) VALUES ($1,$2,$3,$4,$5)`
	_, err := tx.ExecContext(ctx, query, p.ID, p.BookingID, p.Amount, p.Status, p.ProcessedAt)
	if err != nil {
		return fmt.Errorf("ошибка создания оплаты")
	}

	updateBookingQuery := `UPDATE bookings SET status = $1, paid_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err = tx.ExecContext(ctx, updateBookingQuery, entities.BookingStatusPaid, p.BookingID)
	if err != nil {
		return fmt.Errorf("ошибка обновления оплаты брони")
	}
	return nil
}

// GetEventByID получает событие по ID из бд
func (d *DB) GetEventByID(id string) (*entities.Event, error) {
	ctx := context.Background()
	query := `SELECT * FROM events WHERE id = $1`
	row := d.db.QueryRowContext(ctx, query, id)

	var e entities.Event
	err := row.Scan(&e.ID, &e.Name, &e.Date, &e.TotalSeats, &e.AvailableSeats, &e.Price, &e.BookingDeadlineMinutes, &e.Status)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

// GetAllBookingsByUserName получает все брони из бд
func (d *DB) GetAllBookingsByUserName(tx *sql.Tx, userName string) ([]entities.Booking, error) {
	query := `SELECT * FROM bookings WHERE user_name = $1`

	var rows *sql.Rows
	var err error
	ctx := context.Background()
	if tx != nil {
		rows, err = tx.QueryContext(ctx, query, userName)
	} else {
		rows, err = d.db.QueryContext(ctx, query, userName)
	}
	if err != nil {
		return nil, err
	}

	var bookings []entities.Booking
	for rows.Next() {
		var b entities.Booking
		var paidAt sql.NullTime

		err = rows.Scan(&b.ID, &b.UserName, &b.EventID, &b.EventName, &b.Status, &b.Price, &b.CreatedAt, &b.PaymentDeadline, &paidAt)
		if err != nil {
			return nil, err
		}
		if paidAt.Valid {
			b.PaidAt = paidAt.Time
		} else {
			b.PaidAt = time.Time{}
		}

		bookings = append(bookings, b)
	}

	return bookings, nil
}

// GetAllBookings получает все брони из бд
func (d *DB) GetAllBookings(tx *sql.Tx) ([]entities.Booking, error) {
	query := `SELECT * FROM bookings`

	var rows *sql.Rows
	var err error
	ctx := context.Background()
	if tx != nil {
		rows, err = tx.QueryContext(ctx, query)
	} else {
		rows, err = d.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, err
	}

	var bookings []entities.Booking
	for rows.Next() {
		var b entities.Booking
		var paidAt sql.NullTime

		err = rows.Scan(&b.ID, &b.UserName, &b.EventID, &b.EventName, &b.Status, &b.Price, &b.CreatedAt, &b.PaymentDeadline, &paidAt)
		if err != nil {
			return nil, err
		}
		if paidAt.Valid {
			b.PaidAt = paidAt.Time
		} else {
			b.PaidAt = time.Time{}
		}

		bookings = append(bookings, b)
	}

	return bookings, nil
}

// GetBookingByID получает бронь из бд по ее ID
func (d *DB) GetBookingByID(tx *sql.Tx, id string) (*entities.Booking, error) {
	query := `SELECT * FROM bookings WHERE id = $1`
	var row *sql.Row
	ctx := context.Background()
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, id)
	} else {
		row = d.db.QueryRowContext(ctx, query, id)
	}
	var paidAt sql.NullTime
	var b entities.Booking
	err := row.Scan(&b.ID, &b.UserName, &b.EventID, &b.EventName, &b.Status, &b.Price, &b.CreatedAt, &b.PaymentDeadline, &paidAt)
	if err != nil {
		return nil, err
	}
	if paidAt.Valid {
		b.PaidAt = paidAt.Time
	} else {
		b.PaidAt = time.Time{}
	}
	return &b, nil
}

// GetBookingByEventIDAndUserName получает бронь из бд по ID события и имени пользователя
func (d *DB) GetBookingByEventIDAndUserName(tx *sql.Tx, id string, userName string) (*entities.Booking, error) {
	query := `SELECT * FROM bookings WHERE event_id = $1 AND user_name = $2`
	var row *sql.Row
	ctx := context.Background()
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, id, userName)
	} else {
		row = d.db.QueryRowContext(ctx, query, id, userName)
	}
	var paidAt sql.NullTime
	var b entities.Booking
	err := row.Scan(&b.ID, &b.UserName, &b.EventID, &b.EventName, &b.Status, &b.Price, &b.CreatedAt, &b.PaymentDeadline, &paidAt)
	if err != nil {
		return nil, err
	}
	if paidAt.Valid {
		b.PaidAt = paidAt.Time
	} else {
		b.PaidAt = time.Time{}
	}
	return &b, nil
}

// GetBookingStatus получает статус брони в бд
func (d *DB) GetBookingStatus(tx *sql.Tx, id string) (string, error) {
	query := `SELECT status FROM bookings WHERE id = $1`
	var row *sql.Row
	ctx := context.Background()
	if tx != nil {
		row = tx.QueryRowContext(ctx, query, id)
	} else {
		row = d.db.QueryRowContext(ctx, query, id)
	}

	var status string
	err := row.Scan(&status)
	if err != nil {
		return "", err
	}

	return status, nil
}

// UpdateBookingStatus обновляет статус брони в бд
func (d *DB) UpdateBookingStatus(tx *sql.Tx, id string, status string) error {
	query := `UPDATE bookings SET status = $1 WHERE id = $2`
	var err error
	ctx := context.Background()
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, status, id)
	} else {
		_, err = d.db.ExecContext(ctx, query, status, id)
	}
	return err
}

// DecreaseAvailableSeats уменьшает кол-во свободных мест события
func (d *DB) DecreaseAvailableSeats(tx *sql.Tx, eventID string, amount int) error {
	query := `UPDATE events SET available_seats = available_seats - $1 WHERE id = $2`
	var err error
	ctx := context.Background()
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, amount, eventID)
	} else {
		_, err = d.db.ExecContext(ctx, query, amount, eventID)
	}
	return err
}

// IncreaseAvailableSeats увеличивает кол-во свободных мест события
func (d *DB) IncreaseAvailableSeats(tx *sql.Tx, eventID string, amount int) error {
	query := `UPDATE events SET available_seats = available_seats + $1 WHERE id = $2`
	var err error
	ctx := context.Background()
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, amount, eventID)
	} else {
		_, err = d.db.ExecContext(ctx, query, amount, eventID)
	}
	return err
}

// DeleteEvent удаляет событие из бд
func (d *DB) DeleteEvent(tx *sql.Tx, eventID string) error {
	_, err := tx.Exec("DELETE FROM events WHERE id = $1", eventID)
	if err != nil {
		return fmt.Errorf("ошибка при удалении: %w", err)
	}
	return err
}

func createTable(db *dbpg.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS events (
    		id VARCHAR(36) PRIMARY KEY,
    		event_name TEXT NOT NULL,
    		event_date TIMESTAMPTZ NOT NULL ,
			total_seats int,
    		available_seats int,
    		price int NOT NULL,
    		booking_deadline_minutes int,
            status VARCHAR(9) NOT NULL DEFAULT 'active'
    	)`,
		`CREATE TABLE IF NOT EXISTS bookings (
    		id VARCHAR(36) PRIMARY KEY,
    		user_name TEXT NOT NULL,
    		event_id VARCHAR(36) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    		event_name TEXT NOT NULL,
    		status VARCHAR(9) NOT NULL DEFAULT 'booked',
    		price int NOT NULL,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    		payment_deadline TIMESTAMPTZ NOT NULL,
    		paid_at TIMESTAMPTZ
    	)`,
		`CREATE TABLE IF NOT EXISTS payments (
    		id VARCHAR(36) PRIMARY KEY,
    		booking_id VARCHAR(36) NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    		amount DECIMAL,
			status VARCHAR(10) NOT NULL DEFAULT 'processing',
    		processed_at TIMESTAMPTZ
    	)`,
	}

	for _, q := range queries {
		_, err := db.ExecContext(context.Background(), q)
		if err != nil {
			return err
		}
	}

	return nil
}
