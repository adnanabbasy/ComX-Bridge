# ComX-Bridge Architecture Design

> **"One Engine to Connect Any Communication"**
>
> An AI-based open-source communication platform that abstracts RS232/RS485/Socket/TCP/UDP/HTTP/MQTT into a unified communication layer and extends via protocol plugins.

---

## 1. Project Overview

### 1.1 Vision
An **Intelligent Communication Hub** that easily connects everything from industrial communication to IoT via plugins, while AI automatically analyzes and optimizes protocols.

### 1.2 Core Values
- **Unified**: All communication methods through a single API
- **Extensible**: Infinite expansion via plugins
- **Intelligent**: AI-based automatic analysis/optimization
- **Open**: Building an open-source ecosystem

### 1.3 Tech Stack
- **Core Language**: Go 1.24+
- **Plugin System**: HashiCorp go-plugin (gRPC based)
- **Script Engine**: Lua (gopher-lua) / JavaScript (goja)
- **AI Integration**: Native Go + Multi-LLM Providers (OpenAI, Gemini, Claude, Ollama)
- **Configuration**: YAML / TOML
- **API Gateway**: REST + WebSocket + gRPC

---

## 2. Core Architecture

### 2.1 Layer Structure

```
┌─────────────────────────────────────────────────────────────────┐
│                        Applications                              │
│           (CLI / REST API / WebSocket / Dashboard)               │
40: ├─────────────────────────────────────────────────────────────────┤
│                      AI Engine Layer                             │
│    (Protocol Analyzer / Anomaly Detector / Code Generator)       │
43: ├─────────────────────────────────────────────────────────────────┤
│                    Protocol Plugin Layer                         │
│         (Modbus / Custom Serial / RF / IoT Protocols)            │
46: ├─────────────────────────────────────────────────────────────────┤
│                  Unified Communication API                       │
│              (Connect / Send / Receive / Close)                  │
49: ├─────────────────────────────────────────────────────────────────┤
│                   Packet Parser Engine                           │
│        (Delimiter / Length / Header+CRC / Binary/ASCII)          │
52: ├─────────────────────────────────────────────────────────────────┤
│                  Transport Layer Drivers                         │
│    (Serial / TCP / UDP / WebSocket / HTTP / MQTT / BLE)          │
55: └─────────────────────────────────────────────────────────────────┘
```

### 2.2 Core Components

```
                    ┌──────────────────┐
                    │   Config Loader  │
                    └────────┬─────────┘
                             │
┌────────────────────────────┼────────────────────────────┐
│                            ▼                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│  │   Gateway   │◄───│   Engine    │───►│  AI Sidecar │  │
│  │   Manager   │    │    Core     │    │   (Python)  │  │
│  └──────┬──────┘    └──────┬──────┘    └─────────────┘  │
│         │                  │                             │
│         ▼                  ▼                             │
│  ┌─────────────┐    ┌─────────────┐                     │
│  │  Transport  │    │  Protocol   │                     │
│  │   Registry  │    │   Registry  │                     │
│  └──────┬──────┘    └──────┬──────┘                     │
│         │                  │                             │
│         ▼                  ▼                             │
│  ┌─────────────────────────────────────────────────┐    │
│  │              Plugin Manager                      │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐           │    │
│  │  │ Serial  │ │   TCP   │ │ Modbus  │ ...       │    │
│  │  └─────────┘ └─────────┘ └─────────┘           │    │
│  └─────────────────────────────────────────────────┘    │
│                                                          │
│                     ComX-Bridge Core                     │
87: └──────────────────────────────────────────────────────────┘
```

---

## 3. Core Interface Definitions

### 3.1 Transport Interface

