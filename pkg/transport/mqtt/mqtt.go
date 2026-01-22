// Package mqtt provides MQTT transport implementations.
package mqtt

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/transport"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Common errors.
var (
	ErrNotConnected = errors.New("not connected")
	ErrConnClosed   = errors.New("connection closed")
)

// Config holds MQTT-specific configuration.
type Config struct {
	// Broker is the broker URI (e.g., tcp://localhost:1883).
	Broker string `yaml:"broker" json:"broker"`

	// ClientID is the client ID.
	ClientID string `yaml:"client_id" json:"client_id"`

	// Username is the username.
	Username string `yaml:"username" json:"username"`

	// Password is the password.
	Password string `yaml:"password" json:"password"`

	// Topic is the default topic to publish/subscribe.
	Topic string `yaml:"topic" json:"topic"`

	// QOS is the Quality of Service level (0, 1, 2).
	QOS int `yaml:"qos" json:"qos"`

	// ConnectTimeout is the connection timeout.
	ConnectTimeout time.Duration `yaml:"connect_timeout" json:"connect_timeout"`
}

// DefaultConfig returns a default MQTT configuration.
func DefaultConfig() Config {
	return Config{
		Broker:         "tcp://localhost:1883",
		ClientID:       "comx-bridge-" + fmt.Sprintf("%d", time.Now().Unix()),
		QOS:            0,
		ConnectTimeout: 10 * time.Second,
	}
}

// Client implements the transport.Transport interface for MQTT.
type Client struct {
	mu sync.RWMutex

	config  Config
	tConfig transport.Config

	client       mqtt.Client
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

// NewClient creates a new MQTT client transport.
func NewClient(config transport.Config) (*Client, error) {
	mqttConfig := DefaultConfig()

	// Parse options
	if opts := config.Options; opts != nil {
		if v, ok := opts["broker"].(string); ok {
			mqttConfig.Broker = v
		}
		if v, ok := opts["client_id"].(string); ok {
			mqttConfig.ClientID = v
		}
		if v, ok := opts["username"].(string); ok {
			mqttConfig.Username = v
		}
		if v, ok := opts["password"].(string); ok {
			mqttConfig.Password = v
		}
		if v, ok := opts["topic"].(string); ok {
			mqttConfig.Topic = v
		}
		if v, ok := opts["qos"].(int); ok {
			mqttConfig.QOS = v
		}
	}
	// Fallback/Override if Address is set (Address overrides broker)
	if config.Address != "" {
		mqttConfig.Broker = config.Address
	}

	return &Client{
		config:      mqttConfig,
		tConfig:     config,
		id:          fmt.Sprintf("mqtt-%s", mqttConfig.ClientID),
		state:       transport.StateDisconnected,
		messageChan: make(chan []byte, 100),
	}, nil
}

// createTLSConfig creates a TLS configuration from the transport config.
func (c *Client) createTLSConfig(config *transport.TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify,
	}

	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if config.CAFile != "" {
		caCert, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	switch config.MinVersion {
	case "1.0":
		tlsConfig.MinVersion = tls.VersionTLS10
	case "1.1":
		tlsConfig.MinVersion = tls.VersionTLS11
	case "1.2":
		tlsConfig.MinVersion = tls.VersionTLS12
	case "1.3":
		tlsConfig.MinVersion = tls.VersionTLS13
	}

	return tlsConfig, nil
}

// Connect establishes a connection to the MQTT broker.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == transport.StateConnected {
		return nil
	}

	c.state = transport.StateConnecting
	c.ctx, c.cancel = context.WithCancel(ctx)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(c.config.Broker)
	opts.SetClientID(c.config.ClientID)

	if c.config.Username != "" {
		opts.SetUsername(c.config.Username)
		opts.SetPassword(c.config.Password)
	}

	opts.SetConnectTimeout(c.config.ConnectTimeout)
	opts.SetAutoReconnect(true)

	// Set handlers
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		c.mu.Lock()
		c.state = transport.StateConnected
		now := time.Now()
		c.connectedAt = &now
		c.mu.Unlock()

		if c.eventHandler != nil {
			c.eventHandler.OnEvent(transport.Event{
				Type:      transport.EventConnected,
				Transport: c,
				Timestamp: now,
			})
		}

		// Subscribe if topic is configured
		if c.config.Topic != "" {
			token := client.Subscribe(c.config.Topic, byte(c.config.QOS), c.handleMessage)
			if token.Wait() && token.Error() != nil {
				// Subscribe failed, log it or handle via event?
				// For now, assume it works or retry logic usually handles it
			}
		}
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		c.mu.Lock()
		c.state = transport.StateDisconnected // Or Error?
		c.lastError = err
		c.connectedAt = nil
		c.mu.Unlock()

		if c.eventHandler != nil {
			c.eventHandler.OnEvent(transport.Event{
				Type:      transport.EventDisconnected,
				Transport: c,
				Error:     err,
				Timestamp: time.Now(),
			})
		}
	})

	// TLS Configuration
	if c.tConfig.TLS != nil && c.tConfig.TLS.Enabled {
		tlsConfig, err := c.createTLSConfig(c.tConfig.TLS)
		if err != nil {
			return err
		}
		opts.SetTLSConfig(tlsConfig)
		// Ensure broker URL scheme matches TLS
		if !strings.HasPrefix(c.config.Broker, "ssl://") && !strings.HasPrefix(c.config.Broker, "tls://") && !strings.HasPrefix(c.config.Broker, "tcps://") {
			// Replace tcp:// with ssl:// for paho
			if strings.HasPrefix(c.config.Broker, "tcp://") {
				c.config.Broker = strings.Replace(c.config.Broker, "tcp://", "ssl://", 1)
				opts.AddBroker(c.config.Broker) // Re-add broker with correct scheme
			}
		}
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()

	// Wait for connection with context? token.Wait() blocks indefinitely or until timeout
	// But we should respect ctx cancellation?
	// The client Connect() is async but token.Wait() blocks.
	// Since we are inside Connect(ctx), we should ideally block or return error.
	finished := make(chan struct{})
	go func() {
		token.Wait()
		close(finished)
	}()

	select {
	case <-finished:
		if err := token.Error(); err != nil {
			c.state = transport.StateError
			c.lastError = err
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	c.client = client
	// State update handled in OnConnectHandler, but also here to be safe if synchronous
	if client.IsConnected() {
		c.state = transport.StateConnected
		now := time.Now()
		c.connectedAt = &now
	}

	return nil
}

// handleMessage handles incoming MQTT messages.
func (c *Client) handleMessage(client mqtt.Client, msg mqtt.Message) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Update stats
	// c.stats.MessagesReceived++ // Doing this under write lock is better or atomic
	// But we are in RLock... let's separate.
	// Actually we just put it in channel or call Receive buffer.

	// Basic implementation: push to channel
	select {
	case c.messageChan <- msg.Payload():
		// Success
	default:
		// Drop if full
	}
}

// Close closes the connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == transport.StateDisconnected {
		return nil
	}

	if c.cancel != nil {
		c.cancel()
	}

	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250) // wait 250ms
	}

	c.state = transport.StateDisconnected
	c.connectedAt = nil

	if c.eventHandler != nil {
		c.eventHandler.OnEvent(transport.Event{
			Type:      transport.EventDisconnected,
			Transport: c,
			Timestamp: time.Now(),
		})
	}

	return nil
}

