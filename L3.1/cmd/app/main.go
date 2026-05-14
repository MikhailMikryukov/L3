package main

import (
	"L3.1/internal/config"
	"L3.1/internal/entities"
	"L3.1/internal/handlers"
	"L3.1/internal/handlers/middleware"
	"L3.1/internal/queue"
	"L3.1/internal/repository/postgres"
	"L3.1/internal/services"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
	"log"
	"net/http"
	"time"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	//Подключение к бд
	var repository services.NotificationRepository
	repository, err := postgres.New(cfg.DBConnectionString)
	if err != nil {
		log.Fatalf(err.Error())
	}

	// Создание клиента Redis
	redisClient := redis.New(cfg.RedisAddr, cfg.RedisPassword, 0)

	// Настройка и создание клиента RabbitMQ
	strategy := retry.Strategy{
		Attempts: 3,
		Delay:    3 * time.Second,
		Backoff:  2,
	}
	rabbitCfg := rabbitmq.ClientConfig{
		URL:            cfg.RabbitAddress,
		ConnectionName: "my-services",
		ConnectTimeout: 5 * time.Second,
		Heartbeat:      10 * time.Second,
		ProducingStrat: strategy,
		ConsumingStrat: strategy,
	}

	rabbitClient, err := rabbitmq.NewClient(rabbitCfg)
	if err != nil {
		log.Fatalf("Ошибка подключения к RabbitMQ: %v", err)
	}
	defer rabbitClient.Close()

	err = queue.SetupRabbitMq(rabbitClient)
	if err != nil {
		fmt.Println(err)
	}

	// Создание отправителя RabbitMQ
	publisher := rabbitmq.NewPublisher(rabbitClient, "notifications-dlx", "application/json")

	// Создание notification service
	ns, err := services.New(repository, publisher, redisClient)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Уведомления из бд в кеш
	err = ns.RestoreCache()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Создание получателя сообщений
	// Параметры получателя
	consumerCfg := rabbitmq.ConsumerConfig{
		Queue: "notifications-ready",
	}

	// Обработчик полученных сообщений
	handler := func(ctx context.Context, d amqp091.Delivery) error {
		log.Printf("Получено сообщение: %s", string(d.Body))
		// Получаем ID из очереди
		id := string(d.Body)

		// Проверяем статус уведомления
		status, err := ns.GetNotificationStatus(id)
		if err != nil {
			log.Printf(err.Error())
			return err
		}

		// Если уведомление отменено - игнорируем
		if status == "canceled" {
			return nil // ACK
		}

		// Достаем из кеша уведомление
		var notification entities.Notification
		nCached, err := redisClient.Get(ctx, id)
		if err != nil {
			log.Printf(err.Error())
			return err
		}

		err = json.Unmarshal([]byte(nCached), &notification)
		if err != nil {
			return err
		}

		// Отправляем уведомления с экспоненциальной задержкой
		// Имитация (логирование)
		return retry.Do(
			func() error {
				err = ns.SendNotification(notification)
				if err != nil {
					log.Printf(err.Error())
					return err
				}

				return nil // ACK
			}, retry.Strategy{
				Attempts: 4,
				Delay:    1 * time.Minute,
				Backoff:  3,
			})

	}

	consumer := rabbitmq.NewConsumer(rabbitClient, consumerCfg, handler)

	// Старт получателя сообщений
	go func() {
		if err := consumer.Start(context.Background()); err != nil {
			log.Fatalf("Ошибка при потреблении сообщений: %v", err)
		}
	}()

	// Настройка и создание обработчика HTTP запросов
	router := handlers.NewRouter(ns)
	loggedRouter := middleware.Logging(router)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: loggedRouter,
	}

	// Запускаем сервер на порту
	fmt.Println("Starting server at port " + cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Could not start server: %v\n", err)
	}

}
