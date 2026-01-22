# ComX-Bridge C API (FFI)

The C API allows access to ComX-Bridge's core features from C/C++ and any language supporting FFI (Python, C#, Rust, etc.).

## 1. Header File (comx.h)

The header file below is used with `comx.dll` (Windows) or `libcomx.so` (Linux).

```c
// comx.h - ComX-Bridge C API
#ifndef COMX_H
#define COMX_H

#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

// Handle Types
typedef void* ComxEngine;
typedef void* ComxGateway;
typedef void* ComxTransport;

// Error Codes
typedef enum {
    COMX_OK = 0,
    COMX_ERR_INVALID_PARAM = -1,
    COMX_ERR_NOT_CONNECTED = -2,
    COMX_ERR_TIMEOUT = -3,
    COMX_ERR_SEND_FAILED = -4,
    COMX_ERR_RECEIVE_FAILED = -5,
    COMX_ERR_CONFIG_INVALID = -6,
    COMX_ERR_GATEWAY_NOT_FOUND = -7,
    COMX_ERR_MEMORY = -8,
    COMX_ERR_UNKNOWN = -99
} ComxError;

// Connection State
typedef enum {
    COMX_STATE_DISCONNECTED = 0,
    COMX_STATE_CONNECTING = 1,
    COMX_STATE_CONNECTED = 2,
    COMX_STATE_RECONNECTING = 3,
    COMX_STATE_ERROR = 4
} ComxState;

// ============== Engine API ==============

// Create Engine
// config_json: Configuration string in JSON format
ComxEngine comx_engine_create_with_config(const char* config_json);

// Destroy Engine
void comx_engine_destroy(ComxEngine engine);

// Start Engine
ComxError comx_engine_start(ComxEngine engine);

// Stop Engine
ComxError comx_engine_stop(ComxEngine engine);

// Get Gateway
ComxGateway comx_engine_get_gateway(ComxEngine engine, const char* name);

// ============== Gateway API ==============

// 3. Check Gateway State
ComxState comx_gateway_state(ComxGateway gateway);

// 4. Send Data
// data: Byte array to send
// len: Data length
ComxError comx_gateway_send(ComxGateway gateway, const uint8_t* data, int len);

// 5. Receive Data (Blocking/Timeout)
// buffer: Receive buffer
// max_len: Buffer size
// timeout_ms: Timeout (milliseconds)
// Returns: Number of bytes received (0 if timeout or no data, negative for error)
int comx_gateway_receive(ComxGateway gateway, uint8_t* buffer, int max_len, int timeout_ms);

// ============== Utilities ==============

const char* comx_version(void);
void comx_free(void* ptr);

#ifdef __cplusplus
}
#endif

#endif // COMX_H
```

## 2. Build Instructions

Build Go source code into a C shared library.

### Windows (DLL)
```powershell
go build -buildmode=c-shared -o comx.dll ./pkg/capi/
```

### Linux (.so)
```bash
go build -buildmode=c-shared -o libcomx.so ./pkg/capi/
```

## 3. Usage Examples (C++ & C#)

### C++ Example

```cpp
#include <iostream>
#include "comx.h"

int main() {
    const char* config = R"({"gateways": [{"name": "test-gw", "transport": {"type": "tcp", "address": "127.0.0.1:8080"}}]})";
    
    // Initialize Engine
    ComxEngine engine = comx_engine_create_with_config(config);
    comx_engine_start(engine);
    
    // Get Gateway
    ComxGateway gw = comx_engine_get_gateway(engine, "test-gw");
    
    // Send Data
    uint8_t data[] = "Hello Bridge";
    comx_gateway_send(gw, data, sizeof(data));
    
    // Cleanup
    comx_engine_stop(engine);
    comx_engine_destroy(engine);
    return 0;
}
```

### C# Example (Windows P/Invoke)

```csharp
using System.Runtime.InteropServices;

class ComxBridge {
    [DllImport("comx.dll")]
    public static extern IntPtr comx_engine_create_with_config(string json);
    
    [DllImport("comx.dll")]
    public static extern int comx_engine_start(IntPtr engine);
    
    // ... Other P/Invoke declarations ...
}

class Program {
    static void Main() {
        var engine = ComxBridge.comx_engine_create_with_config("{}");
        ComxBridge.comx_engine_start(engine);
    }
}
```
