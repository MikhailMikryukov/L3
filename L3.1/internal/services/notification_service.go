package services

import (
	"L3.1/internal/entities"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
	"log"
	"sync"
	"time"
)

const (
	// StatusCanceled для бд статус отмененного уведомления
	StatusCanceled = "canceled"
	// StatusSent для бд статус отправленного уведомления
	StatusSent = "sent"
)

// NotificationRepository интерфейс обращения в бд
type NotificationRepository interface {
	SaveNotification(ctx context.Context, n entities.Notification) error
	GetNotificationStatus(ctx context.Context, id string) (string, error)
	SetStatus(ctx context.Context, id string, status string) error
	GetAllNotifications(ctx context.Context) ([]entities.Notification, error)
	GetAllIDs(ctx context.Context) ([]string, error)
}

// NotificationsService сервис работы с уведомлениями
type NotificationsService struct {
	rep       NotificationRepository
	publisher *rabbitmq.Publisher
	redis     *redis.Client
	mu        sync.Mutex
}

// New создание экземпляра
func New(rep NotificationRepository, publisher *rabbitmq.Publisher, redis *redis.Client) (*NotificationsService, error) {
	return &NotificationsService{rep: rep, publisher: publisher, redis: redis}, nil
}

// CreateNotification создает уведомление, сохраняет в бд и кеше, помещает ID в очередь
func (ns *NotificationsService) CreateNotification(dateStr string, event string) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return err
	}

	sendAt, err := time.ParseInLocation("2006-01-02 15:04:05", dateStr, location)
	if err != nil {
		return fmt.Errorf("некорректная дата")
	}

	notificationID := generateID()
	n := entities.Notification{
		ID:     notificationID,
		SendAt: sendAt,
		Event:  event,
		Status: "pending",
	}

	// Записываем уведомление в бд
	ctx := context.Background()
	err = ns.rep.SaveNotification(ctx, n)
	if err != nil {
		return err
	}

	// Кешируем уведомление
	notificationJSON, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("ошибка преобразования в JSON %v", err)
	}

	err = ns.redis.Set(ctx, notificationID, notificationJSON)
	if err != nil {
		return fmt.Errorf("ошибка кеширования %v", err)
	}

	// Отправляем ID уведомления в очередь
	return retry.Do(
		func() error {
			routingKey := "notifications.delayed"
			id := []byte(notificationID)

			ttl := sendAt.Sub(time.Now())
			err = ns.publisher.Publish(
				ctx,
				id,
				routingKey,
				rabbitmq.WithExpiration(ttl),
			)

			if err != nil {
				return fmt.Errorf("ошибка отправки в очередь %v", err)
			}

			log.Printf("Уведомление %s запланировано на %s (через %v)",
				notificationID,
				sendAt.Format("15:04:05"),
				ttl,
			)

			return nil
		}, retry.Strategy{
			Attempts: 3,
			Delay:    1 * time.Minute,
		})
}

// GetNotificationStatus получает статус уведомления
func (ns *NotificationsService) GetNotificationStatus(id string) (string, error) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	// Проверяем в кеше уведомление
	ctx := context.Background()
	cachedNotification, err := ns.redis.Get(ctx, id)
	if err == nil {
		var n entities.Notification
		err = json.Unmarshal([]byte(cachedNotification), &n)
		if err != nil {
			return "", fmt.Errorf("ошибка преобразования JSON из кеша %v", err)
		}
		return n.Status, nil
	}

	// Получаем статус из бд если нет в кеше
	status, err := ns.rep.GetNotificationStatus(ctx, id)
	if err != nil {
		return "", err
	}

	return status, nil
}

// DeleteNotification отменяет уведомление
func (ns *NotificationsService) DeleteNotification(id string) error {

	status, err := ns.GetNotificationStatus(id)
	if err != nil {
		return fmt.Errorf("ошибка получения статуса %v", err)
	}

	ns.mu.Lock()
	defer ns.mu.Unlock()

	// Меняем статус в бд и кеше
	if status == "pending" {
		// Меняем в бд
		ctx := context.Background()
		err = ns.rep.SetStatus(ctx, id, StatusCanceled)
		if err != nil {
			return err
		}

		// Проверяем в кеше
		cachedNotification, err := ns.redis.Get(ctx, id)
		if err == nil {
			var n entities.Notification
			err = json.Unmarshal([]byte(cachedNotification), &n)
			if err != nil {
				return fmt.Errorf("ошибка преобразования JSON из кеша %v", err)
			}

			// Меняем статус
			n.Status = "canceled"
			notificationJSON, err := json.Marshal(n)
			if err != nil {
				return fmt.Errorf("ошибка преобразования в JSON %v", err)
			}
			// Помещаем обратно в кеш
			err = ns.redis.Set(ctx, n.ID, notificationJSON)
			if err != nil {
				return fmt.Errorf("ошибка записи в кеш %v", err)
			}

			return nil
		}
	}

	log.Printf("Уведомление %s отменено", id)

	return nil
}

// SendNotification имитация отправки уведомления
func (ns *NotificationsService) SendNotification(n entities.Notification) error {
	ctx := context.Background()
	err := ns.rep.SetStatus(ctx, n.ID, StatusSent)
	if err != nil {
		return err
	}

	n.Status = "sent"
	notificationJSON, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("ошибка преобразования в JSON %v", err)
	}

	err = ns.redis.Set(ctx, n.ID, notificationJSON)
	if err != nil {
		return err
	}

	log.Printf("ID: %s, Notification: %s, Sent at: %s",
		n.ID,
		n.Event,
		n.SendAt.Format("2006-01-02 15:04:05"),
	)
	return nil
}

// RestoreCache помещает все уведомления из бд в кеш
func (ns *NotificationsService) RestoreCache() error {
	// Запрос в базу данных
	ctx := context.Background()
	notifications, err := ns.rep.GetAllNotifications(ctx)
	if err != nil {
		return err
	}

	// Кешируем
	for _, notification := range notifications {
		notificationJSON, err := json.Marshal(notification)
		if err != nil {
			return fmt.Errorf("ошибка преобразования в JSON %v", err)
		}

		// Помещаем в кеш
		err = ns.redis.Set(ctx, notification.ID, notificationJSON)
		if err != nil {
			return fmt.Errorf("ошибка записи в кеш %v", err)
		}
	}

	return nil
}

// GetAllNotifications функция для отображения всех уведомлений в браузере
func (ns *NotificationsService) GetAllNotifications() ([]string, error) {
	var result []string
	ctx := context.Background()

	// Получаем все ID
	IDs, err := ns.rep.GetAllIDs(ctx)
	if err != nil {
		return nil, err
	}

	for _, v := range IDs {
		// Достаем из кеша
		notificationCached, err := ns.redis.Get(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения из кеша %v", err)
		}

		result = append(result, notificationCached)
	}

	return result, nil
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
