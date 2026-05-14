package services

import (
	"L3.2/internal/entities"
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"
)

// URLRepository интерфейс обращения в бд
type URLRepository interface {
	SaveShortURL(ctx context.Context, originalURL string, shortURL string) error
	GetOriginalURL(ctx context.Context, shortURL string) (string, error)
	SaveClick(ctx context.Context, shortURL string, URLData entities.URLData) error
	DailyStatQuery(ctx context.Context, shortURL string, from time.Time, to time.Time) ([]entities.Analytics, error)
	SaveDailyStats(ctx context.Context) error
}

// URLService сервис работы с уведомлениями
type URLService struct {
	rep         URLRepository
	URLTemplate string
}

// New создание экземпляра URLService
func New(repo URLRepository, template string) *URLService {
	return &URLService{rep: repo,
		URLTemplate: template,
	}
}

// MakeShortURL создает короткую ссылку
func (s *URLService) MakeShortURL(URL string) (string, error) {

	shortCode := getShortCode(URL)

	ctx := context.Background()
	err := s.rep.SaveShortURL(ctx, URL, shortCode)
	if err != nil {
		return "", err
	}

	result := s.URLTemplate + shortCode
	return result, nil
}

// GetOriginalURL получить полную ссылку по короткой
func (s *URLService) GetOriginalURL(URL string) (string, error) {
	if !strings.Contains(URL, s.URLTemplate) {
		return "", fmt.Errorf("invalid URL")
	}

	short := URL[len(URL)-8:]

	ctx := context.Background()
	originalURL, err := s.rep.GetOriginalURL(ctx, short)
	if err != nil {
		return "", err
	}

	return originalURL, nil
}

// GetAnalytics получить аналитику за заданный период (periodType "all" - за все время)
func (s *URLService) GetAnalytics(URL string, periodType string, from string, to string) ([]entities.Analytics, error) {
	if !strings.Contains(URL, s.URLTemplate) {
		return nil, fmt.Errorf("invalid URL")
	}

	short := URL[len(URL)-8:]

	var layout string

	if periodType == "daily" {
		layout = "2006-01-02"
	}
	if periodType == "monthly" {
		layout = "2006-01"
	}

	var dateFrom time.Time
	var dateTo time.Time
	var err error

	// Если за все время - устанавливаем нулевые значения
	if periodType == "all" {
		dateFrom = time.Time{}
		dateTo = time.Time{}
	} else {
		dateFrom, err = time.Parse(layout, from)
		dateTo, err = time.Parse(layout, to)
		if err != nil {
			return []entities.Analytics{}, err
		}
	}

	// Получаем из бд статистику за период
	ctx := context.Background()
	analytics, err := s.rep.DailyStatQuery(ctx, short, dateFrom, dateTo)
	if err != nil {
		log.Println(err)
	}

	return analytics, nil
}

// SaveClick сохранить данные о переходе
func (s *URLService) SaveClick(URL string, date time.Time, userAgent string, ip string) error {
	URLData := entities.URLData{
		UserAgent: userAgent,
		Date:      date,
		IP:        ip,
	}

	short := URL[len(URL)-8:]

	// Сохраняем данные о переходе по ссылке
	ctx := context.Background()
	err := s.rep.SaveClick(ctx, short, URLData)
	if err != nil {
		return err
	}

	return nil
}

// StartDailyAggregation ежедневное агрегирование данных
func (s *URLService) StartDailyAggregation() {
	go func() {
		for {
			// Ждем до 00:05 следующего дня
			now := time.Now()
			next := time.Date(
				now.Year(), now.Month(), now.Day()+1,
				0, 5, 0, 0, now.Location(),
			)

			time.Sleep(next.Sub(now))

			// Сохраняем в бд статистику за день
			ctx := context.Background()
			err := s.rep.SaveDailyStats(ctx)
			if err != nil {
				log.Println(err)
			}
		}
	}()
}

func getShortCode(URL string) string {
	hash := md5.Sum([]byte(URL))
	encoded := base64.URLEncoding.EncodeToString(hash[:])
	shortCode := strings.ReplaceAll(encoded, "/", "_")
	shortCode = strings.ReplaceAll(shortCode, "+", "-")
	shortCode = strings.ReplaceAll(shortCode, "=", "")
	if len(shortCode) > 8 {
		return shortCode[:8]
	}

	return shortCode
}
