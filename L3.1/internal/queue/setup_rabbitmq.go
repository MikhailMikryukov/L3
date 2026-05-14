package queue

import (
	"fmt"
	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/rabbitmq"
)

// SetupRabbitMq создание нужных exchange и очередей
func SetupRabbitMq(client *rabbitmq.RabbitClient) error {

	ch, err := client.GetChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Exchange для готовых
	err = ch.ExchangeDeclare(
		"notifications",
		"direct",
		true,
		false,
		false,
		false,
		nil)

	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}
	// Exchange для отложенных
	err = ch.ExchangeDeclare(
		"notifications-dlx",
		"direct",
		true,
		false,
		false,
		false,
		nil)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Очередь, где сообщения ждут
	waitingQueueArgs := amqp091.Table{
		"x-dead-letter-exchange":    "notifications",       // куда отправить после TTL
		"x-dead-letter-routing-key": "notifications.ready", // с каким ключом
	}

	_, err = ch.QueueDeclare(
		"notifications-waiting", // имя очереди
		true,                    // durable
		false,                   // autoDelete
		false,                   // exclusive (только для этого соединения)
		false,                   // noWait
		waitingQueueArgs,        // аргументы
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Привязываем очередь с exchange ждущих сообщений
	err = ch.QueueBind(
		"notifications-waiting",
		"notifications.delayed",
		"notifications-dlx",
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Очередь для готовых к отправке сообщений
	_, err = ch.QueueDeclare(
		"notifications-ready",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Привязываем очередь готовых к основному exchange
	err = ch.QueueBind(
		"notifications-ready",
		"notifications.ready", // binding key
		"notifications",
		false,
		nil,
	)

	return nil
}
