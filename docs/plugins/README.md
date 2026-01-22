# ComX-Bridge Plugin Development Guide

ComX-Bridge allows easy addition of new transports, protocols, and AI modules via its **Plugin Architecture**.

## Table of Contents

1. [Plugin System Overview](#plugin-system-overview)
2. [Plugin Types](#plugin-types)
3. [Quick Start](#quick-start)
4. [Detailed Guide](#detailed-guide)
5. [Deployment](#deployment)
6. [Best Practices](#best-practices)

---

## Plugin System Overview

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Plugin Manager                          │
│  ┌─────────────┐  ┌─────────────┐                           │
│  │   Loader    │  │  Manager    │                           │
│  │ (DynamicLoad)│  │ (Lifecycle) │                           │
│  └──────┬──────┘  └──────┬──────┘                           │
│         └────────────────┼──────────────────────────────────┤
│                          ▼                                   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                  Plugin Interface                    │   │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐         │   │
│  │  │ Transport │ │ Protocol  │ │    AI     │  ...    │   │
│  │  │  Plugin   │ │  Plugin   │ │  Plugin   │         │   │
│  │  └───────────┘ └───────────┘ └───────────┘         │   │
│  └─────────────────────────────────────────────────────┘   │
36: └─────────────────────────────────────────────────────────────┘
```

### Plugin Loading Methods

ComX-Bridge supports three plugin loading methods:

| Method | Description | Use Case |
|------|------|----------|
| **Built-in** | Included at compile time | Core features |
| **Go Plugin** | `.so` dynamic library | High-performance extensions |
| **Script** | Lua/JavaScript | Rapid prototyping |

### Plugin Lifecycle

```
Load → Init → Start → [Running] → Stop → Unload
              ↑                      │
              └──── OnConfigChange ──┘
```

1. **Load**: Load and validate plugin file
2. **Init**: Apply configuration and initialize
3. **Start**: Activate plugin
4. **Running**: Normal operation
5. **Stop**: Stop and clean up resources
6. **Unload**: Remove from memory

---

## Plugin Types

### 1. Transport Plugin

Implement physical/logical communication channels.

```go
type TransportPlugin interface {
    Plugin
    CreateTransport(config transport.Config) (transport.Transport, error)
    SupportedTypes() []string
}
```

**Examples:**
- Serial (RS232/RS485)
- TCP Client/Server
- UDP
- WebSocket
- MQTT
- BLE (Bluetooth Low Energy)

📖 [Transport Plugin Guide](./transport-plugin.md)

### 2. Protocol Plugin

Implement communication protocols.

```go
type ProtocolPlugin interface {
    Plugin
    CreateProtocol(config protocol.Config) (protocol.Protocol, error)
    SupportedTypes() []string
}
```

**Examples:**
- Modbus RTU/TCP
- BACnet
- OPC-UA
- Custom Binary Protocol

📖 [Protocol Plugin Guide](./protocol-plugin.md)

### 3. AI Plugin

Implement AI feature modules.

```go
type AIPlugin interface {
    Plugin
    GetAnalyzer() (ai.ProtocolAnalyzer, error)
    GetDetector() (ai.AnomalyDetector, error)
}
```

**Examples:**
- Packet Pattern Analyzer
- Anomaly Detector
- Protocol Inference Engine

📖 [AI Plugin Guide](./ai-plugin.md)

---

## Quick Start

### Minimal Plugin Structure

```go
package main

import (
    "context"
    "github.com/commatea/ComX-Bridge/pkg/plugin"
)

// MyPlugin implements plugin.Plugin
type MyPlugin struct {
    config plugin.Config
}

// Info returns plugin metadata
func (p *MyPlugin) Info() plugin.Info {
    return plugin.Info{
        Name:        "my-plugin",
        Version:     "1.0.0",
        Type:        plugin.TypeTransport,
        Description: "My custom transport plugin",
        Author:      "Your Name",
    }
}

// Init initializes the plugin
func (p *MyPlugin) Init(ctx context.Context, config plugin.Config) error {
    p.config = config
    return nil
}

// Start starts the plugin
func (p *MyPlugin) Start() error {
    return nil
}

// Stop stops the plugin
func (p *MyPlugin) Stop() error {
    return nil
}

// Health returns the plugin health status
func (p *MyPlugin) Health() plugin.HealthStatus {
    return plugin.HealthHealthy
}

// Export the plugin (for Go plugins)
var Plugin MyPlugin
```

### Directory Structure

```
my-plugin/
├── plugin.go          # Main plugin code
├── transport.go       # Transport implementation (if Transport plugin)
├── protocol.go        # Protocol implementation (if Protocol plugin)
├── config.go          # Config struct
├── plugin.yaml        # Plugin metadata
├── go.mod
├── go.sum
├── README.md
└── examples/
    └── example.go
```

### plugin.yaml

```yaml
name: my-plugin
version: 1.0.0
type: transport
description: My custom transport plugin
author: Your Name
license: Apache-2.0
homepage: https://github.com/yourname/my-plugin

# Supported types
supported_types:
  - my-transport

# Dependencies
requires:
  - ComX-Bridge >= 0.1.0

# Config Schema
config_schema:
  type: object
  properties:
    option1:
      type: string
      description: First option
    option2:
      type: integer
      default: 100
```

---

## Deployment

### Build Go Plugin

```bash
# Build .so file on Linux
go build -buildmode=plugin -o my-plugin.so ./plugin.go

# Copy to plugins directory
cp my-plugin.so ~/.comx/plugins/
```

### Install Plugin

```bash
# Install via CLI (Coming soon)
comx plugin install ./my-plugin.so

# Or add to configuration
# config.yaml
plugins:
  directory: "./plugins"
  configs:
    - name: my-plugin
      enabled: true
      options:
        option1: "value"
```

---

## Best Practices

### 1. Error Handling

```go
// ✅ Good: Detailed error message
if err := conn.Connect(); err != nil {
    return fmt.Errorf("failed to connect to %s: %w", addr, err)
}
```

### 2. Context Usage

```go
// ✅ Good: Handle timeout/cancellation with Context
func (t *Transport) Send(ctx context.Context, data []byte) (int, error) {
    select {
    case <-ctx.Done():
        return 0, ctx.Err()
    default:
        return t.conn.Write(data)
    }
}
```

### 3. Resource Cleanup

```go
// ✅ Good: Clean up all resources in Stop
func (p *MyPlugin) Stop() error {
    // Close channels
    close(p.done)

    // Close connection
    if p.conn != nil {
        p.conn.Close()
    }

    // Wait for goroutines
    p.wg.Wait()

    return nil
}
```

---

## Next Steps

- [Transport Plugin Guide](./transport-plugin.md)
- [Protocol Plugin Guide](./protocol-plugin.md)
- [AI Plugin Guide](./ai-plugin.md)
- [Script Plugin Guide](./script-plugin.md)
- [Example Plugins](../../plugins/examples/)
