/**
 * ComX-Bridge C API
 *
 * A unified communication platform for industrial and IoT protocols.
 * This header provides C-compatible interface for using ComX-Bridge
 * from C, C++, C#, Python, and other languages via FFI.
 *
 * Build the shared library:
 *   go build -buildmode=c-shared -o libcomx.so ./pkg/capi/
 *
 * Usage (C++):
 *   #include "comx.h"
 *   ComxEngine engine = comx_engine_create_with_config(json);
 *   comx_engine_start(engine);
 *   ...
 *   comx_engine_destroy(engine);
 */

#ifndef COMX_H
#define COMX_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/* ============== Type Definitions ============== */

/**
 * Handle types - opaque pointers to internal structures
 */
typedef uintptr_t ComxEngine;
typedef uintptr_t ComxGateway;
typedef uintptr_t ComxTransport;

/**
 * Error codes returned by API functions
 */
typedef enum ComxError {
    COMX_OK = 0,                    /**< Success */
    COMX_ERR_INVALID_PARAM = -1,    /**< Invalid parameter */
    COMX_ERR_NOT_CONNECTED = -2,    /**< Not connected */
    COMX_ERR_TIMEOUT = -3,          /**< Operation timed out */
    COMX_ERR_SEND_FAILED = -4,      /**< Failed to send data */
    COMX_ERR_RECEIVE_FAILED = -5,   /**< Failed to receive data */
    COMX_ERR_CONFIG_INVALID = -6,   /**< Invalid configuration */
    COMX_ERR_GATEWAY_NOT_FOUND = -7,/**< Gateway not found */
    COMX_ERR_MEMORY = -8,           /**< Memory allocation failed */
    COMX_ERR_ENGINE_NOT_STARTED = -9, /**< Engine not started */
    COMX_ERR_UNKNOWN = -99          /**< Unknown error */
} ComxError;

/**
 * Connection state
 */
typedef enum ComxState {
    COMX_STATE_DISCONNECTED = 0,    /**< Not connected */
    COMX_STATE_CONNECTING = 1,      /**< Connection in progress */
    COMX_STATE_CONNECTED = 2,       /**< Connected */
    COMX_STATE_RECONNECTING = 3,    /**< Reconnecting */
    COMX_STATE_ERROR = 4            /**< Error state */
} ComxState;

/**
 * Event types for callbacks
 */
typedef enum ComxEventType {
    COMX_EVENT_CONNECTED = 0,       /**< Connection established */
    COMX_EVENT_DISCONNECTED = 1,    /**< Connection lost */
    COMX_EVENT_ERROR = 2,           /**< Error occurred */
    COMX_EVENT_DATA = 3,            /**< Data received */
    COMX_EVENT_STATE_CHANGED = 4    /**< State changed */
} ComxEventType;

/**
 * Callback for receiving data asynchronously
 *
 * @param data     Pointer to received data
 * @param len      Length of data in bytes
 * @param userdata User-provided context pointer
 */
typedef void (*ComxDataCallback)(const uint8_t* data, int len, void* userdata);

/**
 * Callback for receiving events
 *
 * @param event_type  Type of event (see ComxEventType)
 * @param message     Event message (may be NULL)
 * @param userdata    User-provided context pointer
 */
typedef void (*ComxEventCallback)(int event_type, const char* message, void* userdata);


/* ============== Engine API ============== */

/**
 * Create a new engine from configuration file
 *
 * @param config_path  Path to YAML configuration file
 * @return Engine handle, or 0 on failure
 */
ComxEngine comx_engine_create(const char* config_path);

/**
 * Create a new engine from JSON configuration
 *
 * @param config_json  JSON string containing configuration
 * @return Engine handle, or 0 on failure
 *
 * Example:
 * @code
 * const char* config = "{"
 *     "\"gateways\": [{"
 *         "\"name\": \"serial-gw\","
 *         "\"transport\": {\"type\": \"serial\", \"address\": \"/dev/ttyUSB0\"}"
 *     "}]"
 * "}";
 * ComxEngine engine = comx_engine_create_with_config(config);
 * @endcode
 */
