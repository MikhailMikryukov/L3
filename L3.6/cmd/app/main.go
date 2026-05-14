package main

import (
	"L3.6/internal/config"
	"L3.6/internal/handlers"
	"L3.6/internal/handlers/middleware"
	"L3.6/internal/repository/postgres"
	"errors"
	"fmt"
	"log"
	"net/http"
)

func main() {
	cfg := config.LoadConfig()
	

	// Подключение к бд
	db, err := postgres.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	// Настройка и создание обработчика HTTP запросов
	router := handlers.NewRouter(db)
	loggedRouter := middleware.Logging(router)

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: loggedRouter,
	}

	// Запускаем сервер на порту
	fmt.Println("Starting server at port ", cfg.ServerPort)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("Could not start server: %v\n", err)
	}
}
