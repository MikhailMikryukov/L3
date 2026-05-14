package middleware

import (
	"log"
	"net/http"
)

// Logging логирование каждого запроса (метод, URL, время) в Stdout
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Вызываем следующий обработчик
		next.ServeHTTP(w, r)

		// Логируем после выполнения
		log.Printf("%s %s", r.Method, r.URL.String())
	})
}
