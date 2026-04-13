# screen_configuration — Documento Raíz

Documento principal que el sistema de compresión del pontón consume para generar los streams compuestos.

## Campos

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| _id | ObjectId | Auto | Identificador único del documento (generado por MongoDB) |
| id_center | ObjectId | Sí | Referencia al centro/sitio al que pertenece esta configuración |
| fileName | String | Sí | Nombre de la pantalla (ej. "pantalla1"). Se usa para la URL RTSP de salida: `rtsp://<ipServer>:8554/<fileName>` |
| active | Boolean | Sí | true = el sistema transmite la grilla. false = se detiene toda transmisión |
| bitrate | Int | Sí | Bitrate de salida del encoder en kbps (ej. 4096) |
| hardwareEncoding | Int | Sí | Modo de codificación: 1=CPU, 2=GPU completo, 3=GPU dec + CPU comp + GPU enc, 4=CPU dec + GPU enc |
| ipServer | String | Sí | IP del servidor donde se publica la salida RTSP (ej. "localhost") |
| grid_configuration | Object | Sí | Objeto anidado con toda la configuración de la grilla (ver grid_configuration.md) |

## Notas
- El campo `fileName` es crítico: define la URL de salida RTSP que el VMS consume via WHEP
- El campo `active` controla si el sistema de compresión transmite o no
- `hardwareEncoding` depende del hardware disponible en el PC del pontón