ComxEngine comx_engine_create_with_config(const char* config_json);

/**
 * Destroy an engine and release all resources
 *
 * @param engine  Engine handle
 */
void comx_engine_destroy(ComxEngine engine);

/**
 * Start the engine and all configured gateways
 *
 * @param engine  Engine handle
 * @return COMX_OK on success, error code on failure
 */
ComxError comx_engine_start(ComxEngine engine);

/**
 * Stop the engine and all gateways
 *
 * @param engine  Engine handle
 * @return COMX_OK on success, error code on failure
 */
ComxError comx_engine_stop(ComxEngine engine);

/**
 * Check if engine is running
 *
 * @param engine  Engine handle
 * @return true if running, false otherwise
 */
bool comx_engine_is_running(ComxEngine engine);

/**
 * Get a gateway by name
 *
 * @param engine  Engine handle
 * @param name    Gateway name
 * @return Gateway handle, or 0 if not found
 */
ComxGateway comx_engine_get_gateway(ComxEngine engine, const char* name);

/**
 * List all gateways
 *
 * @param engine  Engine handle
 * @return JSON array of gateway names (caller must free with comx_free)
 */
const char* comx_engine_list_gateways(ComxEngine engine);

/**
 * Add a gateway at runtime
 *
 * @param engine       Engine handle
 * @param config_json  Gateway configuration as JSON
 * @return COMX_OK on success, error code on failure
 */
ComxError comx_engine_add_gateway(ComxEngine engine, const char* config_json);

/**
 * Remove a gateway
 *
 * @param engine  Engine handle
 * @param name    Gateway name to remove
 * @return COMX_OK on success, error code on failure
 */
ComxError comx_engine_remove_gateway(ComxEngine engine, const char* name);


/* ============== Gateway API ============== */

/**
 * Get gateway connection state
 *
 * @param gateway  Gateway handle
 * @return Connection state (see ComxState)
 */
ComxState comx_gateway_state(ComxGateway gateway);

/**
 * Get gateway information as JSON
 *
 * @param gateway  Gateway handle
 * @return JSON string with gateway info (caller must free with comx_free)
 */
const char* comx_gateway_info(ComxGateway gateway);

/**
 * Send raw data through gateway
 *
 * @param gateway  Gateway handle
 * @param data     Data to send
 * @param len      Length of data
 * @return COMX_OK on success, error code on failure
 */
ComxError comx_gateway_send(ComxGateway gateway, const uint8_t* data, int len);

/**
 * Receive data from gateway (blocking with timeout)
 *
 * @param gateway     Gateway handle
 * @param buffer      Buffer to store received data
 * @param max_len     Maximum bytes to receive
 * @param timeout_ms  Timeout in milliseconds
 * @return Number of bytes received, or negative error code
 */
int comx_gateway_receive(ComxGateway gateway, uint8_t* buffer, int max_len, int timeout_ms);

/**
 * Execute a protocol command
 *
 * @param gateway        Gateway handle
 * @param command_json   Command as JSON
 * @param result_buffer  Buffer for result JSON
 * @param buffer_size    Size of result buffer
 * @return COMX_OK on success, error code on failure
 */
ComxError comx_gateway_execute(ComxGateway gateway, const char* command_json,
                                char* result_buffer, int buffer_size);

/**
 * Set callback for asynchronous data reception
 *
 * @param gateway   Gateway handle
 * @param cb        Callback function
 * @param userdata  User context passed to callback
 * @return COMX_OK on success, error code on failure
 */
ComxError comx_gateway_set_data_callback(ComxGateway gateway, ComxDataCallback cb, void* userdata);

/**
 * Set callback for events
 *
 * @param gateway   Gateway handle
 * @param cb        Callback function
 * @param userdata  User context passed to callback
 * @return COMX_OK on success, error code on failure
 */
ComxError comx_gateway_set_event_callback(ComxGateway gateway, ComxEventCallback cb, void* userdata);


/* ============== Transport Direct API ============== */

/**
 * Create a transport directly (without engine)
 *
 * @param type         Transport type ("serial", "tcp", "udp")
 * @param config_json  Configuration as JSON
 * @return Transport handle, or 0 on failure
 */
