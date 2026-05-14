package postgres

import (
	"L3.7/internal/models"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/wb-go/wbf/dbpg"
	"log"
	"time"
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

// SetCurrentUser устанавливает текущего пользователя в сессию БД (для триггеров)
func (d *DB) SetCurrentUser(username string) error {
	_, err := d.db.ExecContext(context.Background(), "SELECT set_config('app.current_user', $1, false)", username)
	return err
}

// SaveUser сохраняет юзера в БД
func (d *DB) SaveUser(user models.User) error {
	query := `INSERT INTO users (username, role) VALUES ($1, $2)`
	_, err := d.db.ExecContext(context.Background(), query, user.Username, user.Role)
	return err
}

// SaveItem записывает айтем в бд
func (d *DB) SaveItem(item models.Item) error {
	_, err := d.db.ExecContext(context.Background(), `INSERT INTO items (name, quantity, price) VALUES ($1,$2,$3)`,
		item.Name, item.Quantity, item.Price)
	return err
}

// GetItem получает айтем из бд
func (d *DB) GetItem(id int) (models.Item, error) {
	query := `SELECT * FROM items WHERE id = $1`
	row := d.db.QueryRowContext(context.Background(), query, id)
	var item models.Item
	err := row.Scan(&item.ID, &item.Name, &item.Quantity, &item.Price, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

// GetAllItems получает все данные айтемов из бд
func (d *DB) GetAllItems() ([]models.Item, error) {
	query := `SELECT * FROM items`
	rows, err := d.db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		err = rows.Scan(&item.ID, &item.Name, &item.Quantity, &item.Price, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// Exists существует ли запись в бд
func (d *DB) Exists(id int) (bool, error) {
	var exists bool
	err := d.db.QueryRowContext(context.Background(), `SELECT EXISTS (SELECT 1 FROM items WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

// UpdateItem обновляет данные в бд
func (d *DB) UpdateItem(id int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	updates["updated_at"] = time.Now()

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
func (d *DB) DeleteItem(id int) error {
	_, err := d.db.ExecContext(context.Background(), `DELETE FROM items WHERE id = $1`, id)
	return err
}

// GetUser получает юзера из бд
func (d *DB) GetUser(username string) (models.User, error) {
	query := `SELECT * FROM users WHERE username = $1`
	row := d.db.QueryRowContext(context.Background(), query, username)
	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.Role)
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

// GetItemHistory получает историю изменения из бд
func (d *DB) GetItemHistory(filter models.Filter) ([]models.ItemHistory, error) {
	query := `SELECT * FROM item_history WHERE 1=1`

	var args []interface{}
	argsCount := 1

	if filter.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argsCount)
		args = append(args, filter.Action)
		argsCount++
	}

	if filter.User != "" {
		query += fmt.Sprintf(" AND changed_by = $%d", argsCount)
		args = append(args, filter.User)
		argsCount++
	}

	if filter.ID != 0 {
		query += fmt.Sprintf(" AND item_id = $%d", argsCount)
		args = append(args, filter.ID)
		argsCount++
	}

	if !filter.DateFrom.IsZero() {
		query += fmt.Sprintf(" AND changed_at >= $%d", argsCount)
		args = append(args, filter.DateFrom)
		argsCount++
	}

	if !filter.DateTo.IsZero() {
		query += fmt.Sprintf(" AND changed_at <= $%d", argsCount)
		args = append(args, filter.DateTo)
		argsCount++
	}

	rows, err := d.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ItemHistory
	for rows.Next() {
		var h models.ItemHistory
		var oldData, newData []byte
		var itemID sql.NullInt32

		err = rows.Scan(&h.ID, &itemID, &h.Action, &oldData, &newData, &h.ChangedBy, &h.ChangedAt)
		if err != nil {
			return nil, err
		}
		if len(oldData) > 0 {
			err = json.Unmarshal(oldData, &h.OldData)
			if err != nil {
				return nil, err
			}
		}

		if len(newData) > 0 {
			err = json.Unmarshal(newData, &h.NewData)
			if err != nil {
				return nil, err
			}
		}

		if itemID.Valid {
			h.ItemID = int(itemID.Int32)
		}
		result = append(result, h)
	}
	return result, nil
}

func createTable(db *dbpg.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
    		id SERIAL PRIMARY KEY,
    		username VARCHAR(50) UNIQUE NOT NULL,
    		role VARCHAR(7) NOT NULL CHECK (role IN ('admin', 'manager', 'viewer'))
    		)`,
		`CREATE TABLE IF NOT EXISTS items (
				id SERIAL PRIMARY KEY,
				name VARCHAR(200) NOT NULL,
				quantity INTEGER NOT NULL DEFAULT 0,
				price DECIMAL(10, 2) NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
		`CREATE TABLE IF NOT EXISTS item_history (
				id SERIAL PRIMARY KEY,
				item_id INTEGER REFERENCES items(id) ON DELETE SET NULL,
				action VARCHAR(50) NOT NULL,
				old_data JSONB,
				new_data JSONB,
				changed_by VARCHAR(50), -- username из JWT
				changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				)`,
		// Функция для триггера INSERT
		`
			CREATE OR REPLACE FUNCTION log_item_insert()
			RETURNS TRIGGER AS $$
			BEGIN
				INSERT INTO item_history (item_id, action, new_data, changed_by)
				VALUES (NEW.id, 'INSERT', row_to_json(NEW), current_setting('app.current_user', true));
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;
			`,
		// Функция для триггера UPDATE
		`
			CREATE OR REPLACE FUNCTION log_item_update()
			RETURNS TRIGGER AS $$
			BEGIN
				INSERT INTO item_history (item_id, action, old_data, new_data, changed_by)
				VALUES (NEW.id, 'UPDATE', row_to_json(OLD), row_to_json(NEW), current_setting('app.current_user', true));
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;
			`,
		// Функция для триггера DELETE
		`CREATE OR REPLACE FUNCTION log_item_delete()
		RETURNS TRIGGER AS $$
		BEGIN
			INSERT INTO item_history (item_id, action, old_data, changed_by)
			VALUES (OLD.id, 'DELETE', row_to_json(OLD), current_setting('app.current_user', true));
			RETURN OLD;
		END;
		$$ LANGUAGE plpgsql;
		`,
		// Создание триггеров
		`
		CREATE OR REPLACE TRIGGER trigger_item_insert
			AFTER INSERT ON items
			FOR EACH ROW EXECUTE FUNCTION log_item_insert();
		
		CREATE OR REPLACE TRIGGER trigger_item_update
			AFTER UPDATE ON items
			FOR EACH ROW EXECUTE FUNCTION log_item_update();
		
		CREATE OR REPLACE TRIGGER trigger_item_delete
			BEFORE DELETE ON items
			FOR EACH ROW EXECUTE FUNCTION log_item_delete();
		`,
	}

	for _, q := range queries {
		_, err := db.ExecContext(context.Background(), q)
		if err != nil {
			return err
		}
	}

	return nil
}
