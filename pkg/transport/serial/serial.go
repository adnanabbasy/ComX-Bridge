// Package serial provides a serial port transport implementation
// for RS232/RS485 communication.
package serial

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/transport"
	"go.bug.st/serial"
)

// Common errors.
var (
	ErrPortNotOpen    = errors.New("serial port not open")
	ErrInvalidConfig  = errors.New("invalid serial configuration")
	ErrReadTimeout    = errors.New("read timeout")
)

// Config holds serial-specific configuration.
type Config struct {
	// Port is the serial port path (e.g., "/dev/ttyUSB0", "COM1").
	Port string `yaml:"port" json:"port"`

	// BaudRate is the baud rate (e.g., 9600, 115200).
	BaudRate int `yaml:"baudrate" json:"baudrate"`

	// DataBits is the number of data bits (5, 6, 7, 8).
	DataBits int `yaml:"databits" json:"databits"`

	// Parity is the parity mode ("none", "odd", "even", "mark", "space").
	Parity string `yaml:"parity" json:"parity"`

	// StopBits is the number of stop bits (1, 1.5, 2).
	StopBits float64 `yaml:"stopbits" json:"stopbits"`

	// FlowControl is the flow control mode ("none", "hardware", "software").
	FlowControl string `yaml:"flow_control" json:"flow_control"`

	// ReadTimeout is the read timeout.
	ReadTimeout time.Duration `yaml:"read_timeout" json:"read_timeout"`

	// WriteTimeout is the write timeout.
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`

	// BufferSize is the read buffer size.
	BufferSize int `yaml:"buffer_size" json:"buffer_size"`

	// RS485 enables RS485 mode.
	RS485 *RS485Config `yaml:"rs485" json:"rs485"`
}

// RS485Config holds RS485-specific configuration.
type RS485Config struct {
	// Enabled enables RS485 mode.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// DelayRtsBeforeSend is the delay before sending.
	DelayRtsBeforeSend time.Duration `yaml:"delay_rts_before_send" json:"delay_rts_before_send"`

	// DelayRtsAfterSend is the delay after sending.
	DelayRtsAfterSend time.Duration `yaml:"delay_rts_after_send" json:"delay_rts_after_send"`

	// RtsHighDuringSend sets RTS high during send.
	RtsHighDuringSend bool `yaml:"rts_high_during_send" json:"rts_high_during_send"`

	// RtsHighAfterSend sets RTS high after send.
	RtsHighAfterSend bool `yaml:"rts_high_after_send" json:"rts_high_after_send"`

	// RxDuringTx enables receiving during transmission.
	RxDuringTx bool `yaml:"rx_during_tx" json:"rx_during_tx"`
}

// DefaultConfig returns a default serial configuration.
func DefaultConfig() Config {
	return Config{
		BaudRate:     9600,
		DataBits:     8,
		Parity:       "none",
		StopBits:     1,
		FlowControl:  "none",
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 1 * time.Second,
		BufferSize:   4096,
	}
}

// Transport implements the transport.Transport interface for serial ports.
type Transport struct {
	mu sync.RWMutex

	config  Config
	tConfig transport.Config

	// Port handle
	port serial.Port

	id           string
	state        transport.ConnectionState
	eventHandler transport.EventHandler
	stats        transport.Statistics

	// Internal state
	readBuffer  []byte
	connectedAt *time.Time

	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new serial transport.
func New(config transport.Config) (*Transport, error) {
	serialConfig := DefaultConfig()

	// Parse options from transport config
	if config.Address != "" {
		serialConfig.Port = config.Address
	}

	if opts := config.Options; opts != nil {
		if v, ok := opts["baudrate"].(int); ok {
			serialConfig.BaudRate = v
		}
		if v, ok := opts["databits"].(int); ok {
			serialConfig.DataBits = v
		}
		if v, ok := opts["parity"].(string); ok {
			serialConfig.Parity = v
		}
		if v, ok := opts["stopbits"].(float64); ok {
			serialConfig.StopBits = v
		}
		if v, ok := opts["flow_control"].(string); ok {
			serialConfig.FlowControl = v
		}
	}

	if config.BufferSize > 0 {
		serialConfig.BufferSize = config.BufferSize
	}
	if config.Timeout > 0 {
		serialConfig.ReadTimeout = config.Timeout
	}

	return &Transport{
		config:     serialConfig,
		tConfig:    config,
		id:         fmt.Sprintf("serial-%s", serialConfig.Port),
		state:      transport.StateDisconnected,
		readBuffer: make([]byte, serialConfig.BufferSize),
	}, nil
}

// Connect opens the serial port.
func (t *Transport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == transport.StateConnected {
		return nil
	}

	t.state = transport.StateConnecting
	t.ctx, t.cancel = context.WithCancel(ctx)

	mode := &serial.Mode{
		BaudRate: t.config.BaudRate,
		DataBits: t.config.DataBits,
		Parity:   t.parseParity(),
		StopBits: t.parseStopBits(),
	}

	port, err := serial.Open(t.config.Port, mode)
	if err != nil {
		t.state = transport.StateError
		if t.cancel != nil {
			t.cancel()
		}
		return err
	}

	// Set header timeout mainly
	if err := port.SetReadTimeout(t.config.ReadTimeout); err != nil {
		port.Close()
		return err
	}

	t.port = port

	now := time.Now()
	t.connectedAt = &now
	t.state = transport.StateConnected

	if t.eventHandler != nil {
		t.eventHandler.OnEvent(transport.Event{
			Type:      transport.EventConnected,
			Transport: t,
			Timestamp: now,
		})
	}

	return nil
}

// Close closes the serial port.
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
	if t.port != nil {
		err = t.port.Close()
		t.port = nil
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

// IsConnected returns true if the port is open.
func (t *Transport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state == transport.StateConnected
}

// Send writes data to the serial port.
func (t *Transport) Send(ctx context.Context, data []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state != transport.StateConnected || t.port == nil {
		return 0, ErrPortNotOpen
	}

	n, err := t.port.Write(data)
	if err != nil {
		t.stats.Errors++
		// If read error, maybe connection lost? handling simplistic for now
		return n, err
	}

	t.stats.BytesSent += uint64(n)
	t.stats.MessagesSent++

	return n, nil
}

// Receive reads data from the serial port.
func (t *Transport) Receive(ctx context.Context) ([]byte, error) {
	t.mu.RLock()
	if t.state != transport.StateConnected || t.port == nil {
		t.mu.RUnlock()
		return nil, ErrPortNotOpen
	}
	port := t.port
	t.mu.RUnlock()

	// Check context
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	n, err := port.Read(t.readBuffer)
	if err != nil {
		// EOF is usually not returned by serial ports unless closed, but check anyway
		if err == io.EOF {
			return nil, ErrPortNotOpen
		}
		
		// In go.bug.st/serial, read timeout might not return error but 0 bytes?
		// or it might return error. we will see.
		// For now simple return.
		return nil, err
	}

	if n == 0 {
		return nil, nil // No data
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

	// Can only reconfigure when disconnected
	if t.state == transport.StateConnected {
		return errors.New("cannot reconfigure while connected")
	}

	// Update configuration
	if config.Address != "" {
		t.config.Port = config.Address
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
		Type:        "serial",
		Address:     t.config.Port,
		State:       t.state,
		Statistics:  t.stats,
		ConnectedAt: t.connectedAt,
	}

	return info
}

// SetEventHandler sets the event handler.
func (t *Transport) SetEventHandler(handler transport.EventHandler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.eventHandler = handler
}

// parseParity converts parity string to serial.Parity.
func (t *Transport) parseParity() serial.Parity {
	switch t.config.Parity {
	case "odd":
		return serial.OddParity
	case "even":
		return serial.EvenParity
	case "mark":
		return serial.MarkParity
	case "space":
		return serial.SpaceParity
	default:
		return serial.NoParity
	}
}

// parseStopBits converts stopbits float to serial.StopBits.
func (t *Transport) parseStopBits() serial.StopBits {
	switch t.config.StopBits {
	case 1.5:
		return serial.OnePointFiveStopBits
	case 2:
		return serial.TwoStopBits
	default:
		return serial.OneStopBit
	}
}

// Factory creates serial transport instances.
type Factory struct{}

// NewFactory creates a new serial transport factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Type returns the transport type.
func (f *Factory) Type() string {
	return "serial"
}

// Create creates a new serial transport.
func (f *Factory) Create(config transport.Config) (transport.Transport, error) {
	return New(config)
}

// Validate validates the configuration.
func (f *Factory) Validate(config transport.Config) error {
	if config.Address == "" {
		return errors.New("serial port address is required")
	}
	return nil
}
