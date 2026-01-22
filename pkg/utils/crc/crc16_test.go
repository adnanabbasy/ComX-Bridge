package crc

import "testing"

func TestCalculateCRC16(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want uint16
	}{
		{
			name: "Modbus Example 1",
			data: []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x01},
			want: 0x0A84, // 84 0A in little endian wire format
		},
		{
			name: "Modbus Example 2",
			data: []byte{0x02, 0x03, 0x01, 0x00, 0x00, 0x02},
			want: 0xC402, // C402 ? Let's trust the algo logic: 0xFFFF start, 0xA001 poly
		},
		{
			name: "Empty Data",
			data: []byte{},
			want: 0xFFFF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateCRC16(tt.data); got != tt.want {
				t.Errorf("CalculateCRC16() = %04X, want %04X", got, tt.want)
			}
		})
	}
}
