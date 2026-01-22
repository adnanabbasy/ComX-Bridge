// Package websocket provides WebSocket transport implementations.
package websocket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/transport"
	"github.com/gorilla/websocket"
)

// Common errors.
var (
	ErrNotConnected = errors.New("not connected")
)

// Config holds WebSocket-specific configuration.
type Config struct {
	// Address is the URL (client) or Bind Address (server).
	Address string `yaml:"address" json:"address"`

	// Mode is "client" or "server".
	Mode string `yaml:"mode" json:"mode"`

	// Path is the websocket path (e.g. /ws) for server mode.
	Path string `yaml:"path" json:"path"`

	// ReadBufferSize is the read buffer size.
	ReadBufferSize int `yaml:"read_buffer_size" json:"read_buffer_size"`

	// WriteBufferSize is the write buffer size.
	WriteBufferSize int `yaml:"write_buffer_size" json:"write_buffer_size"`
}

// DefaultConfig returns a default WebSocket configuration.
func DefaultConfig() Config {
	return Config{
		Mode:            "client",
		Address:         "ws://localhost:8080/ws",
		Path:            "/ws",
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}

// Transport implements the transport.Transport interface for WebSocket.
type Transport struct {
	mu sync.RWMutex

	config  Config
	tConfig transport.Config

	conn         *websocket.Conn
	id           string
	state        transport.ConnectionState
	eventHandler transport.EventHandler
	stats        transport.Statistics

	connectedAt *time.Time
	lastError   error

	messageChan chan []byte
	ctx         context.Context
	cancel      context.CancelFunc

	// Server specific
	server *http.Server
}

// NewTransport creates a new WebSocket transport.
func NewTransport(config transport.Config) (*Transport, error) {
	wsConfig := DefaultConfig()

	// Parse options
	if opts := config.Options; opts != nil {
		if v, ok := opts["mode"].(string); ok {
			wsConfig.Mode = v
		}
		if v, ok := opts["path"].(string); ok {
			wsConfig.Path = v
		}
	}
	if config.Address != "" {
		wsConfig.Address = config.Address
	}

	return &Transport{
		config:      wsConfig,
		tConfig:     config,
		id:          fmt.Sprintf("ws-%s-%s", wsConfig.Mode, wsConfig.Address),
		state:       transport.StateDisconnected,
		messageChan: make(chan []byte, 100),
	}, nil
}

// Connect establishes a WebSocket connection (or starts server).
func (t *Transport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == transport.StateConnected {
		return nil
	}

	t.state = transport.StateConnecting
	t.ctx, t.cancel = context.WithCancel(ctx)

	if t.config.Mode == "server" {
		return t.startServer(ctx)
	}

	return t.connectClient(ctx)
}

func (t *Transport) connectClient(ctx context.Context) error {
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, t.config.Address, nil)
	if err != nil {
		t.state = transport.StateError
		t.lastError = err
		return err
	}

	t.conn = conn
	t.onConnected()

	// Start reading loop
	go t.readLoop()

	return nil
}

func (t *Transport) startServer(ctx context.Context) error {
	// For server mode, we need to start an HTTP server and upgrade connections.
	// NOTE: This simple implementation only supports ONE client connection at a time for simplicity
	// matching the Transport interface's single-stream nature.

	mux := http.NewServeMux()
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	mux.HandleFunc(t.config.Path, func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		t.mu.Lock()
		if t.conn != nil {
			conn.Close() // Reject if already connected
			t.mu.Unlock()
			return
		}
		t.conn = conn
		t.mu.Unlock()

		t.onConnected()
		go t.readLoop()
	})

	server := &http.Server{
		Addr:    t.config.Address, // e.g. :8080
		Handler: mux,
	}
	t.server = server

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.mu.Lock()
			t.state = transport.StateError
			t.lastError = err
			t.mu.Unlock()
		}
	}()

	// Server started listening, but strictly speaking "Connected" means a client connected?
	// Transport interface suggests Connect() creates the channel.
	// For server, Connect() usually means "Start listening".
	// But Send() requires an active connection.
	// We'll mark as Connecting until a client connects?
	// Or mark as Connected (Listening) but Send fails?
	// Let's mark Connected as Listening.

	t.state = transport.StateConnected
	return nil
}

func (t *Transport) onConnected() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.state = transport.StateConnected
	now := time.Now()
	t.connectedAt = &now

	if t.eventHandler != nil {
		t.eventHandler.OnEvent(transport.Event{
			Type:      transport.EventConnected,
			Transport: t,
			Timestamp: now,
		})
	}
}

func (t *Transport) readLoop() {
	defer t.Close()

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			if t.conn == nil {
				return
			}
			_, message, err := t.conn.ReadMessage()
			if err != nil {
				t.mu.Lock()
				t.lastError = err
				t.mu.Unlock()
				return
			}

			select {
			case t.messageChan <- message:
			default:
				// Drop
			}

			t.mu.Lock()
			t.stats.BytesReceived += uint64(len(message))
			t.stats.MessagesReceived++
			t.mu.Unlock()
		}
	}
}

// Close closes the connection.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == transport.StateDisconnected {
		return nil
	}

	if t.cancel != nil {
		t.cancel()
	}

	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}

	if t.server != nil {
		t.server.Close()
		t.server = nil
	}

	t.state = transport.StateDisconnected
	t.connectedAt = nil

	if t.eventHandler != nil {
		t.eventHandler.OnEvent(transport.Event{
			Type:      transport.EventDisconnected,
			Transport: t,
			Timestamp: time.Now(),
		})
	}

	return nil
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

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err := conn.WriteMessage(websocket.TextMessage, data) // Or BinaryMessage depending on config
	if err != nil {
		t.mu.Lock()
		t.stats.Errors++
		t.lastError = err
		t.mu.Unlock()
		return 0, err
	}

	t.mu.Lock()
	t.stats.BytesSent += uint64(len(data))
	t.stats.MessagesSent++
	t.mu.Unlock()

	return len(data), nil
}

// Receive reads data from the connection.
func (t *Transport) Receive(ctx context.Context) ([]byte, error) {
	select {
	case msg := <-t.messageChan:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-t.ctx.Done():
		return nil, ErrNotConnected
	}
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
		Type:        "websocket",
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

// Factory creates WebSocket transport instances.
type Factory struct{}

// NewFactory creates a new WebSocket transport factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Type returns the transport type.
func (f *Factory) Type() string {
	return "websocket"
}

// Create creates a new WebSocket transport.
func (f *Factory) Create(config transport.Config) (transport.Transport, error) {
	return NewTransport(config)
}

// Validate validates the configuration.
func (f *Factory) Validate(config transport.Config) error {
	if config.Address == "" {
		return errors.New("address is required")
	}
	return nil
}
