package main

import (
	"L3.4/internal/config"
	"L3.4/internal/handlers"
	"L3.4/internal/handlers/middleware"
	"L3.4/internal/repository/postgres"
	"L3.4/internal/services"
	"L3.4/internal/workers"
	"L3.4/pkg/messaging"
	"errors"
	"fmt"
	"github.com/wb-go/wbf/logger"
	"log"
	"net/http"
)

func main() {
	cfg := config.LoadConfig()

	// Подключение к бд
	var db services.TasksRepository
	db, err := postgres.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	kafkaLog, err := logger.InitLogger(logger.ZerologEngine, cfg.LoggerAppName, "")
	if err != nil {
		log.Fatal(err)
	}
	// Создание консюмера, продюсера кафка
	kafka := messaging.NewWBKafka([]string{cfg.KafkaBroker}, cfg.KafkaTopic, kafkaLog, "0")

	// Создаем сервисы
	storage := services.NewStorageService(cfg)
	processor := services.NewImageProcessor(db, storage)

	// Создаем воркер
	worker := workers.NewImageWorker(kafka, db, processor, storage)
	// Старт воркера
	go func() {
		err := worker.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()
	// Настройка и создание обработчика HTTP запросов
	router := handlers.NewRouter(processor, db, storage, kafka)
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
