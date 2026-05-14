package postgres

import (
	"L3.1/internal/entities"
	"context"
	"fmt"
	"github.com/wb-go/wbf/dbpg"
)

// Postgres обертка над wbf
type Postgres struct {
	db *dbpg.DB
}

// New создание экземпляра
func New(conn string) (*Postgres, error) {
	opts := &dbpg.Options{MaxOpenConns: 10, MaxIdleConns: 5}
	db, err := dbpg.New(conn, nil, opts)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Создаем таблицу если нужно
	err = CreateTable(db)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &Postgres{
		db: db,
	}, nil
}

// SaveNotification сохранить уведомление
func (p *Postgres) SaveNotification(ctx context.Context, n entities.Notification) error {
	query := "INSERT INTO notifications (id, send_at, notification, status) VALUES ($1, $2, $3, $4)"
	_, err := p.db.ExecContext(ctx, query, n.ID, n.SendAt, n.Event, n.Status)
	if err != nil {
		return fmt.Errorf("ошибка записи в бд %v", err)
	}

	return nil
}

// GetNotificationStatus получить статус уведомления
func (p *Postgres) GetNotificationStatus(ctx context.Context, id string) (string, error) {
	query := "SELECT status FROM notifications WHERE id = '" + id + "'"
	row := p.db.QueryRowContext(ctx, query)
	var status string
	err := row.Scan(&status)
	if err != nil {
		return "", fmt.Errorf("ошибка запроса статуса к бд %v", err)
	}

	return status, nil
}

// SetStatus поменять статус уведомления
func (p *Postgres) SetStatus(ctx context.Context, id string, status string) error {
	query := "UPDATE notifications SET status = '" + status + "' WHERE id = '" + id + "'"
	_, err := p.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса в бд %v", err)
	}

	return nil
}

// GetAllNotifications получить все уведомления
func (p *Postgres) GetAllNotifications(ctx context.Context) ([]entities.Notification, error) {
	query := "SELECT * FROM notifications"
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса к бд %v", err)
	}

	var result []entities.Notification

	for rows.Next() {
		var notification entities.Notification
		err := rows.Scan(&notification.ID, &notification.SendAt, &notification.Event, &notification.Status)
		if err != nil {
			return nil, fmt.Errorf("ошибка сбора уведомления из бд %v", err)
		}
		result = append(result, notification)
	}

	return result, nil
}

// GetAllIDs получить все ID
func (p *Postgres) GetAllIDs(ctx context.Context) ([]string, error) {
	var result []string

	// Получаем все ID
	rows, err := p.db.QueryContext(ctx, "SELECT id FROM notifications")
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса к бд %v", err)
	}

	for rows.Next() {
		var notificationID string
		err = rows.Scan(&notificationID)
		if err != nil {
			return nil, fmt.Errorf("ошибка записи в структуру уведомления %v", err)
		}

		result = append(result, notificationID)
	}

	return result, nil
}

// CreateTable создает таблицу в бд
func CreateTable(db *dbpg.DB) error {
	query := `CREATE TABLE IF NOT EXISTS notifications (
            id VARCHAR(16) PRIMARY KEY,
            send_at TIMESTAMP NOT NULL,
            notification TEXT NOT NULL,
            status VARCHAR(10) NOT NULL DEFAULT 'pending'
        )`

	_, err := db.ExecContext(context.Background(), query)
	if err != nil {
		return fmt.Errorf("не удалось создать таблицу %v", err)
	}

	return nil
}
