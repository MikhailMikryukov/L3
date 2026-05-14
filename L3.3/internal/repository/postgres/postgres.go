package postgres

import (
	"L3.3/internal/entities"
	"context"
	"database/sql"
	"fmt"
	"github.com/wb-go/wbf/dbpg"
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

// SaveComment сохранить комментарий в бд
func (db *Postgres) SaveComment(ctx context.Context, comment entities.Comment) error {
	query := "INSERT INTO comments (author, parent_id, text) VALUES ($1, $2, $3)"

	_, err := db.db.ExecContext(ctx, query, comment.Author, comment.Parent, comment.Text)
	if err != nil {
		return fmt.Errorf("ошибка записи в comments %v", err)
	}

	return nil
}

// GetComments получить комментарий и все вложенные из бд
func (db *Postgres) GetComments(ctx context.Context, id string) ([]entities.Comment, error) {
	query := `WITH RECURSIVE comment_tree AS (
			-- Базовый запрос: находим начальный комментарий
			SELECT 
				id,
				parent_id,
				author,
				text,
				created_at,
				0 as level,  -- уровень вложенности
				ARRAY[id] as path  -- путь для сортировки
			FROM comments
			WHERE id = $1
			
			UNION ALL
			
			-- Находим все дочерние комментарии
			SELECT 
				c.id,
				c.parent_id,
				c.author,
				c.text,
				c.created_at,
				ct.level + 1,
				ct.path || c.id
			FROM comments c
			INNER JOIN comment_tree ct ON c.parent_id = ct.id
		)
		SELECT 
			id,
			author,
			parent_id,
			text,
			created_at
			
		FROM comment_tree
		ORDER BY path;  -- сортируем так, чтобы родители были перед детьми`

	rows, err := db.db.QueryContext(ctx, query, id)

	comments, err := parseRowsToComment(rows)
	return comments, err
}

// DeleteComment удалить комментарий и все вложенные из бд
func (db *Postgres) DeleteComment(ctx context.Context, id string) error {
	query := "DELETE FROM comments WHERE id = $1 OR parent_id = $1"

	_, err := db.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления комментария: %v", err)
	}

	return nil
}

// GetAllCommentsWithLimit получить комментарии с заданными параметрами из бд
func (db *Postgres) GetAllCommentsWithLimit(ctx context.Context, limit int, offset int, sortBy string, searchQuery string) ([]entities.Comment, error) {
	// Базовый запрос
	query := `SELECT id, author, parent_id, text, created_at 
              FROM comments`

	// Добавляем поиск, если есть
	if searchQuery != "" {
		query += " WHERE text LIKE '%" + searchQuery + "%'"
	}

	// Добавляем сортировку, если есть
	if sortBy != "" {
		switch sortBy {
		case "author":
			query += " ORDER BY author ASC"
		case "created_at_asc":
			query += " ORDER BY created_at ASC"
		default: // created_at (новые сверху)
			query += " ORDER BY created_at DESC"
		}
	}

	// Добавляем ограничение на количество, если есть
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	// Добавляем пропуск записей, если есть
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", offset)
	}

	rows, err := db.db.QueryContext(ctx, query)

	comments, err := parseRowsToComment(rows)

	return comments, err
}

func parseRowsToComment(rows *sql.Rows) ([]entities.Comment, error) {
	defer rows.Close()
	var comments []entities.Comment

	for rows.Next() {
		var comment entities.Comment
		err := rows.Scan(&comment.ID, &comment.Author, &comment.Parent, &comment.Text, &comment.CreatedAt)
		if err != nil {
			return []entities.Comment{}, fmt.Errorf("ошибка получения комментария из бд: %v", err)
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

// CreateTable создает таблицу в бд
func CreateTable(db *dbpg.DB) error {
	query := `CREATE TABLE IF NOT EXISTS comments (
    		id SERIAL PRIMARY KEY,
			author TEXT NOT NULL,
    		parent_id INT DEFAULT 0,
    		text TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    	)`

	_, err := db.ExecContext(context.Background(), query)
	if err != nil {
		return fmt.Errorf("не удалось создать таблицу %v", err)
	}

	return nil
}
