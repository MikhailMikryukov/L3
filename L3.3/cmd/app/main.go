package main

import (
	"L3.3/internal/config"
	"L3.3/internal/handlers"
	"L3.3/internal/handlers/middleware"
	"L3.3/internal/repository/postgres"
	"L3.3/internal/services"
	"errors"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	//Подключение postgres
	var db services.CommentsRepository
	db, err := postgres.New(cfg.DBConnectionString)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Создание Comment service
	s := services.New(db)

	// Настройка и создание обработчика HTTP запросов
	router := handlers.NewRouter(s)
	loggedRouter := middleware.Logging(router)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: loggedRouter,
	}

	// Запускаем сервер на порту
	fmt.Println("Starting server at port ")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("Could not start server: %v\n", err)
	}
}
