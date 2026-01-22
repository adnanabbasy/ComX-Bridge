// Package udp provides UDP transport implementations.
package udp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/transport"
)

// Common errors.
var (
	ErrNotConnected = errors.New("not connected")
	ErrConnClosed   = errors.New("connection closed")
)

// Config holds UDP-specific configuration.
type Config struct {
	// Address is the local address to listen on (server) or remote address (client).
	Address string `yaml:"address" json:"address"`

	// Mode is the UDP mode (unicast, multicast, broadcast).
	Mode string `yaml:"mode" json:"mode"`

	// ReadBufferSize is the read buffer size.
	ReadBufferSize int `yaml:"read_buffer_size" json:"read_buffer_size"`

	// WriteBufferSize is the write buffer size.
	WriteBufferSize int `yaml:"write_buffer_size" json:"write_buffer_size"`

	// ReadTimeout is the read timeout.
	ReadTimeout time.Duration `yaml:"read_timeout" json:"read_timeout"`

	// WriteTimeout is the write timeout.
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
}

// DefaultConfig returns a default UDP configuration.
func DefaultConfig() Config {
	return Config{
		Mode:            "unicast",
		ReadBufferSize:  8192,
		WriteBufferSize: 8192,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
	}
}

// Transport implements the transport.Transport interface for UDP.
type Transport struct {
	mu sync.RWMutex

	config  Config
	tConfig transport.Config

	conn         *net.UDPConn
	id           string
	state        transport.ConnectionState
	eventHandler transport.EventHandler
	stats        transport.Statistics

	readBuffer  []byte
	connectedAt *time.Time
	lastError   error

	ctx    context.Context
	cancel context.CancelFunc
}

// NewTransport creates a new UDP transport.
func NewTransport(config transport.Config) (*Transport, error) {
	udpConfig := DefaultConfig()

	// Parse configuration
	udpConfig.Address = config.Address

	if opts := config.Options; opts != nil {
		if v, ok := opts["mode"].(string); ok {
			udpConfig.Mode = v
		}
		if v, ok := opts["read_buffer_size"].(int); ok {
			udpConfig.ReadBufferSize = v
		}
	}

	if config.Timeout > 0 {
		udpConfig.ReadTimeout = config.Timeout
	}
	if config.BufferSize > 0 {
		udpConfig.ReadBufferSize = config.BufferSize
	}

	return &Transport{
		config:     udpConfig,
		tConfig:    config,
		id:         fmt.Sprintf("udp-%s", udpConfig.Address),
		state:      transport.StateDisconnected,
		readBuffer: make([]byte, udpConfig.ReadBufferSize),
	}, nil
}

// Connect establishes a UDP connection.
func (t *Transport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == transport.StateConnected {
		return nil
	}

	t.state = transport.StateConnecting
	t.ctx, t.cancel = context.WithCancel(ctx)

	addr, err := net.ResolveUDPAddr("udp", t.config.Address)
	if err != nil {
		t.state = transport.StateError
		t.lastError = err
		return err
	}

	// For UDP, we "Listen" if we want to receive.
	// If it's pure client (sending only), Dial is fine, but typically we want both.
	// We'll treat it as a symmetric endpoint.
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		// Try dialing if listen fails (maybe it's a remote address?)
		// Actually, for UDP, "Connect" usually means resolving the address for Send.
		// But in this framework, we need a `conn` to Read from.
		// If Address is remote, we should Listen on local interface (e.g. :0 or specific port).
		// This implementation assumes Address is LOCAL BIND ADDRESS.
		// If we want to send to a specific remote, we might need separate config.
		// However, typical "Connect" patterns:
		// 1. Server: Listen on port
		// 2. Client: Dial remote (unconnected UDP or connected UDP)

		// Let's assume Address is "host:port".
		// If host is empty or 0.0.0.0, we Listen.
		// If host is remote, we Dial?
		// Modbus TCP vs RTU over UDP?
		// Let's stick to: ListenUDP for receiving.
		// And for sending... `conn.WriteToUDP` if we know destination, or `DialUDP` if we want fixed remote.

		// Simplify: We assume "connected" UDP to the configured address.
		// If we use DialUDP, we get a conn that sends to that addr and reads from that addr.
		conn, err = net.DialUDP("udp", nil, addr)
		if err != nil {
			t.state = transport.StateError
			t.lastError = err
			return err
		}
	}

	t.conn = conn
	now := time.Now()
	t.connectedAt = &now
	t.state = transport.StateConnected

	if t.config.ReadBufferSize > 0 {
		t.conn.SetReadBuffer(t.config.ReadBufferSize)
	}
	if t.config.WriteBufferSize > 0 {
		t.conn.SetWriteBuffer(t.config.WriteBufferSize)
	}

	if t.eventHandler != nil {
		t.eventHandler.OnEvent(transport.Event{
			Type:      transport.EventConnected,
			Transport: t,
			Timestamp: now,
		})
	}

	return nil
}

