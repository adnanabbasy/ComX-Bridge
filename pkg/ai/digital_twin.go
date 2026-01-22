// Package ai - Digital Twin
// Provides device simulation and virtual testing capabilities.
package ai

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TwinConfig holds configuration for digital twin.
type TwinConfig struct {
	// Enabled enables digital twin simulation.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// DeviceID is the ID of the device being simulated.
	DeviceID string `yaml:"device_id" json:"device_id"`

	// DeviceType is the type of device (modbus, serial, etc.).
	DeviceType string `yaml:"device_type" json:"device_type"`

	// SimulationMode defines how the twin behaves.
	SimulationMode SimulationMode `yaml:"simulation_mode" json:"simulation_mode"`

	// ResponseDelay is the simulated response delay.
	ResponseDelay time.Duration `yaml:"response_delay" json:"response_delay"`

	// ErrorRate is the simulated error rate (0.0 - 1.0).
	ErrorRate float64 `yaml:"error_rate" json:"error_rate"`
}

// SimulationMode defines how the digital twin operates.
type SimulationMode int

const (
	// SimModeReplay replays recorded data.
	SimModeReplay SimulationMode = iota

	// SimModeRandom generates random responses.
	SimModeRandom

	// SimModeModel uses a learned model.
	SimModeModel

	// SimModeScript uses a custom script.
	SimModeScript
)

// DigitalTwin simulates a device for virtual testing.
type DigitalTwin struct {
	mu     sync.RWMutex
	config TwinConfig

	// Device state
	registers map[int]uint16    // Modbus-style registers
	coils     map[int]bool      // Modbus-style coils
	memory    map[string][]byte // Generic memory

	// Recording
	recordedData []RecordedPacket
	recordIndex  int

	// Statistics
	requestCount  int64
	responseCount int64
	errorCount    int64

	// State
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// RecordedPacket represents a recorded request/response pair.
type RecordedPacket struct {
	Timestamp time.Time     `json:"timestamp"`
	Request   []byte        `json:"request"`
	Response  []byte        `json:"response"`
	Delay     time.Duration `json:"delay"`
}

// NewDigitalTwin creates a new digital twin.
func NewDigitalTwin(config TwinConfig) *DigitalTwin {
	return &DigitalTwin{
		config:    config,
		registers: make(map[int]uint16),
		coils:     make(map[int]bool),
		memory:    make(map[string][]byte),
	}
}

// Start starts the digital twin.
func (dt *DigitalTwin) Start(ctx context.Context) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if dt.running {
		return nil
	}

	dt.ctx, dt.cancel = context.WithCancel(ctx)
	dt.running = true

	return nil
}

// Stop stops the digital twin.
func (dt *DigitalTwin) Stop() error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if !dt.running {
		return nil
	}

	if dt.cancel != nil {
		dt.cancel()
	}
	dt.running = false

	return nil
}

// ProcessRequest simulates processing a request.
func (dt *DigitalTwin) ProcessRequest(ctx context.Context, request []byte) ([]byte, error) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.requestCount++

	// Simulate delay
	if dt.config.ResponseDelay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(dt.config.ResponseDelay):
		}
	}

	// Simulate errors
	if dt.config.ErrorRate > 0 {
		// Simple deterministic error based on request count
		if float64(dt.requestCount%100)/100.0 < dt.config.ErrorRate {
			dt.errorCount++
			return nil, fmt.Errorf("simulated error")
		}
	}

	var response []byte
	var err error

	switch dt.config.SimulationMode {
	case SimModeReplay:
		response, err = dt.replayResponse(request)
	case SimModeRandom:
		response, err = dt.randomResponse(request)
	case SimModeModel:
		response, err = dt.modelResponse(request)
	default:
		response, err = dt.echoResponse(request)
	}

	if err == nil {
		dt.responseCount++
	}

	return response, err
}

// SetRegister sets a Modbus-style register value.
func (dt *DigitalTwin) SetRegister(address int, value uint16) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.registers[address] = value
}

// GetRegister gets a Modbus-style register value.
func (dt *DigitalTwin) GetRegister(address int) uint16 {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.registers[address]
}

// SetCoil sets a Modbus-style coil value.
func (dt *DigitalTwin) SetCoil(address int, value bool) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.coils[address] = value
}

// GetCoil gets a Modbus-style coil value.
func (dt *DigitalTwin) GetCoil(address int) bool {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.coils[address]
}

// SetMemory sets a memory value.
func (dt *DigitalTwin) SetMemory(key string, value []byte) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.memory[key] = value
}

// GetMemory gets a memory value.
func (dt *DigitalTwin) GetMemory(key string) []byte {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.memory[key]
}

