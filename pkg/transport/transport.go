// Package transport defines the abstract interface for communication channels.
// It provides a unified API for different physical/logical transports like
// Serial, TCP, UDP, WebSocket, MQTT, etc.
package transport

import (
	"context"
	"time"
)

// ConnectionState represents the current state of a transport connection.
type ConnectionState int

const (
	// StateDisconnected indicates the transport is not connected.
	StateDisconnected ConnectionState = iota
	// StateConnecting indicates a connection attempt is in progress.
	StateConnecting
	// StateConnected indicates the transport is connected and ready.
	StateConnected
	// StateReconnecting indicates the transport is attempting to reconnect.
	StateReconnecting
	// StateError indicates the transport is in an error state.
	StateError
)

func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// Transport is the core interface for all communication channels.
// Implementations must be safe for concurrent use.
type Transport interface {
	// Connect establishes a connection to the remote endpoint.
	// It blocks until connected or context is cancelled.
	Connect(ctx context.Context) error

	// Close gracefully closes the connection.
	// It should release all resources and stop any goroutines.
	Close() error

	// IsConnected returns true if the transport is currently connected.
	IsConnected() bool

	// Send transmits data over the transport.
	// It returns the number of bytes sent and any error encountered.
	Send(ctx context.Context, data []byte) (int, error)

	// Receive reads data from the transport.
	// It blocks until data is available or context is cancelled.
	Receive(ctx context.Context) ([]byte, error)

	// Configure applies configuration to the transport.
	// Some configurations may require the transport to be disconnected.
	Configure(config Config) error

	// Info returns information about the transport.
	Info() Info

	// SetEventHandler sets the handler for transport events.
	SetEventHandler(handler EventHandler)
}

// Config holds the configuration for a transport.
type Config struct {
	// Type is the transport type (serial, tcp, udp, mqtt, etc.)
	Type string `yaml:"type" json:"type"`

	// Address is the connection address.
	// Format depends on transport type:
	//   - serial: "/dev/ttyUSB0" or "COM1"
	//   - tcp/udp: "host:port"
	//   - mqtt: "tcp://broker:1883"
	Address string `yaml:"address" json:"address"`

	// Options contains transport-specific options.
	Options map[string]interface{} `yaml:"options" json:"options"`

	// BufferSize is the size of read/write buffers.
	BufferSize int `yaml:"buffer_size" json:"buffer_size"`

	// Timeout is the default timeout for operations.
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// ReconnectPolicy defines auto-reconnect behavior.
	ReconnectPolicy *ReconnectPolicy `yaml:"reconnect" json:"reconnect"`

	// TLS configures Transport Layer Security.
	TLS *TLSConfig `yaml:"tls" json:"tls"`
}

// TLSConfig holds TLS/SSL configuration.
type TLSConfig struct {
	// Enabled enables TLS.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// CertFile is the path to the certificate file.
	CertFile string `yaml:"cert_file" json:"cert_file" validate:"required_if=Enabled true"`

	// KeyFile is the path to the key file.
	KeyFile string `yaml:"key_file" json:"key_file" validate:"required_if=Enabled true"`

	// CAFile is the path to the CA certificate file for verifying the server or client.
	CAFile string `yaml:"ca_file" json:"ca_file"`

	// InsecureSkipVerify checks whether to skip certificate verification (for internal/testing).
	InsecureSkipVerify bool `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`

	// MinVersion is the minimum TLS version (e.g., "1.2", "1.3").
	MinVersion string `yaml:"min_version" json:"min_version"`
}

// ReconnectPolicy defines how the transport should handle reconnection.
type ReconnectPolicy struct {
	// Enabled enables auto-reconnect.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// MaxAttempts is the maximum number of reconnect attempts (0 = infinite).
	MaxAttempts int `yaml:"max_attempts" json:"max_attempts"`

	// InitialDelay is the initial delay before first reconnect attempt.
	InitialDelay time.Duration `yaml:"initial_delay" json:"initial_delay"`

	// MaxDelay is the maximum delay between reconnect attempts.
	MaxDelay time.Duration `yaml:"max_delay" json:"max_delay"`

	// Multiplier is the multiplier for exponential backoff.
	Multiplier float64 `yaml:"multiplier" json:"multiplier"`
}

// DefaultReconnectPolicy returns a sensible default reconnect policy.
func DefaultReconnectPolicy() *ReconnectPolicy {
	return &ReconnectPolicy{
		Enabled:      true,
		MaxAttempts:  0, // infinite
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// Info contains runtime information about a transport.
type Info struct {
	// ID is a unique identifier for this transport instance.
	ID string `json:"id"`

	// Type is the transport type.
	Type string `json:"type"`

	// Address is the configured address.
	Address string `json:"address"`

	// State is the current connection state.
	State ConnectionState `json:"state"`

	// Statistics contains transport statistics.
	Statistics Statistics `json:"statistics"`

	// ConnectedAt is when the connection was established.
	ConnectedAt *time.Time `json:"connected_at,omitempty"`

	// LastError is the last error that occurred.
	LastError string `json:"last_error,omitempty"`
}

// Statistics contains transport performance statistics.
type Statistics struct {
	// BytesSent is the total number of bytes sent.
	BytesSent uint64 `json:"bytes_sent"`

	// BytesReceived is the total number of bytes received.
	BytesReceived uint64 `json:"bytes_received"`

	// MessagesSent is the total number of messages sent.
	MessagesSent uint64 `json:"messages_sent"`

	// MessagesReceived is the total number of messages received.
	MessagesReceived uint64 `json:"messages_received"`

	// Errors is the total number of errors encountered.
	Errors uint64 `json:"errors"`

	// Reconnects is the number of reconnection attempts.
	Reconnects uint64 `json:"reconnects"`

	// AverageLatency is the average round-trip latency.
	AverageLatency time.Duration `json:"average_latency"`
}

// EventType represents the type of transport event.
type EventType int

const (
	// EventConnected is emitted when connection is established.
	EventConnected EventType = iota
	// EventDisconnected is emitted when connection is lost.
	EventDisconnected
	// EventReconnecting is emitted when reconnection is attempted.
	EventReconnecting
	// EventError is emitted when an error occurs.
	EventError
	// EventDataReceived is emitted when data is received.
	EventDataReceived
)

// Event represents a transport event.
type Event struct {
	// Type is the event type.
	Type EventType

	// Transport is the transport that emitted the event.
	Transport Transport

	// Error is the error (for error events).
	Error error

	// Data is associated data (for data events).
	Data []byte

	// Timestamp is when the event occurred.
	Timestamp time.Time
}

// EventHandler handles transport events.
type EventHandler interface {
	OnEvent(event Event)
}

// EventHandlerFunc is a function adapter for EventHandler.
type EventHandlerFunc func(event Event)

// OnEvent implements EventHandler.
func (f EventHandlerFunc) OnEvent(event Event) {
	f(event)
}

// Factory creates transport instances.
type Factory interface {
	// Type returns the transport type this factory creates.
	Type() string

	// Create creates a new transport instance with the given config.
	Create(config Config) (Transport, error)

	// Validate validates the configuration for this transport type.
	Validate(config Config) error
}

// Registry manages transport factories.
type Registry interface {
	// Register adds a factory to the registry.
	Register(factory Factory) error

	// Get retrieves a factory by type.
	Get(transportType string) (Factory, error)

	// List returns all registered transport types.
	List() []string

	// Create creates a transport using the appropriate factory.
	Create(config Config) (Transport, error)
}