```go
// Transport abstracts a physical/logical communication channel
type Transport interface {
    // Lifecycle
    Connect(ctx context.Context) error
    Close() error
    IsConnected() bool

    // Data Transmission
    Send(ctx context.Context, data []byte) (int, error)
    Receive(ctx context.Context) ([]byte, error)

    // Configuration
    Configure(config TransportConfig) error
    GetInfo() TransportInfo

    // Events
    OnConnect(handler func())
    OnDisconnect(handler func(error))
    OnError(handler func(error))
}

type TransportConfig struct {
    Type       string                 `yaml:"type"`       // serial, tcp, udp, mqtt...
    Address    string                 `yaml:"address"`    // /dev/ttyUSB0, 192.168.1.1:502
    Options    map[string]interface{} `yaml:"options"`    // baudrate, timeout...
    BufferSize int                    `yaml:"buffer_size"`
    Timeout    time.Duration          `yaml:"timeout"`
}

type TransportInfo struct {
    ID          string
    Type        string
    State       ConnectionState
    Statistics  TransportStats
}
```

### 3.2 Protocol Interface

```go
// Protocol abstracts a communication protocol
type Protocol interface {
    // Packet Processing
    Encode(request Request) ([]byte, error)
    Decode(data []byte) (Response, error)

    // Packet Parsing
    GetParser() PacketParser

    // Metadata
    Name() string
    Version() string

    // Command Execution
    Execute(ctx context.Context, cmd Command) (Result, error)
}

// PacketParser extracts complete packets from a byte stream
type PacketParser interface {
    // Packet Boundary Detection
    Parse(buffer []byte) (packet []byte, remaining []byte, complete bool)

    // Parser Type
    Type() ParserType  // Delimiter, Length, HeaderCRC, Custom

    // Validation
    Validate(packet []byte) error
}

type ParserType int

const (
    ParserDelimiter ParserType = iota  // STX...ETX
    ParserLength                        // [LEN][DATA]
    ParserHeaderCRC                     // [HEADER][DATA][CRC]
    ParserFixed                         // Fixed Length
    ParserCustom                        // Custom
)
```

### 3.3 Plugin Interface

```go
// Plugin is a dynamically loaded extension module
type Plugin interface {
    // Metadata
    Info() PluginInfo

    // Lifecycle
    Init(ctx context.Context, config PluginConfig) error
    Start() error
    Stop() error

    // Health Check
    Health() HealthStatus
}

type PluginInfo struct {
    Name        string   `json:"name"`
    Version     string   `json:"version"`
    Type        string   `json:"type"`        // transport, protocol, ai
    Author      string   `json:"author"`
    Description string   `json:"description"`
    Requires    []string `json:"requires"`    // Dependencies
}

// TransportPlugin adds a new transport method
type TransportPlugin interface {
    Plugin
    CreateTransport(config TransportConfig) (Transport, error)
}

// ProtocolPlugin adds a new protocol
type ProtocolPlugin interface {
    Plugin
    CreateProtocol(config ProtocolConfig) (Protocol, error)
}
```

### 3.4 Gateway Interface

```go
// Gateway configures a communication channel by combining Transport + Protocol
type Gateway interface {
    // Configuration
    UseTransport(name string, config TransportConfig) error
    UseProtocol(name string, config ProtocolConfig) error

    // Execution
    Start(ctx context.Context) error
    Stop() error

    // Command Execution
    Execute(ctx context.Context, cmd Command) (Result, error)

    // Data Stream
    Subscribe(topic string) (<-chan Message, error)
    Publish(topic string, msg Message) error

    // Status
    Status() GatewayStatus
}
```

---

## 4. AI Engine Design

### 4.1 AI Feature Modules

