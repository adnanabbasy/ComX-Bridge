// Package http provides HTTP transport implementations.
package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/transport"
)

// Common errors.
var (
	ErrNotConnected = errors.New("not connected")
)

// Config holds HTTP-specific configuration.
type Config struct {
	// URL is the target URL (client) or Bind Address (server).
	URL string `yaml:"url" json:"url"`

	// Mode is "client" or "server".
	Mode string `yaml:"mode" json:"mode"`

	// Method is the HTTP method (for client).
	Method string `yaml:"method" json:"method"`

	// Timeout is the request timeout.
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// DefaultConfig returns a default HTTP configuration.
func DefaultConfig() Config {
	return Config{
		Mode:    "client",
		Method:  "POST",
		Timeout: 30 * time.Second,
	}
}

// Transport implements the transport.Transport interface for HTTP.
type Transport struct {
	mu sync.RWMutex

	config  Config
	tConfig transport.Config

	// For Server
	server *http.Server

	// For Client
	client *http.Client

	id           string
	state        transport.ConnectionState
	eventHandler transport.EventHandler
	stats        transport.Statistics

	connectedAt *time.Time
	lastError   error

	messageChan chan []byte
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewTransport creates a new HTTP transport.
func NewTransport(config transport.Config) (*Transport, error) {
	httpConfig := DefaultConfig()

	// Parse options
	if opts := config.Options; opts != nil {
		if v, ok := opts["mode"].(string); ok {
			httpConfig.Mode = v
		}
		if v, ok := opts["method"].(string); ok {
			httpConfig.Method = v
		}
		if v, ok := opts["url"].(string); ok {
			httpConfig.URL = v
		}
	}
	// Address overrides URL
	if config.Address != "" {
		httpConfig.URL = config.Address
	}

	return &Transport{
		config:      httpConfig,
		tConfig:     config,
		id:          fmt.Sprintf("http-%s-%s", httpConfig.Mode, httpConfig.URL),
		state:       transport.StateDisconnected,
		messageChan: make(chan []byte, 100),
	}, nil
}

// Connect establishes the HTTP transport.
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

	return t.setupClient(ctx)
}

func (t *Transport) setupClient(ctx context.Context) error {
	t.client = &http.Client{
		Timeout: t.config.Timeout,
	}

	t.onConnected()
	return nil
}

func (t *Transport) startServer(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" && r.Method != "PUT" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		t.mu.RLock()
		// Only accept if running
		if t.state != transport.StateConnected {
			t.mu.RUnlock()
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		t.mu.RUnlock()

		select {
		case t.messageChan <- body:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusServiceUnavailable) // Buffer full
		}

		t.mu.Lock()
		t.stats.BytesReceived += uint64(len(body))
		t.stats.MessagesReceived++
		t.mu.Unlock()
	})

	server := &http.Server{
		Addr:    t.config.URL, // assumes URL is bind address e.g. ":8081"
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

	t.onConnected()
	return nil
}

func (t *Transport) onConnected() {
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

// Close closes the transport.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == transport.StateDisconnected {
		return nil
	}

	if t.cancel != nil {
		t.cancel()
	}

	if t.server != nil {
		t.server.Close()
		t.server = nil
	}

	t.client = nil

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

// Send writes data (client: HTTP Request, server: No-op/Response?).
// For simplicity, Send in Client mode sends data to URL.
// In Server mode, Send is not well-defined for simple HTTP server unless we have a specific response context or webhook target.
// Let's assume sending means "Send to configured URL" regardless of mode, or error in Server mode.
// Actually, bidirectional HTTP is usually Client->Server (req), Server->Client (resp).
// But "Send" here is async.
// If Mode == Server, Send is probably invalid or means sending a webhook out?
// Let's implement Client Send.
func (t *Transport) Send(ctx context.Context, data []byte) (int, error) {
	t.mu.RLock()
	mode := t.config.Mode
	client := t.client
	url := t.config.URL
	method := t.config.Method
	t.mu.RUnlock()

	if mode == "server" {
		return 0, errors.New("HTTP server mode cannot initiate Send (use client mode or WebHooks)")
	}

	if client == nil {
		return 0, ErrNotConnected
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		t.mu.Lock()
		t.stats.Errors++
		t.lastError = err
		t.mu.Unlock()
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("http error: %s", resp.Status)
	}

	// Read response body if any and treat as "Receive"?
	// If transport is strictly request-response, Receive might be correlated.
	// But `Receive` method pulls from a channel.
	// We can push response to messageChan.
	respBody, _ := io.ReadAll(resp.Body)
	if len(respBody) > 0 {
		select {
		case t.messageChan <- respBody:
		default:
		}
		t.mu.Lock()
		t.stats.BytesReceived += uint64(len(respBody))
		t.stats.MessagesReceived++
		t.mu.Unlock()
	}

	t.mu.Lock()
	t.stats.BytesSent += uint64(len(data))
	t.stats.MessagesSent++
	t.mu.Unlock()

	return len(data), nil
}

// Receive reads data.
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
		Type:        "http",
		Address:     t.config.URL,
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

// Factory creates HTTP transport instances.
type Factory struct{}

// NewFactory creates a new HTTP transport factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Type returns the transport type.
func (f *Factory) Type() string {
	return "http"
}

// Create creates a new HTTP transport.
func (f *Factory) Create(config transport.Config) (transport.Transport, error) {
	return NewTransport(config)
}

// Validate validates the configuration.
func (f *Factory) Validate(config transport.Config) error {
	if config.Address == "" && (config.Options == nil || config.Options["url"] == nil) {
		return errors.New("url is required")
	}
	return nil
}
