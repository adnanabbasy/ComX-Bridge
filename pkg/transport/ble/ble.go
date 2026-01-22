// Package ble provides Bluetooth Low Energy transport implementations.
package ble

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/transport"
	"tinygo.org/x/bluetooth"
)

// Common errors.
var (
	ErrNotConnected = errors.New("not connected")
	ErrNotFound     = errors.New("device not found")
)

// Config holds BLE-specific configuration.
type Config struct {
	// DeviceName is the target device name to scan for.
	DeviceName string `yaml:"device_name" json:"device_name"`

	// DeviceID is the target device MAC/UUID (optional, overrides name if present).
	DeviceID string `yaml:"device_id" json:"device_id"`

	// ServiceUUID is the service UUID to use.
	ServiceUUID string `yaml:"service_uuid" json:"service_uuid"`

	// CharacteristicUUID is the characteristic UUID for Read/Write.
	CharacteristicUUID string `yaml:"characteristic_uuid" json:"characteristic_uuid"`

	// ScanTimeout is the scanning timeout.
	ScanTimeout time.Duration `yaml:"scan_timeout" json:"scan_timeout"`
}

// DefaultConfig returns a default BLE configuration.
func DefaultConfig() Config {
	return Config{
		ScanTimeout: 10 * time.Second,
	}
}

// Transport implements the transport.Transport interface for Real BLE.
type Transport struct {
	mu sync.RWMutex

	config  Config
	tConfig transport.Config

	id           string
	state        transport.ConnectionState
	eventHandler transport.EventHandler
	stats        transport.Statistics

	connectedAt *time.Time
	lastError   error

	messageChan chan []byte
	ctx         context.Context
	cancel      context.CancelFunc

	// Real BLE State
	adapter        *bluetooth.Adapter
	device         *bluetooth.Device
	service        *bluetooth.DeviceService
	characteristic *bluetooth.DeviceCharacteristic
}

// NewTransport creates a new BLE transport.
func NewTransport(config transport.Config) (*Transport, error) {
	bleConfig := DefaultConfig()

	// Parse options
	if opts := config.Options; opts != nil {
		if v, ok := opts["device_name"].(string); ok {
			bleConfig.DeviceName = v
		}
		if v, ok := opts["device_id"].(string); ok {
			bleConfig.DeviceID = v
		}
		if v, ok := opts["service_uuid"].(string); ok {
			bleConfig.ServiceUUID = v
		}
		if v, ok := opts["characteristic_uuid"].(string); ok {
			bleConfig.CharacteristicUUID = v
		}
	}

	// Validate config
	if bleConfig.ServiceUUID == "" || bleConfig.CharacteristicUUID == "" {
		return nil, errors.New("service_uuid and characteristic_uuid are required")
	}

	return &Transport{
		config:      bleConfig,
		tConfig:     config,
		id:          fmt.Sprintf("ble-%s", bleConfig.DeviceName),
		state:       transport.StateDisconnected,
		messageChan: make(chan []byte, 100),
		adapter:     bluetooth.DefaultAdapter,
	}, nil
}

