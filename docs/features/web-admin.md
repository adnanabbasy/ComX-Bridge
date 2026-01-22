# Web Admin Dashboard

ComX-Bridge provides a **web-based admin dashboard** to visually monitor and manage the engine status.

## 1. Key Features
*   **Real-time Monitoring**:
    *   Overall Gateway Status (Online/Error/Stopped)
    *   Messages Per Second (TPS), Error Counts
    *   AI Engine and Persistence Status
*   **Visualization**: Intuitive Card UI and status lights.
*   **Responsive Design**: Supports PC and Mobile (Tablet) environments.
*   **Dark Mode**: High readability, sleek dark theme (Glassmorphism applied).

## 2. Usage

### 2-1. Configuration
The API server must be enabled in `config.yaml`.

```yaml
api:
  enabled: true
  port: 8080
```

### 2-2. Build Frontend (First time only)
The dashboard is built with React, so it needs to be built into static files before running the engine.

```bash
cd web/admin
npm install
npm run build
```
Once built, the `web/admin/dist` folder is created.

### 2-3. Access
After running the engine, access the following URL in your browser.

*   URL: `http://localhost:8080/admin/` (Note the trailing slash `/`)

## 3. Security
If API Authentication (`api.auth.enabled: true`) is configured:
*   **Backend**: All API calls require `Authorization: Bearer <Token>` or `X-API-Key`.
*   **Frontend**: Currently, the dashboard does not have a comprehensive Login UI. You may need to disable auth for dev monitoring or manually set headers if using custom builds. (Login UI coming in v1.1)
