// plugins/examples/modbus-custom/main.go
package main

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/parser"
	"github.com/commatea/ComX-Bridge/pkg/protocol"
)

// MyModbusProtocol implements a custom simplified Modbus-like protocol.
type MyModbusProtocol struct {
	config protocol.Config
}

// ---------------------------------------------------------------------
// 1. Plugin Export (Required)
// ---------------------------------------------------------------------
var Plugin MyModbusProtocol

// ---------------------------------------------------------------------
// 2. Protocol Interface Implementation
// ---------------------------------------------------------------------

func (p *MyModbusProtocol) Name() string {
	return "modbus-custom"
}

func (p *MyModbusProtocol) Version() string {
	return "1.0.0"
}

func (p *MyModbusProtocol) Configure(config protocol.Config) error {
	p.config = config
	return nil
}

func (p *MyModbusProtocol) Encode(request *protocol.Request) ([]byte, error) {
	// Simple Structure: [SLAVE_ID][FUNC][ADDR_HI][ADDR_LO][COUNT_HI][COUNT_LO][CRC_LO][CRC_HI]
	// Example for Read Holding Registers (03)

	slaveID := byte(1)
	if v, ok := request.Metadata["slave_id"].(int); ok {
		slaveID = byte(v)
	}

	addr := uint16(0)
	if v, ok := request.Address.(int); ok {
		addr = uint16(v)
	}

	count := uint16(1)
	if v, ok := request.Data.(int); ok {
		count = uint16(v)
	}

	frame := make([]byte, 8)
	frame[0] = slaveID
	frame[1] = 0x03 // Fixed to Read Holding Registers for example
	binary.BigEndian.PutUint16(frame[2:], addr)
	binary.BigEndian.PutUint16(frame[4:], count)

	crc := crc16(frame[:6])
	binary.LittleEndian.PutUint16(frame[6:], crc)

	return frame, nil
}

func (p *MyModbusProtocol) Decode(data []byte) (*protocol.Response, error) {
	if len(data) < 5 {
		return nil, errors.New("packet too short")
	}

	// Validate CRC
	payloadForCRC := data[:len(data)-2]
	receivedCRC := binary.LittleEndian.Uint16(data[len(data)-2:])
	if crc16(payloadForCRC) != receivedCRC {
		return nil, errors.New("crc mismatch")
	}

	// Parse
	// [SLAVE][FUNC][BYTES][DATA...][CRC]
	byteCount := int(data[2])
	values := make([]uint16, byteCount/2)
	for i := 0; i < len(values); i++ {
		values[i] = binary.BigEndian.Uint16(data[3+i*2:])
	}

	return &protocol.Response{
		Success:   true,
		Data:      values,
		RawData:   data,
		Timestamp: time.Now(),
	}, nil
}

func (p *MyModbusProtocol) Parser() parser.Parser {
	return &CustomModbusParser{}
}

func (p *MyModbusProtocol) Validate(data []byte) error {
	if len(data) < 4 {
		return errors.New("too short")
	}
	// Simplified validation
	return nil
}

// ---------------------------------------------------------------------
// 3. Parser Implementation
// ---------------------------------------------------------------------

type CustomModbusParser struct{}

func (p *CustomModbusParser) Type() parser.Type {
	return parser.TypeCustom
}

func (p *CustomModbusParser) Parse(buffer []byte) ([]byte, []byte, error) {
	// Simplified parser logic for example
	// Minimum 5 bytes
	if len(buffer) < 5 {
		return nil, buffer, parser.ErrIncompletePacket
	}

	// Check Function Code
	funcCode := buffer[1]

	// Error Response: Fixed 5 bytes
	if funcCode&0x80 != 0 {
		return buffer[:5], buffer[5:], nil
	}

	// Normal Response (Read Holding): 3 + ByteCount + 2
	if len(buffer) >= 3 {
		byteCount := int(buffer[2])
		totalLen := 3 + byteCount + 2
		if len(buffer) >= totalLen {
			return buffer[:totalLen], buffer[totalLen:], nil
		}
	}

	return nil, buffer, parser.ErrIncompletePacket
}

func (p *CustomModbusParser) Validate(packet []byte) error {
	return nil
}

func (p *CustomModbusParser) Reset() {}

// ---------------------------------------------------------------------
// 4. Helper (CRC)
// ---------------------------------------------------------------------

func crc16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func main() {} // Required for plugin build
