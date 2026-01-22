package persistence

import (
	"errors"
	"time"
)

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("item not found")

// Message represents a persisted message.
// We duplicate the core.Message structure here to avoid circular imports if needed,
// but ideally we should use a shared definition or simplistic byte storage.
// For now, let's store raw bytes and metadata.
type Message struct {
	ID        string
	Gateway   string
	Data      []byte
	CreatedAt time.Time
	Retries   int
}

// Store defines the interface for data persistence.
type Store interface {
	// Save persists a message.
	Save(msg *Message) error

	// GetPending retrieves pending messages for a gateway.
	GetPending(gateway string, limit int) ([]*Message, error)

	// Delete removes a message (after successful delivery).
	Delete(id string) error

	// Close closes the store.
	Close() error
}
