package main

import (
	"L3.2/internal/config"
	"L3.2/internal/handlers"
	"L3.2/internal/handlers/middleware"
	"L3.2/internal/repository/postgres"
	"L3.2/internal/services"
	"errors"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	//Подключение к бд
	var db services.URLRepository
	db, err := postgres.New(cfg.DBConnectionString)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Создание URL service
	s := services.New(db, cfg.URLTemplate)

	// Ежедневно сохраняем данные за прошедший день
	s.StartDailyAggregation()

	// Настройка и создание обработчика HTTP запросов
	router := handlers.NewRouter(s)
	loggedRouter := middleware.Logging(router)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: loggedRouter,
	}

	// Запускаем сервер на порту
	fmt.Println("Starting server at port " + cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("Could not start server: %v\n", err)
	}
}