// RecordPacket records a packet for replay mode.
func (dt *DigitalTwin) RecordPacket(request, response []byte, delay time.Duration) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.recordedData = append(dt.recordedData, RecordedPacket{
		Timestamp: time.Now(),
		Request:   request,
		Response:  response,
		Delay:     delay,
	})
}

// ClearRecordings clears recorded data.
func (dt *DigitalTwin) ClearRecordings() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.recordedData = nil
	dt.recordIndex = 0
}

// GetStats returns twin statistics.
func (dt *DigitalTwin) GetStats() TwinStats {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	return TwinStats{
		RequestCount:  dt.requestCount,
		ResponseCount: dt.responseCount,
		ErrorCount:    dt.errorCount,
		RegisterCount: len(dt.registers),
		CoilCount:     len(dt.coils),
		RecordedCount: len(dt.recordedData),
	}
}

// TwinStats contains digital twin statistics.
type TwinStats struct {
	RequestCount  int64 `json:"request_count"`
	ResponseCount int64 `json:"response_count"`
	ErrorCount    int64 `json:"error_count"`
	RegisterCount int   `json:"register_count"`
	CoilCount     int   `json:"coil_count"`
	RecordedCount int   `json:"recorded_count"`
}

// replayResponse returns a recorded response.
func (dt *DigitalTwin) replayResponse(request []byte) ([]byte, error) {
	if len(dt.recordedData) == 0 {
		return nil, fmt.Errorf("no recorded data available")
	}

	// Find matching request or return next in sequence
	for _, rec := range dt.recordedData {
		if bytesEqual(rec.Request, request) {
			return rec.Response, nil
		}
	}

	// Fallback: return next recorded response
	record := dt.recordedData[dt.recordIndex]
	dt.recordIndex = (dt.recordIndex + 1) % len(dt.recordedData)

	return record.Response, nil
}

// randomResponse generates a random response.
func (dt *DigitalTwin) randomResponse(request []byte) ([]byte, error) {
	// Generate response with same length as request
	response := make([]byte, len(request))
	for i := range response {
		// Simple pseudo-random based on request and time
		response[i] = byte((int(request[i]) + int(time.Now().UnixNano())) % 256)
	}
	return response, nil
}

// modelResponse uses a learned model (placeholder for ML integration).
func (dt *DigitalTwin) modelResponse(request []byte) ([]byte, error) {
	// This is a placeholder for future ML model integration
	// For now, return an echo response
	return dt.echoResponse(request)
}

// echoResponse returns the request as the response.
func (dt *DigitalTwin) echoResponse(request []byte) ([]byte, error) {
	response := make([]byte, len(request))
	copy(response, request)
	return response, nil
}

// bytesEqual compares two byte slices.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// SimulateModbusDevice creates a digital twin configured for Modbus simulation.
func SimulateModbusDevice(slaveID int) *DigitalTwin {
	twin := NewDigitalTwin(TwinConfig{
		Enabled:        true,
		DeviceID:       fmt.Sprintf("modbus-slave-%d", slaveID),
		DeviceType:     "modbus",
		SimulationMode: SimModeModel,
		ResponseDelay:  5 * time.Millisecond,
		ErrorRate:      0.0,
	})

	// Initialize some default registers
	for i := 0; i < 100; i++ {
		twin.SetRegister(i, uint16(i*100))
	}

	// Initialize some default coils
	for i := 0; i < 16; i++ {
		twin.SetCoil(i, i%2 == 0)
	}

	return twin
}

// ProcessModbusRequest processes a Modbus request and returns response.
func (dt *DigitalTwin) ProcessModbusRequest(functionCode byte, address, quantity int) ([]byte, error) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.requestCount++

	switch functionCode {
	case 0x03: // Read Holding Registers
		data := make([]byte, quantity*2)
		for i := 0; i < quantity; i++ {
			value := dt.registers[address+i]
			data[i*2] = byte(value >> 8)
			data[i*2+1] = byte(value & 0xFF)
		}
		dt.responseCount++
		return data, nil

	case 0x01: // Read Coils
		byteCount := (quantity + 7) / 8
		data := make([]byte, byteCount)
		for i := 0; i < quantity; i++ {
			if dt.coils[address+i] {
				data[i/8] |= 1 << (i % 8)
			}
		}
		dt.responseCount++
		return data, nil

	case 0x06: // Write Single Register
		dt.registers[address] = uint16(quantity) // quantity holds the value
		dt.responseCount++
		return []byte{byte(address >> 8), byte(address), byte(quantity >> 8), byte(quantity)}, nil

	case 0x05: // Write Single Coil
		dt.coils[address] = quantity != 0
		dt.responseCount++
		return []byte{byte(address >> 8), byte(address), byte(quantity >> 8), byte(quantity)}, nil

	default:
		dt.errorCount++
		return nil, fmt.Errorf("unsupported function code: %d", functionCode)
	}
}
