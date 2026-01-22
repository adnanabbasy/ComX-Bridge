// Package tcp provides TCP client and server transport implementations.
package tcp

import (
	"context"
	"errors"
	"fmt"
	"io"
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

// Config holds TCP-specific configuration.
type Config struct {
	// Host is the remote host.
	Host string `yaml:"host" json:"host"`

	// Port is the remote port.
	Port int `yaml:"port" json:"port"`

	// KeepAlive enables TCP keepalive.
	KeepAlive bool `yaml:"keepalive" json:"keepalive"`

	// KeepAlivePeriod is the keepalive interval.
	KeepAlivePeriod time.Duration `yaml:"keepalive_period" json:"keepalive_period"`

	// NoDelay disables Nagle's algorithm.
	NoDelay bool `yaml:"no_delay" json:"no_delay"`

	// ReadBufferSize is the read buffer size.
	ReadBufferSize int `yaml:"read_buffer_size" json:"read_buffer_size"`

	// WriteBufferSize is the write buffer size.
	WriteBufferSize int `yaml:"write_buffer_size" json:"write_buffer_size"`

	// ConnectTimeout is the connection timeout.
	ConnectTimeout time.Duration `yaml:"connect_timeout" json:"connect_timeout"`

	// ReadTimeout is the read timeout.
	ReadTimeout time.Duration `yaml:"read_timeout" json:"read_timeout"`

	// WriteTimeout is the write timeout.
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`

	// TLS enables TLS encryption.
	TLS *TLSConfig `yaml:"tls" json:"tls"`
}

// TLSConfig holds TLS configuration.
type TLSConfig struct {
	// Enabled enables TLS.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// InsecureSkipVerify skips certificate verification.
	InsecureSkipVerify bool `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`

	// CertFile is the client certificate file.
	CertFile string `yaml:"cert_file" json:"cert_file"`

	// KeyFile is the client key file.
	KeyFile string `yaml:"key_file" json:"key_file"`

	// CAFile is the CA certificate file.
	CAFile string `yaml:"ca_file" json:"ca_file"`
}

// DefaultConfig returns a default TCP configuration.
func DefaultConfig() Config {
	return Config{
		KeepAlive:       true,
		KeepAlivePeriod: 30 * time.Second,
		NoDelay:         true,
		ReadBufferSize:  8192,
		WriteBufferSize: 8192,
		ConnectTimeout:  10 * time.Second,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    10 * time.Second,
	}
}

// Client implements the transport.Transport interface for TCP clients.
type Client struct {
	mu sync.RWMutex

	config  Config
	tConfig transport.Config

	conn         net.Conn
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

// NewClient creates a new TCP client transport.
func NewClient(config transport.Config) (*Client, error) {
	tcpConfig := DefaultConfig()

	// Parse address
	if config.Address != "" {
		host, port, err := net.SplitHostPort(config.Address)
		if err == nil {
			tcpConfig.Host = host
			fmt.Sscanf(port, "%d", &tcpConfig.Port)
		}
	}

	// Parse options
	if opts := config.Options; opts != nil {
		if v, ok := opts["keepalive"].(bool); ok {
			tcpConfig.KeepAlive = v
		}
		if v, ok := opts["no_delay"].(bool); ok {
			tcpConfig.NoDelay = v
		}
		if v, ok := opts["connect_timeout"].(string); ok {
			if d, err := time.ParseDuration(v); err == nil {
				tcpConfig.ConnectTimeout = d
			}
		}
	}

	if config.Timeout > 0 {
		tcpConfig.ReadTimeout = config.Timeout
	}
	if config.BufferSize > 0 {
		tcpConfig.ReadBufferSize = config.BufferSize
	}

	return &Client{
		config:     tcpConfig,
		tConfig:    config,
		id:         fmt.Sprintf("tcp-client-%s:%d", tcpConfig.Host, tcpConfig.Port),
		state:      transport.StateDisconnected,
		readBuffer: make([]byte, tcpConfig.ReadBufferSize),
	}, nil
}

// Connect establishes a TCP connection.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == transport.StateConnected {
		return nil
	}

	c.state = transport.StateConnecting
	c.ctx, c.cancel = context.WithCancel(ctx)

	address := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout:   c.config.ConnectTimeout,
		KeepAlive: c.config.KeepAlivePeriod,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		c.state = transport.StateError
		c.lastError = err
		return err
	}

	// Configure connection
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if c.config.KeepAlive {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(c.config.KeepAlivePeriod)
		}
		tcpConn.SetNoDelay(c.config.NoDelay)
		tcpConn.SetReadBuffer(c.config.ReadBufferSize)
		tcpConn.SetWriteBuffer(c.config.WriteBufferSize)
	}

	c.conn = conn
	now := time.Now()
	c.connectedAt = &now
	c.state = transport.StateConnected

	if c.eventHandler != nil {
		c.eventHandler.OnEvent(transport.Event{
			Type:      transport.EventConnected,
			Transport: c,
			Timestamp: now,
		})
	}

	return nil
}

// Close closes the TCP connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == transport.StateDisconnected {
		return nil
	}

	if c.cancel != nil {
		c.cancel()
	}

	var err error
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
	}

	c.state = transport.StateDisconnected
	c.connectedAt = nil

	if c.eventHandler != nil {
		c.eventHandler.OnEvent(transport.Event{
			Type:      transport.EventDisconnected,
			Transport: c,
			Error:     err,
			Timestamp: time.Now(),
		})
	}

	return err
}

// IsConnected returns true if connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == transport.StateConnected
}

// Send writes data to the connection.
func (c *Client) Send(ctx context.Context, data []byte) (int, error) {
	c.mu.RLock()
	if c.state != transport.StateConnected || c.conn == nil {
		c.mu.RUnlock()
		return 0, ErrNotConnected
	}
	conn := c.conn
	c.mu.RUnlock()

	// Set write deadline
	if c.config.WriteTimeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
	}

	n, err := conn.Write(data)
	if err != nil {
		c.mu.Lock()
		c.stats.Errors++
		c.lastError = err
		c.mu.Unlock()

		if c.eventHandler != nil {
			c.eventHandler.OnEvent(transport.Event{
				Type:      transport.EventError,
				Transport: c,
				Error:     err,
				Timestamp: time.Now(),
			})
		}
		return n, err
	}

	c.mu.Lock()
	c.stats.BytesSent += uint64(n)
	c.stats.MessagesSent++
	c.mu.Unlock()

	return n, nil
}

// Receive reads data from the connection.
func (c *Client) Receive(ctx context.Context) ([]byte, error) {
	c.mu.RLock()
	if c.state != transport.StateConnected || c.conn == nil {
		c.mu.RUnlock()
		return nil, ErrNotConnected
	}
	conn := c.conn
	c.mu.RUnlock()

	// Set read deadline
	if c.config.ReadTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
	}

	n, err := conn.Read(c.readBuffer)
	if err != nil {
		if err == io.EOF {
			return nil, ErrConnClosed
		}
		c.mu.Lock()
		c.stats.Errors++
		c.lastError = err
		c.mu.Unlock()
		return nil, err
	}

	data := make([]byte, n)
	copy(data, c.readBuffer[:n])

	c.mu.Lock()
	c.stats.BytesReceived += uint64(n)
	c.stats.MessagesReceived++
	c.mu.Unlock()

	return data, nil
}

// Configure updates the transport configuration.
func (c *Client) Configure(config transport.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == transport.StateConnected {
		return errors.New("cannot reconfigure while connected")
	}

	c.tConfig = config
	return nil
}

// Info returns transport information.
func (c *Client) Info() transport.Info {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info := transport.Info{
		ID:          c.id,
		Type:        "tcp",
		Address:     fmt.Sprintf("%s:%d", c.config.Host, c.config.Port),
		State:       c.state,
		Statistics:  c.stats,
		ConnectedAt: c.connectedAt,
	}

	if c.lastError != nil {
		info.LastError = c.lastError.Error()
	}

	return info
}

// SetEventHandler sets the event handler.
func (c *Client) SetEventHandler(handler transport.EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.eventHandler = handler
}

// Factory creates TCP transport instances.
type Factory struct{}

// NewFactory creates a new TCP transport factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Type returns the transport type.
func (f *Factory) Type() string {
	return "tcp"
}

// Create creates a new TCP transport.
func (f *Factory) Create(config transport.Config) (transport.Transport, error) {
	return NewClient(config)
}

// Validate validates the configuration.
func (f *Factory) Validate(config transport.Config) error {
	if config.Address == "" {
		return errors.New("TCP address is required (host:port)")
	}

	_, _, err := net.SplitHostPort(config.Address)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	return nil
}
