package main

import (
	"L3.7/internal/config"
	"L3.7/internal/handlers"
	"L3.7/internal/repository/postgres"
	"L3.7/internal/utils"
	"log"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Подключаемся к базе данных
	db, err := postgres.New(cfg.DBConnectionString)
	if err != nil {
		log.Fatal(err)
	}

	// Инициализируем JWT
	utils.InitJwt(cfg.JWTSecret)

	// Создаем обработчики
	authHandler := handlers.NewAuthHandler(db)
	historyHandler := handlers.NewHistoryHandler(db)
	itemHandler := handlers.NewItemHandler(db)

	router := handlers.NewRouter(authHandler, itemHandler, historyHandler)

	// Запускаем сервер
	err = router.Run(":" + cfg.Port)
	if err != nil {
		log.Fatal(err)
	}
}
