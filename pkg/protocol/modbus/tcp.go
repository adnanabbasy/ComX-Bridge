package modbus

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/parser"
	"github.com/commatea/ComX-Bridge/pkg/protocol"
)

// TCPProtocol implements Modbus TCP.
type TCPProtocol struct {
	config protocol.Config
	parser parser.Parser
}

// NewTCP creates a new Modbus TCP protocol instance.
func NewTCP(config protocol.Config) (protocol.Protocol, error) {
	return &TCPProtocol{
		config: config,
		parser: &TCPParser{},
	}, nil
}

func (p *TCPProtocol) Name() string {
	return "modbus-tcp"
}

func (p *TCPProtocol) Version() string {
	return "1.0"
}

func (p *TCPProtocol) Encode(request *protocol.Request) ([]byte, error) {
	if request.Data == nil {
		return nil, fmt.Errorf("empty request data")
	}

	var pdu []byte
	if data, ok := request.Data.([]byte); ok {
		pdu = data
	} else {
		return nil, fmt.Errorf("unsupported data type")
	}

	// Request ID from request.ID? Modbus TCP TransactionID is 2 bytes.
	// We'll use a counter or hash if not provided.
	var transID uint16 = 0
	// TODO: Manage transaction IDs

	// Unit ID
	unitID := byte(1) // Default
	if request.Address != nil {
		if v, ok := request.Address.(int); ok {
			unitID = byte(v)
		} else if v, ok := request.Address.(byte); ok {
			unitID = v
		}
	}

	// MBAP Header:
	// TransID (2)
	// ProtoID (2) = 0
	// Length (2) = UnitID(1) + PDU(N)
	// UnitID (1)

	length := 1 + len(pdu)

	frame := make([]byte, 7+len(pdu))
	binary.BigEndian.PutUint16(frame[0:2], transID)
	binary.BigEndian.PutUint16(frame[2:4], 0) // Protocol ID
	binary.BigEndian.PutUint16(frame[4:6], uint16(length))
	frame[6] = unitID
	copy(frame[7:], pdu)

	return frame, nil
}

func (p *TCPProtocol) Decode(data []byte) (*protocol.Response, error) {
	if len(data) < 7 {
		return nil, ErrInvalidLength
	}

	// TransID := binary.BigEndian.Uint16(data[0:2])
	// ProtoID := binary.BigEndian.Uint16(data[2:4])
	// Length := binary.BigEndian.Uint16(data[4:6])
	// UnitID := data[6]

	// Validate length?

	// Return PDU
	return &protocol.Response{
		Success:   true,
		Data:      data[7:], // Strip MBAP
		RawData:   data,
		Timestamp: time.Now(),
	}, nil
}

func (p *TCPProtocol) Parser() parser.Parser {
	return p.parser
}

func (p *TCPProtocol) Validate(data []byte) error {
	if len(data) < 7 {
		return ErrInvalidLength
	}
	length := binary.BigEndian.Uint16(data[4:6])
	if len(data) != int(6+length) {
		return ErrInvalidLength
	}
	return nil
}

func (p *TCPProtocol) Configure(config protocol.Config) error {
	p.config = config
	return nil
}

// TCPParser implements parser.Parser for Modbus TCP
type TCPParser struct{}

func (p *TCPParser) Type() parser.Type {
	return parser.TypeLength // It is effectively length based, but specific
}

func (p *TCPParser) Parse(buffer []byte) (packet []byte, remaining []byte, err error) {
	// MBAP Header is 7 bytes
	// Length field is at offset 4, size 2.
	// Length value is number of bytes FOLLOWING the length field (UnitID + PDU).
	// Total Packet Size = 6 (TransID+ProtoID+LenField) + LengthValue

	if len(buffer) < 6 {
		return nil, buffer, nil
	}

	lengthVal := binary.BigEndian.Uint16(buffer[4:6])
	totalLen := 6 + int(lengthVal)

	if len(buffer) < totalLen {
		return nil, buffer, nil
	}

	return buffer[:totalLen], buffer[totalLen:], nil
}

func (p *TCPParser) Validate(packet []byte) error {
	return nil
}

func (p *TCPParser) Reset() {}

// Factory
type TCPFactory struct{}

func (f *TCPFactory) Type() string {
	return "modbus-tcp"
}

func (f *TCPFactory) Create(config protocol.Config) (protocol.Protocol, error) {
	return NewTCP(config)
}

func (f *TCPFactory) Validate(config protocol.Config) error {
	return nil
}