```
┌─────────────────────────────────────────────────────────┐
│                     AI Engine                            │
├─────────────────────────────────────────────────────────┤
│  ┌──────────────────┐  ┌──────────────────┐            │
│  │ Protocol Analyzer│  │ Anomaly Detector │            │
│  │  - Hybrid(Rule+AI) │  │  - Anomaly Detect  │            │
│  │  - Struct Infer  │  │  - Quality Monitor │            │
│  │  - Modbus/JSON   │  │  - Predict Alert   │            │
│  │  ✅ Implemented  │  │  ✅ Implemented    │            │
│  └──────────────────┘  └──────────────────┘            │
│                                                          │
│  ┌──────────────────┐  ┌──────────────────┐            │
│  │  Code Generator  │  │  NL Commander    │            │
│  │  - Parser Gen    │  │  - NLP -> Cmd      │            │
│  │  - Plugin Gen    │  │  - Device Control  │            │
│  │  - Config Gen    │  │  - Query Process   │            │
│  │  ✅ Implemented  │  │  ✅ Implemented    │            │
│  └──────────────────┘  └──────────────────┘            │
│                                                          │
│  ┌──────────────────┐  ┌──────────────────┐            │
│  │ Auto Optimizer   │  │ Digital Twin     │            │
│  │  - Timeout Tune  │  │  - Device Sim      │            │
│  │  - Retry Opt.    │  │  - Virtual Test    │            │
│  │  - Conn. Tuning  │  │  - Predict Model   │            │
│  │  ✅ Implemented  │  │  ✅ Implemented    │            │
│  └──────────────────┘  └──────────────────┘            │
│                                                          │
│  ┌─────────────────────────────────────────────────┐   │
│  │              LLM Provider Layer                  │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌───────┐ │   │
│  │  │ OpenAI  │ │ Gemini  │ │ Claude  │ │Ollama │ │   │
│  │  │ GPT-4   │ │ 1.5 Pro │ │ Sonnet  │ │ Local │ │   │
│  │  └─────────┘ └─────────┘ └─────────┘ └───────┘ │   │
│  │              ✅ Implemented                      │   │
│  └─────────────────────────────────────────────────┘   │
│                                                          │
│  ┌─────────────────────────────────────────────────┐   │
│  │              Edge Rule Engine                    │   │
│  │  ┌─────────────────────────────────────────┐   │   │
│  │  │  Lua Scripting (gopher-lua)             │   │   │
│  │  │  - on_message() hook                     │   │   │
│  │  │  - Data Filtering/Transformation         │   │   │
│  │  │  - Conditional Routing                   │   │   │
│  │  └─────────────────────────────────────────┘   │   │
│  │              ✅ Implemented                      │   │
│  └─────────────────────────────────────────────────┘   │
294: └─────────────────────────────────────────────────────────┘
```

### 4.2 AI Interface

```go
// AIEngine is the integrated AI feature interface
type AIEngine interface {
    // Protocol Analysis
    AnalyzePackets(samples [][]byte) (*ProtocolAnalysis, error)
    InferStructure(data []byte) (*PacketStructure, error)

    // Anomaly Detection
    DetectAnomaly(ctx context.Context, stream <-chan []byte) (<-chan Anomaly, error)
    LearnNormalPattern(samples [][]byte) error

    // Code Generation
    GenerateParser(structure *PacketStructure) (string, error)
    GeneratePlugin(spec *PluginSpec) (string, error)

    // Natural Language Processing
    ParseCommand(natural string) (*Command, error)
    ExplainPacket(data []byte) (string, error)
}

type ProtocolAnalysis struct {
    PacketType     string          // binary, ascii, mixed
    HasDelimiter   bool
    Delimiter      []byte
    HasLengthField bool
    LengthOffset   int
    HasCRC         bool
    CRCType        string          // crc16, crc32, checksum
    Fields         []FieldAnalysis
    Confidence     float64
}

type Anomaly struct {
    Type        AnomalyType
    Severity    Severity
    Description string
    Timestamp   time.Time
    Data        []byte
    Suggestion  string
}
```

---

## 5. Directory Structure

