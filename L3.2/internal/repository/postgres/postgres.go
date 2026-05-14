package postgres

import (
	"L3.2/internal/entities"
	"context"
	"encoding/json"
	"fmt"
	"github.com/wb-go/wbf/dbpg"
	"time"
)

// Postgres обертка над wbf
type Postgres struct {
	db *dbpg.DB
}

// New создание экземпляра
func New(conn string) (*Postgres, error) {
	opts := &dbpg.Options{MaxOpenConns: 10, MaxIdleConns: 5}
	db, err := dbpg.New(conn, nil, opts)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Создаем таблицу если нужно
	err = CreateTable(db)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &Postgres{
		db: db,
	}, nil
}

// SaveShortURL сохранить короткую ссылку в бд
func (db *Postgres) SaveShortURL(ctx context.Context, originalURL string, shortURL string) error {
	query := "INSERT INTO urls (short_code, original_url) VALUES ($1, $2)"

	_, err := db.db.ExecContext(ctx, query, shortURL, originalURL)
	if err != nil {
		return fmt.Errorf("ошибка записи в urls %v", err)
	}

	queryDailyStat := "INSERT INTO daily_stats (short_code) VALUES ($1)"
	_, err = db.db.ExecContext(ctx, queryDailyStat, shortURL)
	if err != nil {
		return fmt.Errorf("ошибка записи в daily_stats %v", err)
	}

	return nil
}

// GetOriginalURL получить полную ссылку из бд
func (db *Postgres) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	query := "SELECT original_url FROM urls WHERE short_code = '" + shortURL + "'"

	row := db.db.QueryRowContext(ctx, query)
	var original string
	err := row.Scan(&original)
	if err != nil {
		return "", fmt.Errorf("ошибка получения полной ссылки в бд %v", err)
	}

	return original, nil
}

// SaveClick сохранить в бд данные о переходе
func (db *Postgres) SaveClick(ctx context.Context, shortURL string, URLData entities.URLData) error {
	// Вставляем в бд данные о переходе
	query := "INSERT INTO clicks (short_code, clicked_at, user_agent, ip_address) VALUES ($1, $2, $3, $4)"
	_, err := db.db.ExecContext(ctx, query, shortURL, URLData.Date, URLData.UserAgent, URLData.IP)
	if err != nil {
		return fmt.Errorf("ошибка записи в clicks %v", err)
	}

	// Обновляем счетчик переходов
	updateClicksNumQuery := "UPDATE urls SET total_clicks = total_clicks + 1, today_clicks = today_clicks + 1 WHERE short_code = '" + shortURL + "';"
	_, err = db.db.ExecContext(ctx, updateClicksNumQuery)
	if err != nil {
		return fmt.Errorf("ошибка обновления кликов в urls %v", err)
	}

	return nil
}

// DailyStatQuery запрос в таблицу daily_stat, где по дням агрегированные данные
func (db *Postgres) DailyStatQuery(ctx context.Context, shortURL string, from time.Time, to time.Time) ([]entities.Analytics, error) {
	var period string
	if !from.IsZero() && !to.IsZero() {
		period = "AND stat_date >= '" + from.Format("2006-01-02") + "' AND STAT_date + INTERVAL '1' SECOND <= '" + to.Format("2006-01-02") + "'"
	}
	query := "SELECT \n" +
		"stat_date,\n" +
		"clicks,\n" +
		"user_agent_stat,\n" +
		"click_time_stat\n" +
		"FROM daily_stats \n" +
		"WHERE short_code = '" + shortURL + "'\n" +
		period +
		"\nORDER BY clicks\n" +
		"LIMIT 10"

	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса к daily_stat в бд %v", err)
	}

	var analytics []entities.Analytics

	for rows.Next() {

		var statDate time.Time
		var summary entities.SummaryClicks
		var userAgents []entities.UserAgent
		var clicksTime entities.ClicksTime

		var userAgentsJSON []byte
		var clicksTimeJSON []byte

		err = rows.Scan(&statDate, &summary.Total, &userAgentsJSON, &clicksTimeJSON)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(userAgentsJSON, &userAgents)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(clicksTimeJSON, &clicksTime)
		if err != nil {
			return nil, err
		}

		dayAnalytics := entities.Analytics{
			StatDate:   statDate,
			Summary:    summary,
			UserAgents: userAgents,
			ClicksTime: clicksTime,
		}

		analytics = append(analytics, dayAnalytics)
	}

	return analytics, nil

}

