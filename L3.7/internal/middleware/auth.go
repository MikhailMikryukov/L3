package middleware

import (
	"L3.7/internal/models"
	"L3.7/internal/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// UserContextKey ключ, по которому ищем юзера
const UserContextKey = "user"

// AuthMiddleware проверяет токен и доступ
func AuthMiddleware(requiredRole ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
		}

		// Проверяем формат Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		token := parts[1]

		// Валидируем токен
		claims, err := utils.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Проверяем роль если требуется
		if len(requiredRole) > 0 {
			hasAccess := false
			for _, role := range requiredRole {
				if claims.Role == role {
					hasAccess = true
					break
				}
			}
			if !hasAccess {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				c.Abort()
				return
			}
		}

		// Добавляем информацию о пользователе в контекст
		c.Set(UserContextKey, claims)
		c.Next()
	}
}

// GetUserFromContext получает текущего юзера
func GetUserFromContext(c *gin.Context) *models.Claims {
	if user, exists := c.Get(UserContextKey); exists {
		if claims, ok := user.(*models.Claims); ok {
			return claims
		}
	}

	return nil
}
