# API HTTP - GST-Grid

## Información General

| Parámetro | Valor |
|-----------|-------|
| Puerto | `8087` (configurable en `include/http_server.h`) |
| Protocolo | HTTP GET |
| Formato de respuesta | JSON |
| CORS | Habilitado (`Access-Control-Allow-Origin: *`) |

---

## Endpoints

### 1. Activar Zoom en Cámara

```
GET /{fileName}/start/{cameraId}
```

Activa zoom (maximizar) una cámara específica a pantalla completa. 

**Parámetros:**
- `fileName`: Nombre de la transmisión (ej: `pantalla1`, `Pantalla2`). Debe coincidir con el campo `fileName` del documento MongoDB (sin espacios).
- `cameraId`: Nombre o identificador de la cámara. Se busca con coincidencia flexible:
  - Exacta (case-insensitive)
  - Normalizada (sin espacios, guiones, minúsculas)
  - Parcial (substring)

**Ejemplo:**
```bash
curl http://localhost:8087/pantalla1/start/Camara-01
```

**Respuesta exitosa (200):**
```json
{
  "success": true,
  "message": "Zoom activado en cámara 'Camara 01' (índice 0 de 4)",
  "data": {
    "fileName": "pantalla1",
    "requestedId": "Camara-01",
    "realCameraName": "Camara 01",
    "cameraIndex": 0,
    "zoomActive": true,
    "cameraUrl": "rtsp://admin:pass@192.168.1.10:554/Streaming/Channels/101",
    "totalCameras": 4,
    "rtspMapSize": 4
  }
}
```

**Errores posibles:**
- `400`: Parámetros faltantes
- `404`: Transmisión o cámara no encontrada
- `500`: No se pudo activar el zoom (branch sin sinkpad o cámara desconectada)

---

### 2. Desactivar Zoom (Restaurar Grilla)

```
GET /{fileName}/stop
```

Restaura la vista de grilla normal, desactivando cualquier zoom activo.

**Ejemplo:**
```bash
curl http://localhost:8087/pantalla1/stop
```

**Respuesta exitosa (200):**
```json
{
  "success": true,
  "message": "Zoom desactivado - grilla normal restaurada",
  "data": {
    "fileName": "pantalla1",
    "zoomActive": false
  }
}
```

---

### 3. Consultar Estado del Sistema

```
GET /{fileName}/status
```

Retorna información sobre el estado actual del zoom y la transmisión.

**Ejemplo:**
```bash
curl http://localhost:8087/pantalla1/status
```

**Respuesta exitosa (200):**
```json
{
  "success": true,
  "message": "Estado actual del sistema",
  "data": {
    "fileName": "pantalla1",
    "zoomActive": false,
    "currentCameraIndex": -1,
    "activeCameras": 4,
    "widthFactor": 1.0
  }
}
```

---

### 4. Listar Cámaras Activas

```
GET /{fileName}/cameras
```

Lista todas las cámaras configuradas con su estado de conexión, URL RTSP, tipo y actividad.

**Ejemplo:**
```bash
curl http://localhost:8087/pantalla1/cameras
```

**Respuesta exitosa (200):**
```json
{
  "success": true,
  "message": "Lista de 4 cámaras activas (sincronizada con MongoDB)",
  "data": {
    "fileName": "pantalla1",
    "activeCameras": [
      {
        "cameraName": "Camara 01",
        "index": 0,
        "rtspUrl": "rtsp://admin:pass@192.168.1.10:554/Streaming/Channels/101",
        "cameraType": "submarina",
        "connected": true,
        "hadVideo": true,
        "lastActivityMs": 150,
        "inactive": false
      },
      {
        "cameraName": "Camara 02",
        "index": 1,
        "rtspUrl": "rtsp://admin:pass@192.168.1.10:554/Streaming/Channels/201",
        "connected": false,
        "hadVideo": false,
        "inactive": true,
        "lastError": "pad-removed"
      }
    ],
    "totalActiveCameras": 4,
    "rtspMapSize": 4,
    "branchesSize": 4,
    "zoomActive": false
  }
}
```

**Campos por cámara:**
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `cameraName` | string | Nombre de la cámara |
| `index` | int | Índice en la grilla |
| `rtspUrl` | string | URL RTSP de la fuente |
| `cameraType` | string | Tipo (submarina, ptz, etc.) |
| `connected` | bool | Si la cámara reporta como conectada |
| `hadVideo` | bool | Si alguna vez recibió un pad de video |
| `lastActivityMs` | int | Milisegundos desde el último buffer recibido |
| `inactive` | bool | `true` si supera `GRID_INACTIVITY_MS` (25s por defecto) |
| `lastError` | string | Último error reportado por la rama |

---

### 5. Ajustar Factor de Ancho

```
GET /{fileName}/width-factor/{factor}
```

Ajusta el factor de escalado de ancho de las cámaras submarinas en la grilla.

**Parámetros:**
- `factor`: Valor decimal entre `0.5` y `1.0`

**Ejemplo:**
```bash
curl http://localhost:8087/pantalla1/width-factor/0.75
```

**Respuesta exitosa (200):**
```json
{
  "success": true,
  "message": "Width factor actualizado a 0.75 para transmisión 'pantalla1'",
  "data": {
    "fileName": "pantalla1",
    "previousFactor": 1.0,
    "newFactor": 0.75,
    "camerasRepositioned": 4
  }
}
```

---

## Formato de Error

Todas las respuestas de error siguen el formato:

```json
{
  "success": false,
  "error": "Descripción del error"
}
```
