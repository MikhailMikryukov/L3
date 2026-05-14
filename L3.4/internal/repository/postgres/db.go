package postgres

import (
	"L3.4/internal/entities"
	"L3.4/pkg/messaging"
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"fmt"
	"github.com/wb-go/wbf/dbpg"
)

// DB обертка над DB wbf
type DB struct {
	db *dbpg.DB
}

// New создание экземпляра
func New(conn string) (*DB, error) {
	opts := &dbpg.Options{MaxOpenConns: 10, MaxIdleConns: 5}
	db, err := dbpg.New(conn, nil, opts)
	if err != nil {
		return nil, err
	}

	// Создаем таблицу если нужно
	err = createTable(db)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &DB{
		db: db,
	}, nil
}

// Create вставить задание в бд
func (d *DB) Create(ctx context.Context, task *entities.Task) error {
	params, _ := json.Marshal(task.Params)
	query := `INSERT INTO tasks (id, original_name, original_path, mime_type, params) VALUES ($1, $2, $3, $4, $5)`
	_, err := d.db.ExecContext(ctx, query, task.ID, task.OriginalName, task.OriginalPath, task.MIMEType, params)
	return err
}

// CheckStatus узнать статус задания из бд
func (d *DB) CheckStatus(ctx context.Context, id string) (string, error) {
	query := `SELECT status FROM tasks WHERE id = $1`

	row := d.db.QueryRowContext(ctx, query, id)

	var status string
	err := row.Scan(&status)
	if err != nil {
		return "", err
	}

	return status, nil
}

// UpdateStatus обновить статус в бд
func (d *DB) UpdateStatus(ctx context.Context, taskMsg messaging.TaskMessage, status string, variants *entities.ImageVariants) error {
	query := `UPDATE tasks SET status = $1, variants = $2, updated_at = NOW() WHERE id =$3`

	if variants == nil {
		_, err := d.db.ExecContext(ctx, query, status, nil, taskMsg.ID)
		return err
	}

	variantsJSON, err := json.Marshal(variants)
	if err != nil {
		return err
	}
	_, err = d.db.ExecContext(ctx, query, status, variantsJSON, taskMsg.ID)
	return err
}

// GetTaskByID получить задание по ID из бд
func (d *DB) GetTaskByID(ctx context.Context, id string) (*entities.Task, error) {
	query := `SELECT id, status, original_path, params, variants FROM tasks WHERE id = $1`
	var paramsJSON []byte
	var variantsJSON sql.NullString

	rows := d.db.QueryRowContext(ctx, query, id)
	var task entities.Task
	err := rows.Scan(&task.ID, &task.Status, &task.OriginalPath, &paramsJSON, &variantsJSON)
	if err != nil {
		return nil, err
	}
	var params entities.ProcessingParams
	var variants entities.ImageVariants

	err = json.Unmarshal(paramsJSON, &params)
	if err != nil {
		return nil, err
	}

	if variantsJSON.Valid {
		err = json.Unmarshal([]byte(variantsJSON.String), &variants)
		if err != nil {
			return nil, err
		}
	}

	task.Params = params
	task.Variants = variants
	return &task, nil
}

// Delete удалить задание из бд
func (d *DB) Delete(ctx context.Context, task *entities.Task) error {
	query := `DELETE FROM tasks WHERE id = $1`

	_, err := d.db.ExecContext(ctx, query, task.ID)

	return err
}

func createTable(db *dbpg.DB) error {
	query := `CREATE TABLE IF NOT EXISTS tasks (
    		id VARCHAR(36) PRIMARY KEY,
    		status VARCHAR(10) NOT NULL DEFAULT 'pending',
			original_name TEXT NOT NULL,
    		original_path TEXT NOT NULL,
    		mime_type VARCHAR(128) NOT NULL,
    		variants  JSONB, -- Пути к обработанным файлам
    		params  JSONB,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    	)`

	_, err := db.ExecContext(context.Background(), query)
	if err != nil {
		return fmt.Errorf("не удалось создать таблицу %v", err)
	}

	return nil
}
