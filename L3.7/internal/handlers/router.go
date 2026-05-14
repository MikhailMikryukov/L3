package handlers

import (
	"L3.7/internal/middleware"
	"github.com/gin-gonic/gin"
)

// Router роутер
type Router struct {
	engine         *gin.Engine
	authHandler    *AuthHandler
	itemHandler    *ItemHandler
	historyHandler *HistoryHandler
}

// NewRouter создает и настраивает новый роутер
func NewRouter(authHandler *AuthHandler, itemHandler *ItemHandler, historyHandler *HistoryHandler,
) *Router {
	router := &Router{
		engine:         gin.Default(),
		authHandler:    authHandler,
		itemHandler:    itemHandler,
		historyHandler: historyHandler,
	}

	// Настраиваем маршруты
	router.setupRoutes()

	return router
}

func (r *Router) setupRoutes() {
	r.engine.Static("/static", "./web/static")
	r.engine.GET("/script.js", func(c *gin.Context) {
		c.File("./web/static/script.js")
	})
	r.engine.StaticFile("/", "./web/static/index.html")

	// Логин
	r.engine.POST("/login", r.authHandler.Login)
	// Создание юзера
	r.engine.POST("/register", r.authHandler.Register)

	api := r.engine.Group("/warehouse")
	items := api.Group("/items")

	// Посмотреть всю историю (роль кладовщик)
	api.GET("/history", middleware.AuthMiddleware("admin"), r.historyHandler.GetAllHistory)

	// Создать айтем (доступно всем)
	items.POST("", middleware.AuthMiddleware(), r.itemHandler.CreateItem)
	// Получить айтем (доступно всем)
	items.GET("/:id", middleware.AuthMiddleware(), r.itemHandler.GetItem)
	// Получить все айтемы (доступно всем)
	items.GET("", middleware.AuthMiddleware(), r.itemHandler.GetAllItems)
	// Обновить айтем (роли кладовщик, менеджер)
	items.PUT("/:id", middleware.AuthMiddleware("admin", "manager"), r.itemHandler.UpdateItem)
	// Удалить айтем (роль кладовщик)
	items.DELETE("/:id", middleware.AuthMiddleware("admin"), r.itemHandler.DeleteItem)
	// Посмотреть историю изменения айтема (роли кладовщик, менеджер)
	items.GET("/:id/history", middleware.AuthMiddleware("admin", "manager"), r.historyHandler.GetItemHistory)
}

// Run запускает сервер на переданном порте
func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}
