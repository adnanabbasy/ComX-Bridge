# ComX-Bridge Production Readiness Guide

This document explains the **Security**, **Reliability**, and **Observability** features added for stable operation of ComX-Bridge in commercial (Production) environments.

## 1. Security

### 1-1. API Authentication (JWT & Multi-User)
ComX-Bridge provides a robust **Stateless Authentication** system using **JWT (JSON Web Token)** and **API Keys**. It supports multiple users with different roles.

*   **Configuration (config.yaml)**:
    ```yaml
    api:
      auth:
        enabled: true
        jwt_secret: "your-super-secret-jwt-signing-key" # Used to sign JWTs
        users:
          - name: "admin-user"
            key: "admin-secret-key-123"
            role: "admin"
          - name: "monitor-user"
            key: "view-secret-key-456"
            role: "viewer"
    ```

*   **Authentication Methods**:
    1.  **JWT (Recommended)**:
        *   **Login**: POST `/api/v1/login` with `{"key": "admin-secret-key-123"}` -> Returns `token`.
        *   **Usage**: Header `Authorization: Bearer <JWT_TOKEN>`.
        *   Supported in REST and gRPC (Metadata `authorization`).
    2.  **API Key (Legacy/Simple)**:
        *   **Usage**: Header `X-API-Key: admin-secret-key-123` or `Authorization: Bearer <API_KEY>`.
        *   Supported in REST and gRPC (Metadata `x-api-key`).

*   **gRPC Security**:
    *   The gRPC server automatically validates tokens/keys via Interceptors.
    *   Clients must send metadata `x-api-key` or `authorization`.

### 1-2. TLS/SSL Encryption (Transport & HTTPS)
Supports TLS for both API Server and Transport Layer (MQTT, TCP).
*   **API HTTPS Configuration**:
    ```yaml
    api:
      tls:
        enabled: true
        cert_file: "./certs/server.crt"
        key_file: "./certs/server.key"
    ```
*   **Transport Layer TLS (Gateways)**:
    ```yaml
    transport:
      tls:
        enabled: true
        ca_file: "./certs/ca.crt" # If mutual auth required
        cert_file: "./certs/client.crt"
        key_file: "./certs/client.key"
    ```

## 2. Reliability & HA (High Availability)

### 2-1. Data Persistence
To prevent data loss during network outages, messages are buffered in an internal database (SQLite).
*   **Working Principle**: Transmission Failure -> Save to DB -> Background Retry Loop -> Delete on Success.
*   **Configuration**:
    ```yaml
    persistence:
      enabled: true
      path: "./data/buffer.db"
    ```

### 2-2. Failover
Active-Standby structure allows a standby server to automatically take over in case of a server failure.
*   **Configuration**:
    ```yaml
    cluster:
      enabled: true
      role: "primary" # or "secondary"
      peer_ip: "192.168.1.50"
      port: 9999
    ```
*   **Operation**:
    1.  **Primary**: Sends Heartbeat periodically.
    2.  **Secondary**: Listens for Heartbeat. If none received for 3 seconds, promotes self to Active and starts gateways.

### 2-3. Configuration Validation
Strictly checks for not only syntax errors in `config.yaml` but also logical errors (missing required fields, port range exceeded, etc.) upon engine startup.

## 3. Observability

### 3-1. Prometheus Metrics
Exposes engine status and performance metrics in standard Prometheus format.
*   **Endpoint**: `GET http://<host>:<api_port>/metrics`
*   **Key Metrics**:
    *   `comx_gateway_packets_total{gateway, direction, status}`
    *   `comx_gateway_errors_total`
    *   `comx_connected_gateways_total`
    *   `comx_engine_uptime_seconds`

### 3-2. Structured Logging
Supports structured logging (JSON) for easy integration with ELK Stack or Loki.
*   **Configuration**:
    ```yaml
    logging:
      level: "info" # debug, info, warn, error
      format: "json" # text, json
      output: "stdout" # stdout, file
      file: "./logs/comx.log"
    ```
