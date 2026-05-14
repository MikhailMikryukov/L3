package utils

import (
	"L3.7/internal/models"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

var jwtSecret []byte

// InitJwt записывает ключ
func InitJwt(secret string) {
	jwtSecret = []byte(secret)
}

// GenerateToken создает токен
func GenerateToken(username string, role string) (string, error) {
	claims := models.Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken проверяет токен
func ValidateToken(tokenString string) (*models.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.Claims{}, func(token *jwt.Token) (any, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*models.Claims)
	if ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
