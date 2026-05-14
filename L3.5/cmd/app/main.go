package main

import (
	"L3.5/internal/config"
	"L3.5/internal/handlers"
	"L3.5/internal/handlers/middleware"
	"L3.5/internal/repository/postgres"
	"L3.5/internal/services"
	"errors"
	"fmt"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
	"log"
	"net/http"
	"time"
)

func main() {
	cfg := config.LoadConfig()

	// Подключение к бд
	db, err := postgres.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	// Настройка и создание клиента RabbitMQ
	strategy := retry.Strategy{
		Attempts: 3,
		Delay:    3 * time.Second,
		Backoff:  2,
	}
	rabbitCfg := rabbitmq.ClientConfig{
		URL:            cfg.RabbitURL,
		ConnectionName: cfg.RabbitConnectionName,
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

	// Создание сервиса очереди
	queueService, err := services.NewQueueService(rabbitClient, db)
	if err != nil {
		log.Fatal(err)
	}

	// Начинаем обработку сообщений из очереди
	queueService.StartConsumer()

	// Настройка и создание обработчика HTTP запросов
	router := handlers.NewRouter(db, queueService)
	loggedRouter := middleware.Logging(router)

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: loggedRouter,
	}

	// Запускаем сервер на порту
	fmt.Println("Starting server at port", cfg.ServerPort)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("Could not start server: %v\n", err)
	}
}