// SaveDailyStats сохраняем статистику в бд за прошедший день
func (db *Postgres) SaveDailyStats(ctx context.Context) error {
	// Вставка данных за вчерашний день в таблицу daily_stats и обновление счетчика кликов за день
	query := `INSERT INTO daily_stats (short_code, stat_date, clicks, user_agent_stat, click_time_stat)
				SELECT 
					u.short_code,
					CURRENT_DATE - INTERVAL '1 day' as yesterday,
					u.today_clicks,
					
					-- User-Agent: массив объектов [{"user_agent": "...", "clicks": 5}, ...]
					COALESCE((
						SELECT jsonb_agg(
							jsonb_build_object(
								'user_agent', ua.user_agent,
								'clicks', ua.cnt
							)
							ORDER BY ua.cnt DESC  -- сортируем по количеству кликов
						)
						FROM (
							SELECT 
								user_agent,
								COUNT(*) as cnt
							FROM clicks c
							WHERE c.short_code = u.short_code
							  AND DATE(c.clicked_at) = CURRENT_DATE - INTERVAL '1 day'
							  AND c.user_agent IS NOT NULL
							  AND c.user_agent != ''
							GROUP BY user_agent
							ORDER BY cnt DESC
							LIMIT 50
						) ua
					), '[]'::jsonb) as user_agent_stat,
					
					-- Время суток
					COALESCE((
						SELECT jsonb_build_object(
							'night',   COUNT(CASE WHEN EXTRACT(HOUR FROM clicked_at) < 8 THEN 1 END),
							'day',     COUNT(CASE WHEN EXTRACT(HOUR FROM clicked_at) BETWEEN 8 AND 15 THEN 1 END),
							'evening', COUNT(CASE WHEN EXTRACT(HOUR FROM clicked_at) >= 16 THEN 1 END)
						
						)
						FROM clicks c
						WHERE c.short_code = u.short_code
						  AND DATE(c.clicked_at) = CURRENT_DATE - INTERVAL '1 day'
					), jsonb_build_object('night', 0, 'day', 0, 'evening', 0)) as click_time_stat
					
				FROM urls u
				WHERE u.today_clicks > 0;`

	_, err := db.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("ошибка записи в бд %v", err)
	}

	return nil
}

// CreateTable создает таблицу в бд
func CreateTable(db *dbpg.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS urls (
    		id SERIAL PRIMARY KEY,
    		short_code VARCHAR(8) NOT NULL UNIQUE,
    		original_url TEXT NOT NULL,
    		total_clicks INT DEFAULT 0,
            today_clicks INT DEFAULT 0,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    	)`,
		`CREATE TABLE IF NOT EXISTS clicks (
    		id BIGSERIAL PRIMARY KEY,
            short_code VARCHAR(8),
    		clicked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    		user_agent TEXT NOT NULL,
    		ip_address VARCHAR(45)
        )`,
		`CREATE TABLE IF NOT EXISTS daily_stats (
			short_code VARCHAR(8) NOT NULL,
    		stat_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            clicks INT DEFAULT 0,
    		user_agent_stat JSONB,
    		click_time_stat JSONB
        )`,
	}

	for _, query := range queries {
		_, err := db.ExecContext(context.Background(), query)
		if err != nil {
			return fmt.Errorf("не удалось создать таблицу %v", err)
		}
	}

	return nil
}
