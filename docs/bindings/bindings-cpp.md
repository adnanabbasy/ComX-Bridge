# ComX-Bridge C++ Binding Guide

> [!NOTE]
> **Implementation Status**: The C API (`comx.h`, `comx.dll/libcomx.so`) is implemented. The wrapper classes shown below are **reference implementations** for your projects. Pre-built C++ wrapper libraries are not currently distributed.

This guide details how to develop C++ applications using ComX-Bridge's C API.

## 1. Overview and Requirements

ComX-Bridge provides an API in the form of **C-compatible functions (extern "C")**, rather than C++ classes. This allows common usage across various C++ compilers (MSVC, GCC, Clang) without ABI compatibility issues.

*   **Header File**: `comx.h`
*   **Library**: `comx.dll` (Windows) / `libcomx.so` (Linux)

## 2. Project Setup

### Header File (`comx.h`)

Copy the `comx.h` file to your project's `include` path. (Refer to `c-api.md` for content)

### Compile and Link

**Windows (MSVC)**
```powershell
# Example: with comx.lib
cl /EHsc example.cpp comx.lib
```

**Linux (GCC/G++)**
```bash
g++ -o example example.cpp -L. -lcomx -Wl,-rpath=.
```

## 3. C++ RAII Wrapper Class Implementation

It is recommended to rely on C++'s RAII (Resource Acquisition Is Initialization) pattern to safely manage C API handles rather than managing them directly.

```cpp
#include <stdexcept>
#include <string>
#include <vector>
#include "comx.h"

class ComxEngineWrapper {
private:
    ComxEngine handle;

public:
    ComxEngineWrapper(const std::string& config_json) {
        handle = comx_engine_create_with_config(config_json.c_str());
        if (!handle) throw std::runtime_error("Failed to create engine");
    }

    ~ComxEngineWrapper() {
        if (handle) {
            comx_engine_stop(handle);
            comx_engine_destroy(handle);
        }
    }

    void start() {
        if (comx_engine_start(handle) != COMX_OK) {
            throw std::runtime_error("Failed to start engine");
        }
    }
    
    // Prevent copy (Avoid double free)
    ComxEngineWrapper(const ComxEngineWrapper&) = delete;
    ComxEngineWrapper& operator=(const ComxEngineWrapper&) = delete;

    ComxEngine get() const { return handle; }
};
```

## 4. Data Transmission Example

```cpp
#include <iostream>
#include <thread>
#include <chrono>
#include "comx.h"

int main() {
    // 1. Prepare Config JSON
    const char* config = R"({
        "gateways": [{
            "name": "sim-device",
            "transport": { "type": "serial", "address": "COM1" },
            "protocol": { "type": "modbus" }
        }]
    })";

    try {
        // 2. Create and Start Engine
        ComxEngine engine = comx_engine_create_with_config(config);
        if (!engine) return 1;
        comx_engine_start(engine);

        // 3. Get Gateway Handle
        ComxGateway gw = comx_engine_get_gateway(engine, "sim-device");
        if (!gw) {
            std::cerr << "Gateway not found!" << std::endl;
            return 1;
        }

        // 4. Send Data
        uint8_t payload[] = {0x01, 0x03, 0x00, 0x00, 0x00, 0x01}; // Modbus Read
        comx_gateway_send(gw, payload, sizeof(payload));

        // 5. Receive Data (Polling)
        uint8_t buffer[1024];
        int received = comx_gateway_receive(gw, buffer, sizeof(buffer), 2000); // 2s timeout
        
        if (received > 0) {
            std::cout << "Received " << received << " bytes." << std::endl;
        } else if (received == COMX_ERR_TIMEOUT) {
            std::cout << "Timeout." << std::endl;
        } else {
            std::cout << "Error: " << received << std::endl;
        }
        
        // 6. Shutdown
        comx_engine_stop(engine);
        comx_engine_destroy(engine);

    } catch (const std::exception& e) {
        std::cerr << e.what() << std::endl;
    }

    return 0;
}
```

## 5. Advanced: Using Asynchronous Callbacks

You can register callbacks to process data asynchronously instead of polling.

```cpp
// Callback Function (static or global)
void on_data_received(const uint8_t* data, int len, void* userdata) {
    std::cout << "Async Callback: Received " << len << " bytes" << std::endl;
}

// Registration
// comx_gateway_set_data_callback(gw, on_data_received, nullptr);
```

> **Note**: Callbacks may be invoked from separate threads in the Go runtime, so you must ensure **Thread Safety** using Mutexes, etc., when accessing shared resources within the function.