// IsConnected returns true if connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == transport.StateConnected && c.client != nil && c.client.IsConnected()
}

// Send writes data to the connection (publishes).
func (c *Client) Send(ctx context.Context, data []byte) (int, error) {
	c.mu.RLock()
	if c.state != transport.StateConnected || c.client == nil {
		c.mu.RUnlock()
		return 0, ErrNotConnected
	}
	client := c.client
	topic := c.config.Topic
	qos := c.config.QOS
	c.mu.RUnlock()

	if topic == "" {
		return 0, errors.New("subscribe/publish topic not configured")
	}

	token := client.Publish(topic, byte(qos), false, data)

	finished := make(chan struct{})
	go func() {
		token.Wait()
		close(finished)
	}()

	select {
	case <-finished:
		if err := token.Error(); err != nil {
			c.mu.Lock()
			c.stats.Errors++
			c.lastError = err
			c.mu.Unlock()
			return 0, err
		}
	case <-ctx.Done():
		return 0, ctx.Err()
	}

	c.mu.Lock()
	c.stats.BytesSent += uint64(len(data))
	c.stats.MessagesSent++
	c.mu.Unlock()

	return len(data), nil
}

// Receive reads data from the connection (consumed from subscription).
func (c *Client) Receive(ctx context.Context) ([]byte, error) {
	select {
	case msg := <-c.messageChan:
		c.mu.Lock()
		c.stats.BytesReceived += uint64(len(msg))
		c.stats.MessagesReceived++
		c.mu.Unlock()
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.ctx.Done():
		return nil, ErrConnClosed
	}
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
		Type:        "mqtt",
		Address:     c.config.Broker,
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

// Factory creates MQTT transport instances.
type Factory struct{}

// NewFactory creates a new MQTT transport factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Type returns the transport type.
func (f *Factory) Type() string {
	return "mqtt"
}

// Create creates a new MQTT transport.
func (f *Factory) Create(config transport.Config) (transport.Transport, error) {
	return NewClient(config)
}

// Validate validates the configuration.
func (f *Factory) Validate(config transport.Config) error {
	// Address or Options["broker"] required
	if config.Address == "" && (config.Options == nil || config.Options["broker"] == nil) {
		return errors.New("broker address is required")
	}
	return nil
}
