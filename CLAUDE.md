# OMNIFISH VMS — Cermaq

Sistema de Video Management para centros de cultivo de salmón de Cermaq. Desarrollado por Omnifish.

## Stack

- **Frontend**: React 19 + Vite 8 + React Router 7
- **Backend**: Go (Chi router) + MongoDB driver
- **Database**: MongoDB (`vms_cermaq`)
- **Video**: WebRTC via WHEP (WebRTC-HTTP Egress Protocol) desde MediaMTX
- **Streaming**: MediaMTX (puerto 8889 WHEP, puerto 8554 RTSP)

## Levantar el proyecto

```bash
# Backend (puerto 8080)
cd backend
go run .

# Frontend (puerto 5173)
cd frontend
npm install
npm run dev
```

### Variables de entorno (backend)

| Variable | Default | Descripción |
|----------|---------|-------------|
| PORT | 8080 | Puerto del API |
| MONGO_URI | mongodb://localhost:27017 | URI de MongoDB |
| MONGO_DB_NAME | vms_cermaq | Nombre de la base de datos |

### Variables de entorno (frontend)

| Variable | Default | Descripción |
|----------|---------|-------------|
| VITE_API_URL | http://localhost:8080/api | URL base del API backend |

## Arquitectura

```
frontend/          React SPA
├── src/
│   ├── api/       grids.js (API client), whep.js (WebRTC WHEP)
│   ├── pages/     Launch, Monitor, Config, Screen
│   └── components/ GridScreen, GridOverlay, Navbar, ScreenMapper, OnDemandViewer, PTZControls

backend/           Go REST API
├── config/        Variables de entorno
├── database/      Conexión MongoDB
├── models/        grid.go (Device, Grid, Stream, ScreenConfig), ponton.go (formato pontón)
├── handlers/      CRUD + ponton sync + PTZ
└── routes/        Chi router setup

docs/legacy-system/  Documentación del sistema de compresión inmutable del pontón
```

## Modelo de datos

### Device (NVR o Cámara)
- **NVR**: name, ip, user (RTSP), pass (RTSP)
- **Camera**: name, ip, nvr_id, nvr_channel, camera_type (submarina/PTZ), cage_name, mediamtx_camera1/2, ondemand_mode (nvr/direct), has_ptz

### Grid (Layout template)
- name, type (submarine/dome), rows, cols

### Stream (Grid + cámaras + compresión)
- Referencia a Grid + celdas con camera assignments
- Campos pontón: file_name, ip_server, is_active, bitrate, hardware_encoding, width/height_resolution, select_flow, fps, gop, pc_id
- `file_name` es crítico: define la URL RTSP de salida (`rtsp://<ipServer>:8554/<fileName>`)

### ScreenConfig (Singleton)
- center_name, screens[] (mapeo stream → monitor físico)

## Sistema de compresión del pontón (INMUTABLE)

El pontón tiene un sistema de compresión que **NO se puede modificar**. Lee documentos `screen_configuration` desde MongoDB y genera streams compuestos H264.

### Sync automático
Al crear/actualizar un Stream, el backend genera automáticamente el documento `screen_configuration` en formato pontón y lo escribe en la colección `screen_configuration` de MongoDB. El pontón lo lee de ahí.

### Endpoints de sync
- `POST /api/streams/{id}/sync` — Sincronización manual al pontón
- `GET /api/streams/{id}/screen-config` — Preview del documento sin guardarlo

### Formato del documento pontón
Los BSON tags son camelCase exactos: `fileName`, `gridCells`, `idCell`, `ipCamera`, `ipNvr`, `nvrChannel`, `mediamtxCamera1`, `mediamtxCamera2`, `widthResolution`, `heightResolution`, `selectFlow`, `hardwareEncoding`. **No cambiar estos nombres**.

Ver `docs/legacy-system/` para documentación completa del formato.

## API REST

### Devices
| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | /api/devices | Listar todos |
| POST | /api/devices | Crear NVR o cámara |
| GET | /api/devices/:id | Obtener uno |
| PUT | /api/devices/:id | Actualizar |
| DELETE | /api/devices/:id | Eliminar |

### Grids
| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | /api/grids | Listar |
| POST | /api/grids | Crear |
| GET | /api/grids/:id | Obtener |
| PUT | /api/grids/:id | Actualizar |
| DELETE | /api/grids/:id | Eliminar |

### Streams
| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | /api/streams | Listar |
| POST | /api/streams | Crear (+ auto sync pontón) |
| GET | /api/streams/:id | Obtener |
| PUT | /api/streams/:id | Actualizar (+ auto sync pontón) |
| DELETE | /api/streams/:id | Eliminar |
| GET | /api/streams/:id/full | Stream con grid y cámaras resueltas |
| POST | /api/streams/:id/sync | Sync manual al pontón |
| GET | /api/streams/:id/screen-config | Preview documento pontón |

### PTZ
| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | /api/streams/:id/ptz/:row/:col/move | Mover cámara PTZ |
| POST | /api/streams/:id/ptz/:row/:col/preset | Preset PTZ |

### Screen Config (singleton)
| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | /api/screen-config | Obtener configuración de pantallas |
| PUT | /api/screen-config | Guardar configuración de pantallas |

## Vistas del frontend

1. **Launch** (`/`): Botón "Iniciar" que abre ventanas en monitores físicos usando Multi-Screen Window Placement API
2. **Monitor** (`/monitor`): Vista principal con toolbar, selector de layout/grillas, fullscreen
3. **Config** (`/config`): 4 secciones (Dispositivos, Grillas, Streams, Pantallas) con sidebar vertical de iconos
4. **Screen** (`/screen/:streamId`): Ventana individual por monitor físico, carga stream via WHEP

## Video

- Los streams compuestos del pontón se publican en RTSP (`rtsp://<ip>:8554/<fileName>`)
- MediaMTX republica como WHEP en `http://<ip>:8889/<fileName>/`
- El frontend consume via WHEP (`api/whep.js`) creando RTCPeerConnection
- Aspecto 16:9 forzado con CSS `aspect-ratio` + `min()`/`max()` para mantener proporción en cualquier monitor
- Double-click en celda abre on-demand directo a la cámara (no del mosaico)
- Right-click en celda PTZ abre controles PTZ

## Notas de desarrollo

- **Sin autenticación**: acceso directo, no hay login
- **Branding**: "OMNIFISH VMS" (Omnifish = desarrollador, Cermaq = cliente)
- **Entrada manual**: todos los datos de NVR y cámaras se ingresan a mano, no hay auto-discovery
- **4K**: los textos/iconos del menú de configuración están dimensionados para pantallas 4K
- **Multi-screen**: cada stream abre su propia ventana del navegador en un monitor físico separado
- **Grid overlay**: líneas blancas sólidas (0.8 opacidad) con labels de celda, buena visibilidad sobre fondos submarinos
