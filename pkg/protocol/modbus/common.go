package modbus

import "errors"

// Function Codes
const (
	FuncReadCoils              = 0x01
	FuncReadDiscreteInputs     = 0x02
	FuncReadHoldingRegisters   = 0x03
	FuncReadInputRegisters     = 0x04
	FuncWriteSingleCoil        = 0x05
	FuncWriteSingleRegister    = 0x06
	FuncWriteMultipleCoils     = 0x0F
	FuncWriteMultipleRegisters = 0x10
)

// Exception Codes
const (
	ExceptionIllegalFunction    = 0x01
	ExceptionIllegalDataAddress = 0x02
	ExceptionIllegalDataValue   = 0x03
	ExceptionSlaveDeviceFailure = 0x04
)

// Error definitions
var (
	ErrInvalidLength = errors.New("invalid packet length")
	ErrInvalidCRC    = errors.New("invalid crc")
	ErrTimeout       = errors.New("timeout")
)

// PDU stands for Protocol Data Unit
type PDU struct {
	FunctionCode byte
	Data         []byte
}