```
ComX-Bridge/
├── cmd/
│   ├── comx/              # CLI Main Entrypoint
│   │   └── main.go
│   ├── server/            # API Server (Internal)
│   │   └── main.go
│   └── ai-test/           # AI Test Tool
│       └── main.go
│
├── pkg/
│   ├── core/              # Core Engine
│   │   ├── engine.go
│   │   ├── gateway.go
│   │   └── registry.go
│   │
│   ├── transport/         # Transport Layer
│   │   ├── transport.go   # Interface
│   │   ├── serial/        # RS232/485
│   │   ├── tcp/           # TCP Client/Server
│   │   ├── udp/           # UDP
│   │   ├── websocket/     # WebSocket
│   │   ├── mqtt/          # MQTT
│   │   └── http/          # HTTP Client
│   │
│   ├── protocol/          # Protocol Layer
│   │   ├── protocol.go    # Interface
│   │   ├── modbus/        # Modbus RTU/TCP
│   │   └── raw/           # Raw Binary
│   │
│   ├── parser/            # Packet Parser Engine
│   │   ├── parser.go      # Interface
│   │   ├── delimiter.go   # Delimiter based
│   │   ├── length.go      # Length based
│   │   └── headercrc.go   # Header+CRC based
│   │
│   ├── plugin/            # Plugin System
│   │   ├── manager.go     # Plugin Manager
│   │   ├── loader.go      # Dynamic Loader
│   │   └── plugin.go      # Plugin Interface
│   │
│   ├── ai/                # AI Engine
│   │   ├── engine.go      # Integrated AI Engine
│   │   ├── analyzer.go    # Protocol Analyzer
│   │   ├── detector.go    # Anomaly Detector
│   │   ├── generator.go   # Text-to-Config/Code
│   │   ├── llm/           # LLM Providers (OpenAI, Gemini, etc.)
│   │   └── config/        # AI Config
│   │
│   ├── api/               # API Layer
│   │   ├── middleware/    # Auth & Logging Middleware
│   │   │   ├── auth.go
│   │   │   └── grpc.go
│   │   ├── rest/          # REST API
│   │   │   ├── server.go
│   │   │   ├── handlers.go
│   │   │   └── auth_handler.go
│   │   ├── ws/            # WebSocket API
│   │   │   └── server.go
│   │   └── grpc/          # gRPC API
│   │       ├── server.go
│   │       └── types.go
│   │
│   ├── capi/              # C API Bindings
│   │   └── capi.go
│   │
│   ├── config/            # Configuration Management
│   │   ├── config.go
│   │   └── loader.go
│   │
│   └── rules/             # Edge Rule Engine
│       ├── rules.go       # Rule Interface
│       └── js_engine.go   # JavaScript/Lua Engine
│
├── plugins/               # External Plugins
│   └── examples/          # Example Plugins
│
├── scripts/               # Script Plugins
│   ├── lua/               # Lua Scripts
│   └── js/                # JavaScript Scripts
│
├── examples/              # Usage Examples
│   ├── basic/
│   └── modbus/
│
├── configs/               # Config Examples
│   ├── config.yaml
│   └── plugins.yaml
│
├── api/                   # API Specs
│   ├── openapi.yaml       # OpenAPI Spec
│   └── proto/             # gRPC Proto
│       └── comx.proto
│
├── docs/                  # Documentation
│   ├── architecture/
│   ├── bindings/
│   ├── features/
│   └── plugins/
│
├── web/                   # Web Dashboard
│   └── admin/             # React Admin Sources
│
├── tests/                 # Tests
│   ├── integration/
│   └── e2e/
│
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── README.md
```

---

## 6. Data Flow

### 6.1 Receive Data Flow

```
[Physical Device]
       │
       ▼
┌──────────────┐
│  Transport   │  ← Receive Byte Stream
│   Driver     │
└──────┬───────┘
       │ []byte
       ▼
┌──────────────┐
│   Packet     │  ← Assemble Complete Packet
│   Parser     │
└──────┬───────┘
       │ Packet
       ▼
┌──────────────┐
│   Protocol   │  ← Decode Protocol
│   Decoder    │
└──────┬───────┘
       │ Message
       ▼
┌──────────────┐
│  AI Engine   │  ← Anomaly Detect / Analysis (Optional)
│  (Optional)  │
└──────┬───────┘
       │ Message
       ▼
┌──────────────┐
│  Message     │  ← Distribute to Subscribers
│   Router     │
└──────┬───────┘
       │
       ▼
[Application / API / Dashboard]
```

### 6.2 Send Data Flow

