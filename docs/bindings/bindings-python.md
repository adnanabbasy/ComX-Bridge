# ComX-Bridge Python Binding Guide

> [!NOTE]
> **Implementation Status**: The C API is implemented. The Python wrapper module (`comx.py`) shown below is a **reference implementation**. Official PyPI package is planned but not yet released.

This guide explains how to integrate the ComX-Bridge native library in Python using the `ctypes` module.

Python is powerful for data analysis and AI prototyping, making it easy to build a **Hardware Control + AI Analysis** pipeline by connecting with ComX-Bridge.

## 1. Prerequisites

*   **Python Version**: 3.7+ recommended
*   **Library**: `comx.dll` (Windows) or `libcomx.so` (Linux)

## 2. Python Wrapper Module (`comx.py`)

Write a module that loads C functions using `ctypes` and wraps them in Python-friendly classes.

```python
import ctypes
import json
import os
import sys
from ctypes import c_void_p, c_char_p, c_int, c_ubyte, POINTER

# Load Library
def load_library():
    lib_name = "comx.dll" if sys.platform == "win32" else "libcomx.so"
    lib_path = os.path.join(os.getcwd(), lib_name)
    return ctypes.CDLL(lib_path)

_lib = load_library()

# Function Signature Definition
_lib.comx_engine_create_with_config.argtypes = [c_char_p]
_lib.comx_engine_create_with_config.restype = c_void_p

_lib.comx_engine_destroy.argtypes = [c_void_p]
_lib.comx_engine_start.argtypes = [c_void_p]
_lib.comx_engine_start.restype = c_int

_lib.comx_engine_get_gateway.argtypes = [c_void_p, c_char_p]
_lib.comx_engine_get_gateway.restype = c_void_p

_lib.comx_gateway_send.argtypes = [c_void_p, POINTER(c_ubyte), c_int]
_lib.comx_gateway_send.restype = c_int

_lib.comx_gateway_receive.argtypes = [c_void_p, POINTER(c_ubyte), c_int, c_int]
_lib.comx_gateway_receive.restype = c_int

class ComxError(Exception):
    pass

class Gateway:
    def __init__(self, handle):
        self._handle = handle

    def send(self, data: bytes):
        buf = (c_ubyte * len(data)).from_buffer_copy(data)
        ret = _lib.comx_gateway_send(self._handle, buf, len(data))
        if ret != 0:
            raise ComxError(f"Send failed with code {ret}")

    def receive(self, timeout_ms=1000, max_len=4096) -> bytes:
        buf = (c_ubyte * max_len)()
        ret = _lib.comx_gateway_receive(self._handle, buf, max_len, timeout_ms)
        
        if ret < 0:
            raise ComxError(f"Receive error code {ret}")
        if ret == 0:
            return None # Timeout
        
        return bytes(buf[:ret])

class Engine:
    def __init__(self, config: dict):
        json_str = json.dumps(config).encode('utf-8')
        self._handle = _lib.comx_engine_create_with_config(json_str)
        if not self._handle:
            raise ComxError("Failed to create engine")

    def __enter__(self):
        self.start()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.stop()
        self.destroy()

    def start(self):
        if _lib.comx_engine_start(self._handle) != 0:
            raise ComxError("Failed to start engine")

    def stop(self):
        _lib.comx_engine_stop(self._handle)

    def destroy(self):
        _lib.comx_engine_destroy(self._handle)

    def get_gateway(self, name: str) -> Gateway:
        gw_handle = _lib.comx_engine_get_gateway(self._handle, name.encode('utf-8'))
        if not gw_handle:
            raise KeyError(f"Gateway '{name}' not found")
        return Gateway(gw_handle)
```

## 3. Usage Example (`main.py`)

Import `comx.py` created above.

```python
from comx import Engine
import time

def main():
    config = {
        "gateways": [{
            "name": "loopback",
            "transport": {"type": "udp", "address": "127.0.0.1:9000"},
            "protocol": {"type": "raw"}
        }]
    }

    try:
        # Auto resource cleanup using Context Manager (with)
        with Engine(config) as engine:
            print("Engine started.")
            gw = engine.get_gateway("loopback")

            # Send
            msg = b"Hello Python Bridge!"
            gw.send(msg)
            print(f"Sent: {msg}")

            # Receive
            resp = gw.receive(timeout_ms=2000)
            if resp:
                print(f"Received: {resp}")
            else:
                print("Receive timeout.")

    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    main()
```

## 4. Tip: Using Jupyter Notebook

This method works directly in Jupyter Notebook, making it very useful for fetching hardware data in real-time and visualizing it with `pandas` or `matplotlib`.
