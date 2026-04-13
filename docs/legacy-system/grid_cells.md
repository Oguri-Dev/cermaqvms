# gridCells — Cada cámara en la grilla

Array de objetos dentro de `grid_configuration`. Cada objeto representa una celda/cámara en la grilla.

## Campos

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| idCell | Int | Sí | Posición en la grilla (0-based). Fila = idCell / columns, Columna = idCell % columns |
| name | String | Sí | Nombre descriptivo de la cámara (ej. "PTZ 1", "103-A") |
| ipCamera | String | Sí | IP directa de la cámara |
| ipNvr | String | Sí | IP del NVR al que está conectada la cámara |
| nvrChannel | Int | Sí | Canal del NVR donde está esta cámara |
| user | String | Sí | Usuario para autenticación RTSP (default: "admin") |
| pass | String | Sí | Contraseña para autenticación RTSP |
| mediamtxCamera1 | String | Sí* | Nombre del stream en MediaMTX para flujo primario (requerido si selectFlow=3) |
| mediamtxCamera2 | String | Sí* | Nombre del stream en MediaMTX para flujo secundario (requerido si selectFlow=4) |
| on | Boolean | Sí | true = cámara visible en la grilla, false = celda vacía |
| w_factor | Double | No | Factor de escala horizontal de la cámara (ajusta el ancho dentro de la celda, default: 1.0) |

## Notas
- Los campos `user` y `pass` se heredan del NVR asociado a la cámara
- Los campos `mediamtxCamera1` y `mediamtxCamera2` son nombres de stream en MediaMTX, típicamente formateados como `{nombre_camara}-1` y `{nombre_camara}-2`
- Cuando `on` es `false`, la celda aparece vacía/negra en el mosaico
- `w_factor` permite ajustar el ancho de la imagen de la cámara dentro de la celda (rango: 0.5 a 1.0)
