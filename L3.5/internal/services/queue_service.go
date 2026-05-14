package services

import (
	"L3.5/internal/entities"
	"L3.5/internal/repository/postgres"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
	"log"
	"time"
)

// QueueService работа с очередью
type QueueService struct {
	client    *rabbitmq.RabbitClient
	publisher *rabbitmq.Publisher
	consumer  *rabbitmq.Consumer
	db        *postgres.DB
}

type queueMsg struct {
	BookingID string `json:"booking_id"`
	UserName  string `json:"user_name"`
	EventID   string `json:"event_id"`
}

// NewQueueService создание QueueService
func NewQueueService(client *rabbitmq.RabbitClient, db *postgres.DB) (*QueueService, error) {
	// создание обменников и очередей
	err := setupRabbitMq(client)
	if err != nil {
		return nil, err
	}

	publisher := rabbitmq.NewPublisher(client, "booking-dlx", "application/json")
	// Настройка консюмера
	consumer, err := setupConsumer(client, db)
	if err != nil {
		return nil, err
	}
	return &QueueService{
		client:    client,
		publisher: publisher,
		consumer:  consumer,
		db:        db}, nil
}

// ScheduleCancellation отправляет бронь в очередь для отмены по истечению срока
func (s *QueueService) ScheduleCancellation(b entities.Booking) error {
	// Отправляем ID уведомления в очередь
	return retry.Do(
		func() error {

			ttl := b.PaymentDeadline.Sub(time.Now())

			// Проверяем, что TTL положительный
			if ttl <= 0 {
				// Если дедлайн уже прошел, отменяем сразу
				log.Printf("Бронь %s уже просрочена, отменяем немедленно", b.ID)
				tx, err := s.db.BeginTx()
				if err != nil {
					return err
				}
				defer tx.Rollback()

				// Проверяем статус
				status, err := s.db.GetBookingStatus(tx, b.ID)
				if err != nil {
					return err
				}

				if status == entities.BookingStatusBooked {
					// Отменяем бронь
					err = s.db.UpdateBookingStatus(tx, b.ID, entities.BookingStatusCancelled)
					if err != nil {
						return err
					}

					// Возвращаем место
					err = s.db.IncreaseAvailableSeats(tx, b.EventID, 1)
					if err != nil {
						return err
					}
				}

				return tx.Commit()
			}

			routingKey := "booking.waiting"

			msg := queueMsg{
				BookingID: b.ID,
				UserName:  b.UserName,
				EventID:   b.EventID,
			}

			ctx := context.Background()
			msgJSON, err := json.Marshal(msg)
			if err != nil {
				return err
			}
			// Отправляем сообщение
			err = s.publisher.Publish(
				ctx,
				msgJSON,
				routingKey,
				rabbitmq.WithExpiration(ttl),
			)

			if err != nil {
				return fmt.Errorf("ошибка отправки в очередь %v", err)
			}

			log.Printf("Бронь %s создана, оплатить до %s",
				b.ID,
				b.PaymentDeadline.Format("15:04:05"),
			)

			return nil
		}, retry.Strategy{
			Attempts: 3,
			Delay:    1 * time.Minute,
		})
}

// StartConsumer начинает обработку сообщений из очереди
func (s *QueueService) StartConsumer() {
	fmt.Println("Starting rabbit consumer")
	go func() {
		ctx := context.Background()
		if err := s.consumer.Start(ctx); err != nil {
			log.Fatalf("Ошибка при потреблении сообщений: %v", err)
		}
	}()
}

func setupConsumer(client *rabbitmq.RabbitClient, db *postgres.DB) (*rabbitmq.Consumer, error) {
	// Создание получателя сообщений
	// Параметры получателя
	consumerCfg := rabbitmq.ConsumerConfig{
		Queue: "booking-ready",
	}

	// Обработчик полученных сообщений
	handler := func(ctx context.Context, d amqp091.Delivery) error {
		log.Printf("Получено сообщение: %s", string(d.Body))
		// Получаем сообщение из очереди
		msgJSON := d.Body

		var msg queueMsg
		err := json.Unmarshal(msgJSON, &msg)

		tx, err := db.BeginTx()
		if err != nil {
			log.Printf("ошибка старта транзакции: %v", err)
			return err
		}
		defer tx.Rollback()

		// Проверяем статус брони
		status, err := db.GetBookingStatus(tx, msg.BookingID)
		if err != nil {
			log.Printf("ошибка получения статуса брони: %v", err)
			return err
		}

		// Если бронь оплачена или отменена - игнорируем
		if status != entities.BookingStatusBooked {
			// Завершаем транзакцию
			tx.Commit()
			return nil
		}
		// Меняем статус на отмену
		err = db.UpdateBookingStatus(tx, msg.BookingID, entities.BookingStatusCancelled)
		if err != nil {
			log.Printf("ошибка обновления статуса брони: %v", err)
			return err
		}

		// Возвращаем свободное место
		err = db.IncreaseAvailableSeats(tx, msg.EventID, 1)
		if err != nil {
			log.Printf("ошибка возврата свободного места: %v", err)
			return err
		}

		// Завершаем транзакцию
		tx.Commit()

		log.Printf("Бронь %s отменена по дедлайну", msg.BookingID)
		return nil
	}

	consumer := rabbitmq.NewConsumer(client, consumerCfg, handler)
	return consumer, nil
}

func setupRabbitMq(client *rabbitmq.RabbitClient) error {

	ch, err := client.GetChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Exchange куда посылаются сообщение по истечении ttl
	err = ch.ExchangeDeclare(
		"booking",
		"direct",
		true,
		false,
		false,
		false,
		nil)
	if err != nil {
		return fmt.Errorf("не удалось объявить exchange: %v", err)
	}

	// Exchange куда изначально посылаем сообщение
	err = ch.ExchangeDeclare(
		"booking-dlx",
		"direct",
		true,
		false,
		false,
		false,
		nil)
	if err != nil {
		return fmt.Errorf("не удалось объявить exchange: %v", err)
	}

	waitingQueueArgs := amqp091.Table{
		"x-dead-letter-exchange":    "booking",       // куда отправить после TTL
		"x-dead-letter-routing-key": "booking.ready", // с каким ключом
	}
	// Очередь, где сообщения лежат свой ttl
	_, err = ch.QueueDeclare(
		"booking-waiting",
		true,
		false,
		false,
		false,
		waitingQueueArgs)

	if err != nil {
		return fmt.Errorf("не удалось объявить queue: %v", err)
	}

	// Привязываем очередь ожидания к DLX exchange
	err = ch.QueueBind(
		"booking-waiting",
		"booking.waiting",
		"booking-dlx",
		false,
		nil)

	if err != nil {
		return fmt.Errorf("не удалось привязать queue: %v", err)
	}

	// Очередь, из которой берем сообщения на обработку
	_, err = ch.QueueDeclare(
		"booking-ready",
		true,
		false,
		false,
		false,
		nil)

	if err != nil {
		return fmt.Errorf("не удалось объявить queue: %v", err)
	}

	// Привязываем очередь готовых сообщений к основному exchange
	err = ch.QueueBind(
		"booking-ready",
		"booking.ready",
		"booking",
		false,
		nil)

	if err != nil {
		return fmt.Errorf("не удалось привязать queue: %v", err)
	}

	return nil
}
