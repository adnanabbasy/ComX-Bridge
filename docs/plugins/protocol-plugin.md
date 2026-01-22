# ComX-Bridge Protocol Plugin Development Guide

This document describes how to develop a Protocol plugin for ComX-Bridge.
Through plugins, you can easily integrate not only standard protocols like Modbus and BACnet but also proprietary (Custom) protocols.

## 📁 Example Code

The **complete source code** explained in this document is available at:

`plugins/examples/modbus-custom/main.go`

---

## 1. Interface Definition

A Protocol plugin must implement the interface in the `pkg/protocol` package.

### Core Interface

```go
type Protocol interface {
    // 1. Metadata
    Name() string
    Version() string

    // 2. Encoding (Request -> Bytes)
    Encode(request *Request) ([]byte, error)

    // 3. Decoding (Bytes -> Response)
    Decode(data []byte) (*Response, error)

    // 4. Parser (Stream -> Packet Separation)
    Parser() Parser

    // 5. Configuration
    Configure(config Config) error
}
```

---

## 2. Step-by-Step Development Guide

### Step 1: Project Setup

Create a new plugin directory and initialize `go.mod`.

```bash
mkdir my-protocol-plugin
cd my-protocol-plugin
go mod init my-protocol-plugin
```

### Step 2: Define Struct and Export Plugin

You must export the entry point variable (`Plugin`) of the plugin.

```go
package main

import "github.com/commatea/ComX-Bridge/pkg/protocol"

type MyProtocol struct {}

// Required: ComX Engine looks for this variable to load.
var Plugin MyProtocol
```

### Step 3: Implement Encoding

Convert user requests into byte arrays that the device can understand.

```go
func (p *MyProtocol) Encode(req *protocol.Request) ([]byte, error) {
    // Example: [STX][COMMAND][DATA][ETX]
    buf := new(bytes.Buffer)
    buf.WriteByte(0x02) // STX
    buf.WriteString(req.Command)
    buf.Write(req.Data.([]byte))
    buf.WriteByte(0x03) // ETX
    
    return buf.Bytes(), nil
}
```

### Step 4: Implement Parser

Logic to slice a complete packet from a continuous data stream like TCP/Serial.

```go
type MyParser struct {}

func (p *MyParser) Parse(buffer []byte) ([]byte, []byte, error) {
    // Find STX(0x02) ~ ETX(0x03)
    start := bytes.IndexByte(buffer, 0x02)
    if start == -1 {
        return nil, buffer, parser.ErrIncompletePacket
    }
    
    end := bytes.IndexByte(buffer[start:], 0x03)
    if end == -1 {
        return nil, buffer[start:], parser.ErrIncompletePacket
    }
    
    // Extract complete packet
    packetLen := start + end + 1
    packet := buffer[start:packetLen]
    remaining := buffer[packetLen:]
    
    return packet, remaining, nil
}
```

### Step 5: Implement Decoding

Interpret the sliced packet and convert it into a `Response` object.

```go
func (p *MyProtocol) Decode(data []byte) (*protocol.Response, error) {
    // Validate CRC, etc.
    if !CheckCRC(data) {
        return nil, errors.New("crc error")
    }

    return &protocol.Response{
        Success: true,
        Data:    string(data[1 : len(data)-1]), // Remove STX, ETX
    }, nil
}
```

---

## 3. Build and Install

Compile with Go plugin mode (`-buildmode=plugin`) to generate a `.so` file.

**Linux/Mac:**
```bash
go build -buildmode=plugin -o my-protocol.so main.go
cp my-protocol.so ~/ComX-Bridge/plugins/
```

**Windows:**
Windows does not support Go Plugin mode, so you must include the source code and rebuild the entire project, or use the interface method (gRPC).
(The current example is based on the source inclusion method.)

---

## 4. Troubleshooting

### Q. I get "plugin: not implemented on windows" error.
A. Go's default plugin system supports only Linux/Mac. On Windows, put the code directly into the `plugins/` folder of the ComX-Bridge source tree and enable it in `config.yaml`.

### Q. The parser keeps waiting for data.
A. If the `Parse()` function returns `nil, buffer, ErrIncompletePacket`, the engine waits until more data is received. Check your packet length condition or delimiter logic.

---

## Next Steps

*   [Transport Plugin Development Guide](../transport-plugin.md)
*   [AI Plugin Development Guide](../ai-plugin.md)
*   [View Example Code](../../plugins/examples/)
