package postgres

import (
	"L3.6/internal/entities"
	"context"
	"fmt"
	"github.com/wb-go/wbf/dbpg"
	"log"
	"time"
)

// DB обертка над DB wbf
type DB struct {
	db  *dbpg.DB
}

// New создание экземпляра
func New(conn string) (*DB, error) {
	opts := &dbpg.Options{MaxOpenConns: 10, MaxIdleConns: 5}
	db, err := dbpg.New(conn, nil, opts)
	if err != nil {
		return nil, err
	}

	// Создаем таблицу если нужно
	err = createTable( db)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &DB{
		db:  db,
	}, nil
}

// CreateItem записывает данные в бд
func (d *DB) CreateItem(item entities.Item) error {
	_, err := d.db.ExecContext(context.Background(), `INSERT INTO items (id, type, amount, created_at, category, comment) VALUES ($1,$2,$3,$4,$5, $6)`,
		item.ID, item.Type, item.Amount, item.Date, item.Category, item.Comment)
	return err
}

// GetItem получает данные из бд
func (d *DB) GetItem(id string) (entities.Item, error) {
	query := `SELECT * FROM items WHERE id = $1`
	row := d.db.QueryRowContext(context.Background(), query, id)
	var item entities.Item
	err := row.Scan(&item.ID, &item.Type, &item.Amount, &item.Date, &item.Category, &item.Comment)
	return item, err
}

// Exists существует ли запись в бд
func (d *DB) Exists(id string) (bool, error) {
	var exists bool
	err := d.db.QueryRowContext(context.Background(), `SELECT EXISTS (SELECT 1 FROM items WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

// UpdateItem обновляет данные в бд
func (d *DB) UpdateItem(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	query := `UPDATE items SET `

	var args []interface{}
	i := 1
	for field, value := range updates {
		query += fmt.Sprintf("%s = $%d, ", field, i)
		args = append(args, value)
		i++
	}

	// Убираем последнюю запятую и пробел
	query = query[:len(query)-2]
	query += fmt.Sprintf(" WHERE id = $%d", i)
	args = append(args, id)

	_, err := d.db.ExecContext(context.Background(), query, args...)
	return err
}

// DeleteItem удаляет запись
func (d *DB) DeleteItem(id string) error {
	_, err := d.db.ExecContext(context.Background(), `DELETE FROM items WHERE id = $1`, id)
	return err
}

// GetAnalytics получает аналитику из бд
func (d *DB) GetAnalytics(from time.Time, to time.Time) ([]entities.Analytics, error) {
	query := `SELECT 
    				category,
    				SUM(amount) AS sum_amount, 
    				AVG(amount) as avg_amount, 
    				COUNT(*) as count_items, 
    				PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY amount) AS median_amount,
					PERCENTILE_CONT(0.9) WITHIN GROUP (ORDER BY amount) AS percentile_90
				FROM items 
         		WHERE 1=1`

	var args []interface{}
	argCounter := 1

	if !from.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d ", argCounter)
		args = append(args, from)
		argCounter++
	}

	if !to.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d ", argCounter)
		args = append(args, to)
		argCounter++
	}

	query += `GROUP BY category`

	rows, err := d.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entities.Analytics
	for rows.Next() {
		var a entities.Analytics
		err = rows.Scan(&a.Category, &a.Sum, &a.Avg, &a.Count, &a.Median, &a.Percentile90)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}

	return result, nil
}

// GetFiltered получает фильтрованные данные из бд
func (d *DB) GetFiltered(filter entities.Filter) ([]entities.Item, error) {
	query := `SELECT * FROM items WHERE 1=1`
	var args []interface{}

	query, args = d.getFilterQuery(query, args, filter)

	// Добавляем пагинацию
	argsLen := len(args) + 1
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argsLen)
		args = append(args, filter.Limit)
		argsLen++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argsLen)
		args = append(args, filter.Offset)
	}

	// Выполняем запрос
	rows, err := d.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []entities.Item
	for rows.Next() {
		var item entities.Item
		err := rows.Scan(&item.ID, &item.Type, &item.Amount, &item.Date, &item.Category, &item.Comment)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// GetTotalPages получает общее кол-во записей по запросу из бд
func (d *DB) GetTotalPages(filter entities.Filter) (int, error) {
	query := `SELECT COUNT(*) FROM items WHERE 1=1`
	var args []interface{}

	query, args = d.getFilterQuery(query, args, filter)

	row := d.db.QueryRowContext(context.Background(), query, args...)

	var totalRows int
	err := row.Scan(&totalRows)
	if err != nil {
		return 1, err
	}

	return (totalRows + filter.Limit - 1) / filter.Limit, err
}

func (d *DB) getFilterQuery(initialQuery string, args []interface{}, filter entities.Filter) (string, []interface{}) {
	argCounter := 1

	if !filter.From.IsZero() {
		initialQuery += fmt.Sprintf(" AND created_at >= $%d", argCounter)
		args = append(args, filter.From)
		argCounter++
	}

	if !filter.To.IsZero() {
		initialQuery += fmt.Sprintf(" AND created_at <= $%d", argCounter)
		args = append(args, filter.To)
		argCounter++
	}

	if filter.Type != "" {
		initialQuery += fmt.Sprintf(" AND type = $%d", argCounter)
		args = append(args, filter.Type)
		argCounter++
	}

	if filter.Category != "" {
		initialQuery += fmt.Sprintf(" AND category = $%d", argCounter)
		args = append(args, filter.Category)
		argCounter++
	}

	if filter.AmountMin > 0 {
		initialQuery += fmt.Sprintf(" AND amount >= $%d", argCounter)
		args = append(args, filter.AmountMin)
		argCounter++
	}

	if filter.AmountMax > 0 {
		initialQuery += fmt.Sprintf(" AND amount <= $%d", argCounter)
		args = append(args, filter.AmountMax)
		argCounter++
	}

	if filter.Search != "" {
		initialQuery += fmt.Sprintf(" AND comment ILIKE $%d", argCounter)
		args = append(args, "%"+filter.Search+"%")
		argCounter++
	}

	if filter.SortBy != "" {
		initialQuery += fmt.Sprintf(" ORDER BY %s", filter.SortBy)

		if filter.SortOrder != "" {
			initialQuery += filter.SortOrder
		}
	}

	return initialQuery, args
}

func createTable(db *dbpg.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS items (
    		id VARCHAR(36) PRIMARY KEY,
    		type VARCHAR(7),
			amount int,
    		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    		category TEXT NOT NULL,
    		comment TEXT
    	)`,
	}

	for _, q := range queries {
		_, err := db.ExecContext(context.Background(), q)
		if err != nil {
			return err
		}
	}

	return nil
}