// Close closes the UDP connection.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == transport.StateDisconnected {
		return nil
	}

	if t.cancel != nil {
		t.cancel()
	}

	var err error
	if t.conn != nil {
		err = t.conn.Close()
		t.conn = nil
	}

	t.state = transport.StateDisconnected
	t.connectedAt = nil

	if t.eventHandler != nil {
		t.eventHandler.OnEvent(transport.Event{
			Type:      transport.EventDisconnected,
			Transport: t,
			Error:     err,
			Timestamp: time.Now(),
		})
	}

	return err
}

// IsConnected returns true if connected.
func (t *Transport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state == transport.StateConnected
}

// Send writes data to the connection.
func (t *Transport) Send(ctx context.Context, data []byte) (int, error) {
	t.mu.RLock()
	if t.state != transport.StateConnected || t.conn == nil {
		t.mu.RUnlock()
		return 0, ErrNotConnected
	}
	conn := t.conn
	t.mu.RUnlock()

	if t.config.WriteTimeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(t.config.WriteTimeout))
	}

	n, err := conn.Write(data)
	if err != nil {
		t.mu.Lock()
		t.stats.Errors++
		t.lastError = err
		t.mu.Unlock()
		return n, err
	}

	t.mu.Lock()
	t.stats.BytesSent += uint64(n)
	t.stats.MessagesSent++
	t.mu.Unlock()

	return n, nil
}

// Receive reads data from the connection.
func (t *Transport) Receive(ctx context.Context) ([]byte, error) {
	t.mu.RLock()
	if t.state != transport.StateConnected || t.conn == nil {
		t.mu.RUnlock()
		return nil, ErrNotConnected
	}
	conn := t.conn
	t.mu.RUnlock()

	if t.config.ReadTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(t.config.ReadTimeout))
	}

	n, _, err := conn.ReadFromUDP(t.readBuffer)
	if err != nil {
		t.mu.Lock()
		t.stats.Errors++
		t.lastError = err
		t.mu.Unlock()
		return nil, err
	}

	data := make([]byte, n)
	copy(data, t.readBuffer[:n])

	t.mu.Lock()
	t.stats.BytesReceived += uint64(n)
	t.stats.MessagesReceived++
	t.mu.Unlock()

	return data, nil
}

// Configure updates the transport configuration.
func (t *Transport) Configure(config transport.Config) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == transport.StateConnected {
		return errors.New("cannot reconfigure while connected")
	}

	t.tConfig = config
	return nil
}

// Info returns transport information.
func (t *Transport) Info() transport.Info {
	t.mu.RLock()
	defer t.mu.RUnlock()

	info := transport.Info{
		ID:          t.id,
		Type:        "udp",
		Address:     t.config.Address,
		State:       t.state,
		Statistics:  t.stats,
		ConnectedAt: t.connectedAt,
	}

	if t.lastError != nil {
		info.LastError = t.lastError.Error()
	}

	return info
}

// SetEventHandler sets the event handler.
func (t *Transport) SetEventHandler(handler transport.EventHandler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.eventHandler = handler
}

// Factory creates UDP transport instances.
type Factory struct{}

// NewFactory creates a new UDP transport factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Type returns the transport type.
func (f *Factory) Type() string {
	return "udp"
}

// Create creates a new UDP transport.
func (f *Factory) Create(config transport.Config) (transport.Transport, error) {
	return NewTransport(config)
}

// Validate validates the configuration.
func (f *Factory) Validate(config transport.Config) error {
	if config.Address == "" {
		return errors.New("UDP address is required")
	}
	return nil
}
