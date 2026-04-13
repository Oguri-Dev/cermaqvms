# grid_configuration

Objeto embebido dentro de `screen_configuration` que define la grilla y sus celdas.

## Campos

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| rows | Int | Sí | Número de filas de la grilla (ej. 3) |
| columns | Int | Sí | Número de columnas de la grilla (ej. 5) |
| widthResolution | Int | Sí | Ancho total de la salida en píxeles (ej. 1920) |
| heightResolution | Int | Sí | Alto total de la salida en píxeles (ej. 1080) |
| selectFlow | Int | Sí | Fuente del stream RTSP: 1/2=NVR (primario/secundario), 3/4=MediaMTX camera1/camera2, 5/6=cámara directa |
| fps | Int | Sí | Frames por segundo de la salida (ej. 20) |
| gop | Int | Sí | Group of Pictures del encoder (ej. 25) |
| pc_id | Int | No | Identificador del PC que ejecuta esta grilla |
| gridCells | Array | Sí | Array de objetos, cada uno representando una cámara en la grilla (ver gridCells.md) |
| unionCells | Array | No | Array de celdas unidas, para cámaras que ocupan más de una celda |

## Notas sobre selectFlow
- **1/2**: El stream se toma directamente del NVR (primario/secundario). Usa `ipNvr`, `nvrChannel`, `user`, `pass`
- **3/4**: El stream se toma de MediaMTX. Usa `mediamtxCamera1` / `mediamtxCamera2`
- **5/6**: El stream se toma directamente de la IP de la cámara. Usa `ipCamera`
