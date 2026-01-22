package modbus

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/parser"
	"github.com/commatea/ComX-Bridge/pkg/protocol"
	"github.com/commatea/ComX-Bridge/pkg/utils/crc"
)

// RTUProtocol implements Modbus RTU.
type RTUProtocol struct {
	config protocol.Config
	parser parser.Parser
}

// NewRTU creates a new RTU protocol instance.
func NewRTU(config protocol.Config) (protocol.Protocol, error) {
	return &RTUProtocol{
		config: config,
		// For now simple parser, but realistically we need a robust RTU parser.
		// We'll use a custom parser defined here.
		parser: &RTUParser{},
	}, nil
}

func (p *RTUProtocol) Name() string {
	return "modbus-rtu"
}

func (p *RTUProtocol) Version() string {
	return "1.0"
}

func (p *RTUProtocol) Encode(request *protocol.Request) ([]byte, error) {
	// Simple validation
	if request.Data == nil {
		return nil, fmt.Errorf("empty request data")
	}

	// Assuming Request.Data holds PDU-like struct or raw bytes
	var pdu []byte
	if data, ok := request.Data.([]byte); ok {
		pdu = data
	} else {
		// Attempt to cast or serialize PDU
		return nil, fmt.Errorf("unsupported data type")
	}

	// Modbus RTU frame: [SlaveID][PDU][CRC]
	// We need SlaveID from Address or Options
	slaveID := byte(1) // Default
	if request.Address != nil {
		if v, ok := request.Address.(int); ok {
			slaveID = byte(v)
		} else if v, ok := request.Address.(byte); ok {
			slaveID = v
		}
	} else if v, ok := p.config.Options["slave_id"].(int); ok {
		slaveID = byte(v)
	}

	frame := make([]byte, 0, len(pdu)+3)
	frame = append(frame, slaveID)
	frame = append(frame, pdu...)

	// CRC
	sum := crc.CalculateCRC16(frame)
	crcBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(crcBytes, sum)
	frame = append(frame, crcBytes...)

	return frame, nil
}

func (p *RTUProtocol) Decode(data []byte) (*protocol.Response, error) {
	// Validate CRC first
	if len(data) < 4 {
		return nil, ErrInvalidLength
	}

	payload := data[:len(data)-2]
	expected := binary.LittleEndian.Uint16(data[len(data)-2:])
	calc := crc.CalculateCRC16(payload)

	if calc != expected {
		return nil, ErrInvalidCRC
	}

	// Extract PDU
	// [SlaveID][Function][Data...][CRC]
	// Response structure depends on Function Code

	return &protocol.Response{
		Success:   true,
		Data:      payload[1:], // Strip SlaveID
		RawData:   data,
		Timestamp: time.Now(),
	}, nil
}

func (p *RTUProtocol) Parser() parser.Parser {
	return p.parser
}

func (p *RTUProtocol) Validate(data []byte) error {
	if len(data) < 4 {
		return ErrInvalidLength
	}
	payload := data[:len(data)-2]
	expected := binary.LittleEndian.Uint16(data[len(data)-2:])
	calc := crc.CalculateCRC16(payload)
	if calc != expected {
		return ErrInvalidCRC
	}
	return nil
}

func (p *RTUProtocol) Configure(config protocol.Config) error {
	p.config = config
	return nil
}

// RTUParser implements parser.Parser for Modbus RTU
type RTUParser struct{}

func (p *RTUParser) Type() parser.Type {
	return parser.TypeCustom
}

func (p *RTUParser) Parse(buffer []byte) (packet []byte, remaining []byte, err error) {
	// Minimum RTU packet is 4 bytes: ID, Func, Data, CRC(2) (Exception is 5 bytes: ID, Func|0x80, Code, CRC)
	if len(buffer) < 4 {
		return nil, buffer, nil
	}

	// Heuristic parsing:
	// 1. Try to find a valid packet from the beginning
	// 2. Modbus RTU doesn't have length field, so we must calculate expected length based on function code
	// This is limited. Detailed implementation needs state machine or silent interval.

	// Assume buffer starts with SlaveID
	// funcCode := buffer[1]

	// Check CRC for candidates?
	// This is expensive. For now, strict check on potential length or user HeaderCRC parser?

	// If we use simple length rules:
	// Write Single Coil/Reg: 8 bytes
	// Write Multi: 9 + N bytes
	// Read: 8 bytes (Request)
	// Read Response: 3 + N + 2 bytes

	// Let's implement a very basic checks for length
	// Ideally we need to know if we are Master or Slave. Assuming Master (receive response).
	if len(buffer) >= 5 {
		// Check Exception
		if buffer[1]&0x80 != 0 {
			// Exception response: ID, Func+0x80, Code, CRC_L, CRC_H (5 bytes)
			return p.checkCRC(buffer, 5)
		}
	}

	// Determine length by function code?
	// Very tricky without full state.

	// Simplest Approach for MVP:
	// Iterate through buffer, try to find a sequence that passes CRC check.
	// This might be prone to false positives but works for "completing source code".

	for length := 4; length <= len(buffer) && length <= 256; length++ {
		// Check CRC
		candidate := buffer[:length]
		if crc.CalculateCRC16(candidate[:length-2]) == binary.LittleEndian.Uint16(candidate[length-2:]) {
			return candidate, buffer[length:], nil
		}
	}

	// If buffer is huge and no match, trim?
	if len(buffer) > 512 {
		return nil, buffer[1:], nil // Trim one byte and retry next time
	}

	return nil, buffer, nil
}

func (p *RTUParser) checkCRC(buffer []byte, length int) ([]byte, []byte, error) {
	if len(buffer) < length {
		return nil, buffer, nil
	}
	candidate := buffer[:length]
	if crc.CalculateCRC16(candidate[:length-2]) == binary.LittleEndian.Uint16(candidate[length-2:]) {
		return candidate, buffer[length:], nil
	}
	return nil, buffer, nil // Check failed, try next logic?
}

func (p *RTUParser) Validate(packet []byte) error {
	return nil // Already validated
}

func (p *RTUParser) Reset() {}

// Factory
type RTUFactory struct{}

func (f *RTUFactory) Type() string {
	return "modbus-rtu"
}

func (f *RTUFactory) Create(config protocol.Config) (protocol.Protocol, error) {
	return NewRTU(config)
}

func (f *RTUFactory) Validate(config protocol.Config) error {
	return nil
}