// Connect establishes the BLE connection (Scan & Connect).
func (t *Transport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state == transport.StateConnected {
		return nil
	}

	// Enable BLE Adapter
	if err := t.adapter.Enable(); err != nil {
		return fmt.Errorf("failed to enable BLE adapter: %w", err)
	}

	t.state = transport.StateConnecting
	t.ctx, t.cancel = context.WithCancel(ctx)

	// Scan for device
	var foundDevice bluetooth.ScanResult
	found := false

	// Create a channel to signal scan completion
	scanDone := make(chan struct{})

	// Start scanning
	fmt.Printf("[BLE] Scanning for device: %s (ID: %s)...\n", t.config.DeviceName, t.config.DeviceID)

	err := t.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if found {
			return
		}

		if t.config.DeviceID != "" && result.Address.String() == t.config.DeviceID {
			foundDevice = result
			found = true
			adapter.StopScan()
			close(scanDone)
			return
		}

		if t.config.DeviceName != "" && result.LocalName() == t.config.DeviceName {
			foundDevice = result
			found = true
			adapter.StopScan()
			close(scanDone)
			return
		}
	})

	if err != nil {
		t.state = transport.StateError
		return fmt.Errorf("failed to start scan: %w", err)
	}

	// Wait for scan result or timeout
	select {
	case <-scanDone:
		// Device found
	case <-time.After(t.config.ScanTimeout):
		t.adapter.StopScan()
		t.state = transport.StateDisconnected
		return fmt.Errorf("scan timeout: device not found")
	case <-ctx.Done():
		t.adapter.StopScan()
		t.state = transport.StateDisconnected
		return ctx.Err()
	}

	if !found {
		return ErrNotFound
	}

	// Connect to device
	fmt.Printf("[BLE] Connecting to %s...\n", foundDevice.Address.String())
	device, err := t.adapter.Connect(foundDevice.Address, bluetooth.ConnectionParams{})
	if err != nil {
		t.state = transport.StateError
		return fmt.Errorf("failed to connect: %w", err)
	}
	t.device = &device

	// Discover Services
	fmt.Printf("[BLE] Discovering services...\n")
	srvUUID, _ := bluetooth.ParseUUID(t.config.ServiceUUID)
	services, err := device.DiscoverServices([]bluetooth.UUID{srvUUID})
	if err != nil || len(services) == 0 {
		device.Disconnect()
		return fmt.Errorf("failed to discover service %s: %w", t.config.ServiceUUID, err)
	}
	t.service = &services[0]

	// Discover Characteristics
	fmt.Printf("[BLE] Discovering characteristics...\n")
	charUUID, _ := bluetooth.ParseUUID(t.config.CharacteristicUUID)
	chars, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{charUUID})
	if err != nil || len(chars) == 0 {
		device.Disconnect()
		return fmt.Errorf("failed to discover characteristic %s: %w", t.config.CharacteristicUUID, err)
	}
	t.characteristic = &chars[0]

	// Enable Notifications
	fmt.Printf("[BLE] Enabling notifications...\n")
	err = t.characteristic.EnableNotifications(func(buf []byte) {
		// Copy data to avoid race conditions if buf is reused
		data := make([]byte, len(buf))
		copy(data, buf)

		select {
		case t.messageChan <- data:
			t.mu.Lock()
			t.stats.BytesReceived += uint64(len(data))
			t.stats.MessagesReceived++
			t.mu.Unlock()
		default:
			// Buffer full, drop
		}
	})

	if err != nil {
		fmt.Printf("[BLE] Warning: Failed to enable notifications: %v\n", err)
		// We don't fail here, maybe it's write-only or user only initiates reads (though Receive() implies waiting)
	}

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

	return nil
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

	if t.device != nil {
		t.device.Disconnect()
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

// Send writes data to the characteristic.
func (t *Transport) Send(ctx context.Context, data []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.state != transport.StateConnected {
		return 0, ErrNotConnected
	}

	if t.characteristic == nil {
		return 0, errors.New("characteristic not found")
	}

	// Try WriteWithoutResponse first for throughput, if fails try Write
	// Actually, standard Write is safer for reliability.
	// tinygo/bluetooth usually exposes Write returning int, error
	n, err := t.characteristic.Write(data)
	if err != nil {
		return 0, err
	}

	t.stats.BytesSent += uint64(n)
	t.stats.MessagesSent++

	return n, nil
}

// Receive reads data (Notification/Indication).
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
		Type:        "ble",
		Address:     t.config.DeviceName,
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

// Factory creates BLE transport instances.
type Factory struct{}

// NewFactory creates a new BLE transport factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Type returns the transport type.
func (f *Factory) Type() string {
	return "ble"
}

// Create creates a new BLE transport.
func (f *Factory) Create(config transport.Config) (transport.Transport, error) {
	return NewTransport(config)
}

// Validate validates the configuration.
func (f *Factory) Validate(config transport.Config) error {
	// Basic validation
	return nil
}
