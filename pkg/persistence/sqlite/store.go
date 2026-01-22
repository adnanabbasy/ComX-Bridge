package sqlite

import (
	"database/sql"

	"github.com/commatea/ComX-Bridge/pkg/persistence"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// SQLiteStore implements persistence.Store.
type SQLiteStore struct {
	db *sql.DB
}

// NewStore creates a new SQLite store.
func NewStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	s := &SQLiteStore{db: db}
	if err := s.init(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *SQLiteStore) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		gateway TEXT NOT NULL,
		data BLOB,
		created_at DATETIME,
		retries INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_gateway_created ON messages(gateway, created_at);
	`
	_, err := s.db.Exec(query)
	return err
}

// Save persists a message.
func (s *SQLiteStore) Save(msg *persistence.Message) error {
	query := `INSERT INTO messages (id, gateway, data, created_at, retries) VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, msg.ID, msg.Gateway, msg.Data, msg.CreatedAt, msg.Retries)
	return err
}

// GetPending retrieves pending messages for a gateway.
func (s *SQLiteStore) GetPending(gateway string, limit int) ([]*persistence.Message, error) {
	query := `SELECT id, gateway, data, created_at, retries FROM messages WHERE gateway = ? ORDER BY created_at ASC LIMIT ?`
	rows, err := s.db.Query(query, gateway, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*persistence.Message
	for rows.Next() {
		var msg persistence.Message
		if err := rows.Scan(&msg.ID, &msg.Gateway, &msg.Data, &msg.CreatedAt, &msg.Retries); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}
	return messages, nil
}

// Delete removes a message.
func (s *SQLiteStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM messages WHERE id = ?`, id)
	return err
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
