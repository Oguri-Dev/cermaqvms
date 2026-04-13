# API HTTP del Sistema de Compresión (Pontón)

**Puerto**: 8087  
**IMPORTANTE**: Este sistema es inmutable. No se puede modificar. El VMS debe adaptarse a su API.

## Endpoints

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/{fileName}/start/{nombreCamara}` | Activa zoom en la cámara indicada (se maximiza a pantalla completa) |
| GET | `/{fileName}/stop` | Desactiva el zoom y restaura la vista de grilla normal |
| GET | `/{fileName}/status` | Retorna el estado actual del sistema (zoom activo, cámara actual, etc.) |
| GET | `/{fileName}/cameras` | Lista todas las cámaras activas con su estado de conexión |
| GET | `/{fileName}/width-factor/{factor}` | Cambia el factor de escala horizontal (rango: 0.5 a 1.0) |

## Ejemplos

| Acción | URL de ejemplo | Notas |
|--------|----------------|-------|
| Zoom a PTZ 1 | `http://localhost:8087/pantalla1/start/PTZ 1` | La cámara se maximiza a pantalla completa |
| Quitar zoom | `http://localhost:8087/pantalla1/stop` | Restaura la vista de grilla normal |
| Ver estado | `http://localhost:8087/pantalla1/status` | Retorna JSON con estado del zoom y cámaras |
| Listar cámaras | `http://localhost:8087/pantalla1/cameras` | Retorna JSON con listado de cámaras y su estado |
| Cambiar w-factor | `http://localhost:8087/pantalla1/width-factor/0.75` | Rango válido: 0.5 a 1.0 |

## Salida RTSP
El stream compuesto se publica en: `rtsp://<ipServer>:8554/<fileName>`

Ejemplo: `rtsp://localhost:8554/pantalla1`

Este stream es luego consumido por el VMS frontend via WHEP (WebRTC-HTTP Egress Protocol) a través de MediaMTX.