```
[Application Command]
       │
       ▼
┌──────────────┐
│  Command     │  ← Validate Command
│  Validator   │
└──────┬───────┘
       │ Command
       ▼
┌──────────────┐
│   Protocol   │  ← Encode Protocol
│   Encoder    │
└──────┬───────┘
       │ []byte
       ▼
┌──────────────┐
│  Transport   │  ← Transmit Bytes
│   Driver     │
└──────┬───────┘
       │
       ▼
[Physical Device]
```

---

## 7. Configuration Structure

### 7.1 Main Configuration (config.yaml)

```yaml
# ComX-Bridge Configuration
version: "1.0"

server:
  host: "0.0.0.0"
  port: 8080
  grpc_port: 9090

logging:
  level: "info"          # debug, info, warn, error
  format: "json"         # json, text
  output: "stdout"       # stdout, file
  file: "/var/log/comx/comx.log"

gateways:
  - name: "modbus-gateway"
    transport:
      type: "serial"
      address: "/dev/ttyUSB0"
      options:
        baudrate: 9600
        databits: 8
        parity: "none"
        stopbits: 1
    protocol:
      type: "modbus-rtu"
      options:
        slave_id: 1
        timeout: 1000ms

  - name: "tcp-gateway"
    transport:
      type: "tcp"
      address: "192.168.1.100:502"
      options:
        timeout: 5s
        keepalive: true
    protocol:
      type: "modbus-tcp"

ai:
  enabled: true
  llm:
    provider: "openai"       # openai, gemini, claude, ollama
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4"
  sidecar:
    address: "localhost:50051"
  features:
    anomaly_detection: true
    protocol_analysis: true
    auto_optimize: false

plugins:
  directory: "./plugins"
  autoload: true

metrics:
  enabled: true
  endpoint: "/metrics"
```

---

## 8. MVP Roadmap

### Phase 1: Foundation (Completed)
- [x] Project Structure Creation
- [x] Core Interface Definition
- [x] Serial Transport Implementation
- [x] TCP Transport Implementation
- [x] Basic Packet Parser (Delimiter, Length)
- [x] Config Loader
- [x] CLI Basic Features
- [x] UDP Transport Implementation

### Phase 2: Protocol & Parser (Completed)
- [x] Modbus RTU/TCP Protocol
- [x] Header+CRC Parser
- [x] Raw Binary Protocol
- [x] Plugin System Basic Structure
- [x] Logging & Metrics

### Phase 3: API & Integration (Completed)
- [x] REST API Server
- [x] WebSocket Real-time Stream
- [x] Basic Dashboard (Docs/Examples)
- [x] Docker Support (Dockerfile)
- [x] Documentation

### Phase 4: AI Integration (Completed)
- [x] AI Engine Core (Go Native)
- [x] Packet Pattern Analyzer
- [x] Anomaly Detector (Statistical)
- [x] Automatic Protocol Inference
- [x] Code Auto-Generation (Template)
- [x] Natural Language Processing (Keyword)
- [x] Multi-LLM Provider Support (OpenAI, Gemini, Claude, Ollama)

### Phase 5: Ecosystem (Completed)
- [x] Additional Transports (MQTT, WebSocket, HTTP, BLE)
- [x] Additional Protocols (BACnet, OPC-UA)
- [x] C API / gRPC Bindings
- [x] Cloud Integration (MQTT/HTTP)
- [ ] Plugin Marketplace
- [ ] Node-RED Integration

- [ ] Community Building

---

## 9. Technical Considerations

### 9.1 Performance
- Goroutine Pool for concurrent connection handling
- Lock-free Queue for message processing
- Buffer Pooling to minimize GC

### 9.2 Reliability
- Graceful Shutdown
- Auto-Reconnection Mechanism
- Circuit Breaker Pattern

### 9.3 Security
- TLS/mTLS Support (Transport & API)
- JWT & API Key Authentication
- Role-Based Access Control (RBAC)
- Audit Logging

### 9.4 Testing
- Unit Tests > 80% Coverage
- Integration Tests
- Virtual Device Simulator

---

## 10. License

Apache 2.0 or MIT (Commercial Friendly)