ComxTransport comx_transport_create(const char* type, const char* config_json);

/**
 * Destroy a transport
 *
 * @param transport  Transport handle
 */
void comx_transport_destroy(ComxTransport transport);

/**
 * Connect transport
 *
 * @param transport  Transport handle
 * @return COMX_OK on success, error code on failure
 */
ComxError comx_transport_connect(ComxTransport transport);

/**
 * Disconnect transport
 *
 * @param transport  Transport handle
 * @return COMX_OK on success, error code on failure
 */
ComxError comx_transport_disconnect(ComxTransport transport);

/**
 * Check if transport is connected
 *
 * @param transport  Transport handle
 * @return true if connected
 */
bool comx_transport_is_connected(ComxTransport transport);

/**
 * Send data via transport
 *
 * @param transport  Transport handle
 * @param data       Data to send
 * @param len        Data length
 * @return Bytes sent, or negative error code
 */
int comx_transport_send(ComxTransport transport, const uint8_t* data, int len);

/**
 * Receive data from transport
 *
 * @param transport   Transport handle
 * @param buffer      Buffer for received data
 * @param max_len     Buffer size
 * @param timeout_ms  Timeout in milliseconds
 * @return Bytes received, or negative error code
 */
int comx_transport_receive(ComxTransport transport, uint8_t* buffer, int max_len, int timeout_ms);


/* ============== Utility API ============== */

/**
 * Get library version string
 *
 * @return Version string (do not free)
 */
const char* comx_version(void);

/**
 * Get API version number
 *
 * @return API version number
 */
int comx_api_version(void);

/**
 * Get error message for error code
 *
 * @param error  Error code
 * @return Error message (caller must free with comx_free)
 */
const char* comx_error_message(ComxError error);

/**
 * Free memory allocated by library
 *
 * @param ptr  Pointer to free
 */
void comx_free(void* ptr);

/**
 * Set log level
 *
 * @param level  Log level (0=off, 1=error, 2=warn, 3=info, 4=debug)
 */
void comx_set_log_level(int level);


#ifdef __cplusplus
}

/* C++ RAII Wrapper */
#ifdef COMX_CPP_WRAPPER

#include <string>
#include <stdexcept>
#include <functional>

namespace comx {

class Exception : public std::runtime_error {
public:
    ComxError code;
    Exception(ComxError err) : std::runtime_error(comx_error_message(err)), code(err) {}
};

class Gateway {
    ComxGateway handle_;
public:
    Gateway(ComxGateway h) : handle_(h) {}

    ComxState state() const { return comx_gateway_state(handle_); }

    void send(const uint8_t* data, size_t len) {
        auto err = comx_gateway_send(handle_, data, static_cast<int>(len));
        if (err != COMX_OK) throw Exception(err);
    }

    int receive(uint8_t* buffer, size_t maxLen, int timeoutMs = 5000) {
        return comx_gateway_receive(handle_, buffer, static_cast<int>(maxLen), timeoutMs);
    }
};

class Engine {
    ComxEngine handle_;
public:
    Engine(const std::string& configJson) {
        handle_ = comx_engine_create_with_config(configJson.c_str());
        if (!handle_) throw Exception(COMX_ERR_CONFIG_INVALID);
    }

    ~Engine() {
        if (handle_) {
            comx_engine_stop(handle_);
            comx_engine_destroy(handle_);
        }
    }

    Engine(const Engine&) = delete;
    Engine& operator=(const Engine&) = delete;

    void start() {
        auto err = comx_engine_start(handle_);
        if (err != COMX_OK) throw Exception(err);
    }

    void stop() {
        comx_engine_stop(handle_);
    }

    Gateway getGateway(const std::string& name) {
        auto gw = comx_engine_get_gateway(handle_, name.c_str());
        if (!gw) throw Exception(COMX_ERR_GATEWAY_NOT_FOUND);
        return Gateway(gw);
    }
};

} // namespace comx

#endif /* COMX_CPP_WRAPPER */
#endif /* __cplusplus */

#endif /* COMX_H */
