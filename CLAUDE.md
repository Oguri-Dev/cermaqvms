# OMNIFISH VMS — Cermaq

Sistema de Video Management para centros de cultivo de salmón de Cermaq. Desarrollado por Omnifish.

Reemplaza a un VMS anterior que funcionaba mal. **Alcance clave**: el VMS NO comprime; usa el compresor existente del centro (**GST-Grid**). El front solo (1) obtiene el flujo compuesto vía WHEP, (2) hace zoom in-stream a una celda con doble click (instantáneo, vía el API del compresor), y (3) envía comandos PTZ. Topología: pontón/centro en el mar (compresor + Mongo + cámaras/NVRs) → Starlink → VPN → sala remota en Puerto Montt donde corre el front.

## Stack

- **Frontend**: React 19 + Vite 8 + React Router 7 (puerto 5173)
- **Backend**: Go (Chi router) + MongoDB driver (puerto 8080)
- **Database local**: MongoDB (`vms_cermaq`) — config del front, usuarios, historial de uso
- **Database del centro**: MongoDB remota (`camancha_vsmweb`) — `screen_configuration` real (lectura), `cameras`/`nvr`
- **Compresor**: GST-Grid (GStreamer) en el equipo Linux del centro, API HTTP en :8087 (zoom in-stream)
- **Video**: WebRTC vía WHEP servido por el centro en :8889
- **PTZ**: `ptz-service` (módulo Go propio) en el equipo del centro, :8088

## Levantar el proyecto

```bash
# Backend (puerto 8080)
cd backend
go run .

# Frontend (puerto 5173, accesible desde otros equipos de la red)
cd frontend
npm install
npm run dev -- --host
```

El `ptz-service` corre en el equipo de compresión del centro (Linux), no en local. Ver `ptz-service/README.md` para despliegue.

### Variables de entorno (backend)

| Variable | Default | Descripción |
|----------|---------|-------------|
| PORT | 8080 | Puerto del API |
| MONGO_URI | mongodb://localhost:27017 | URI de MongoDB local |
| MONGO_DB_NAME | vms_cermaq | Base local |
| CENTER_MONGO_URI | mongodb://10.1.1.229:27017 | Mongo del centro (lectura de screen_configuration) |
| CENTER_DB_NAME | camancha_vsmweb | Base del centro (¡"vsm", no "vms"!) |
| CENTER_HOST | 10.1.1.229 | Host del centro: zoom GST-Grid (:8087) y WHEP (:8889) |
| CENTER_PTZ_HOST | (vacío) | Override del host del servicio PTZ (:8088); vacío = CENTER_HOST |
| AUTH_SECRET | (vacío) | Secreto de firma JWT; si vacío se genera y persiste en Mongo |

La conexión al centro también es editable en caliente desde el front (Config → Servidor), y se guarda como singleton en la colección local `center_config`.

### Variables de entorno (frontend)

| Variable | Default | Descripción |
|----------|---------|-------------|
| VITE_API_URL | http://`<hostname>`:8080/api | URL base del API; si no se setea, se deriva del host del navegador (permite acceso desde otros equipos) |

## Arquitectura

```
frontend/          React SPA
├── src/
│   ├── api/         grids.js (API client + token), whep.js (WebRTC WHEP)
│   ├── auth/        AuthContext.jsx (sesión, roles)
│   ├── hooks/       usePtz.js, useBootstrap.js, useUsageReporter.js
│   ├── pages/       Monitor, Config, Screen, Login
│   └── components/  GridScreen, GridOverlay, Navbar, ScreenMapper, OnDemandViewer,
│                    PTZControls, PTZCellOverlay

backend/           Go REST API
├── auth/          jwt.go (HS256 propio, stdlib), password.go (bcrypt)
├── config/        Variables de entorno
├── database/      Conexión Mongo local + centro (reconexión en caliente)
├── models/        grid.go (Device, Grid, Stream, ScreenConfig, User, CenterConfig, UsageDaily), ponton.go
├── handlers/      CRUD + auth + center (zoom/PTZ proxy/import) + bootstrap + usage + ponton sync
└── routes/        Chi router (grupos RequireAuth / RequireAdmin)

ptz-service/       Módulo Go independiente (corre en el equipo Linux del centro)
├── api.go         Rutas: /health, /cameras, /ptz/{cam}/*, /compression/restart, /bootstrap
├── controller.go  Dispatch ISAPI→ONVIF, dead-man switch
├── isapi.go       Hikvision ISAPI (digest)
├── onvif.go       ONVIF (WS-Security) fallback
├── database.go    Resuelve cámaras/credenciales desde la Mongo del centro
├── compression.go Reinicio de gst-grid / mediamtx / self
└── deploy/        restart.conf (drop-in systemd), sudoers-vms-ptz

ptzControll/         Servicio PTZ legado (reemplazado por ptz-service/, NO versionado)
docs/api_http.md     Spec del API de zoom del compresor GST-Grid
docs/legacy-system/  Documentación del formato screen_configuration del pontón
```

## Modelo de datos

### Local (`vms_cermaq`)
- **Device**: NVR (name, ip, user, pass) o Camera (name, ip, nvr_id, nvr_channel, camera_type, cage_name, ondemand_mode, **has_ptz**). El front edita has_ptz/OSD; las cámaras se importan del centro.
- **Grid**: name, type (submarine/dome), rows, cols (solo lectura en el front).
- **Stream**: grid + celdas + campos pontón (file_name, ip_server, is_active, bitrate, etc.). `file_name` define la URL RTSP de salida.
- **ScreenConfig** (singleton): center_name, screens[] (mapeo stream → monitor), osd_size/position, grid_color.
- **User**: username, password_hash (bcrypt), role (`admin` | `operator`).
- **CenterConfig** (singleton): mongo_uri, db_name, host del centro.
- **UsageDaily**: `_id` = fecha local (YYYY-MM-DD), bytes_by_stream, uptime_slots (slots de 30s únicos).

### Centro (`camancha_vsmweb`, lectura)
- **screen_configuration**: generada en la puesta en marcha del centro, FUERA del VMS. El VMS la lee; la única escritura permitida es un `$set` quirúrgico del campo `active`. Tiene campos que los modelos Go no conocen (rowspan, hasOverlay, content, etc.) y `gridCells` no siempre = rows×cols. **Nunca hacer ReplaceOne** (borraría campos del sistema legado).
- **cameras** / **nvr**: el ptz-service resuelve credenciales de aquí.

## Compresor del centro (GST-Grid) — INMUTABLE

GStreamer, ramas independientes por cámara (una caída no tumba el mosaico). Zoom instantáneo dentro del mismo stream: `GET :8087/{fileName}/start/{cameraId}` y `/stop` (matching flexible de nombre). El front solo oculta/muestra el overlay de grilla. Spec en `docs/api_http.md`. El compresor nuevo eliminó MediaMTX (por eso el zoom es instantáneo, sin reconexión).

### Sync al pontón (legado, coexiste)
El backend aún puede generar el documento `screen_configuration` en formato pontón al crear/actualizar un Stream (`POST/PUT /api/streams`) y escribirlo en Mongo. BSON tags camelCase exactos: `fileName`, `gridCells`, `idCell`, `ipCamera`, `ipNvr`, `nvrChannel`, `mediamtxCamera1/2`, `widthResolution`, `heightResolution`, `selectFlow`, `hardwareEncoding`. **No cambiar estos nombres.** En operación real con el centro, el flujo es read-mostly de `screen_configuration`.

## Autenticación

- Login JWT HS256 (implementación propia, solo stdlib) + bcrypt. Roles: `admin` | `operator`.
- Semilla `admin`/`admin` al primer arranque (cambiarla). Secreto persistido en colección `app_secrets`.
- Sesión recordable (localStorage vs sessionStorage) — pensado para el PC del wall.
- Alcance: **solo red local cerrada**. CORS/HTTPS no están endurecidos para internet; si se expone, reabrir esa conversación.
- CORS incluye el header `Authorization` (sin él, el preflight bloquea el login).

## API REST

Rutas bajo `/api`. Grupos: público (login), `RequireAuth` (admin u operador), `RequireAdmin`.

### Auth / Usuarios
| Método | Ruta | Acceso |
|--------|------|--------|
| POST | /auth/login | público |
| GET | /auth/me | auth |
| GET/POST/PUT/DELETE | /users[/:id] | admin |

### Centro (operación)
| Método | Ruta | Acceso | Descripción |
|--------|------|--------|-------------|
| GET | /center/screens | auth | Lista screen_configuration (con has_ptz local) |
| GET | /center/screens/:fileName | auth | Una pantalla |
| POST | /center/screens/:fileName/zoom/:cellName | auth | Zoom in-stream (maximiza celda) |
| POST | /center/screens/:fileName/unzoom | auth | Restaura la grilla |
| GET | /center/screens/:fileName/zoom-status | auth | Estado del zoom |
| GET | /center/screens/:fileName/health | auth | Estado por cámara (celdas caídas) |
| PUT | /center/screens/:fileName/active | admin | Activar/detener transmisión ($set quirúrgico) |
| GET/POST | /center/ptz/:camera/:action | auth | move / stop / goto / presets (proxy a ptz-service) |
| POST | /center/compression/restart | auth | Reinicia gst-grid |
| POST | /center/bootstrap | auth | Secuencia de arranque (deduplicada por ventana de tiempo) |
| GET/PUT | /center-config[/status] | admin | Conexión al centro |
| POST | /center/import | admin | Importa cámaras/grillas/streams del centro |

### Streams / Grids / Devices
| Método | Ruta | Acceso |
|--------|------|--------|
| GET/POST/PUT/DELETE | /devices[/:id] | admin |
| GET/POST/PUT/DELETE | /grids[/:id] | admin |
| GET/POST/PUT/DELETE | /streams[/:id] | admin |
| GET | /streams/:id/full | auth | Stream con grid y cámaras resueltas |
| POST | /streams/:id/sync | admin | Sync manual al pontón (legado) |
| GET | /streams/:id/screen-config | admin | Preview documento pontón |

### Screen Config / PTZ local / Usage
| Método | Ruta | Acceso |
|--------|------|--------|
| GET | /screen-config | auth |
| PUT | /screen-config | admin |
| POST | /streams/:id/ptz/:row/:col/{move,preset} | auth |
| POST | /usage/heartbeat | auth | Reporta uptime + bytes por stream |
| GET | /usage?from&to | admin | Reporte por rango |
| GET | /usage/export?from&to | admin | CSV |

### ptz-service (:8088, en el equipo del centro)
`/health`, `/cameras`, `/ptz/{camera}/{move,stop,goto,presets}`, `/compression/restart`, `/bootstrap`. El front nunca llama directo: pasa por el proxy del backend (`/api/center/...`). Dead-man switch: `move` arma auto-stop; el front renueva mientras el botón está presionado.

## Vistas del frontend

1. **Login**: usuario/contraseña, "Recordar sesión".
2. **Monitor** (`/`, `/monitor`): vista principal con toolbar, selector de layout/grillas, fullscreen. (admin va a Config por defecto; operador al launcher del wall).
3. **Config** (`/config`, solo admin): 7 secciones con sidebar de iconos — Servidor, Dispositivos, Grillas, Streams, Pantallas, **Monitoreo**, Usuarios.
4. **Screen** (`/screen/:slot`): ventana individual por monitor físico, carga stream vía WHEP. Es la que reporta consumo de datos.

El wall se abre con "Abrir Pantallas" en la Navbar (Multi-Screen Window Placement API), una ventana por monitor.

## Video

- El centro publica el stream compuesto como WHEP en `http://<CENTER_HOST>:8889/<fileName>/`.
- El front consume vía WHEP (`api/whep.js`, RTCPeerConnection).
- Aspecto: la resolución real del mosaico (width/height del centro) define la proporción; sin ella, 16:9 por CSS. Hay grillas no-16:9 (ej. chonos 1350×1406), así que no se fuerza.
- Double-click en celda: zoom in-stream vía el compresor (oculta el overlay). Si no hay centro, cae al visor on-demand.
- Right-click / hover en celda PTZ: controles PTZ (overlay).
- **Reconexión WHEP**: backoff + watchdog de frames congelados (`requestVideoFrameCallback`); muestra spinner "Reconnecting". Resuelve el congelamiento tras reiniciar la compresión (antes requería F5).

## Bootstrap (arranque del wall)

Al abrir la app (ventana principal, con sesión, no las `/screen/`), el front llama `POST /api/center/bootstrap` una vez por sesión. El backend lo deduplica (ventana de 5 min) y proxea al `/bootstrap` del ptz-service, que reinicia en orden: **gst-grid** → **mediamtx** (solo si la unit existe) → **el propio ptz-service** (en background, systemd lo levanta por Restart=always). Requiere sudoers NOPASSWD (ver `ptz-service/deploy/`).

## Monitoreo de uso (Starlink)

Para vigilar el consumo del enlace Starlink. Cada ventana `/screen/` lee `bytesReceived` de su RTCPeerConnection y envía heartbeats cada 30s con el delta de bytes y el slot del día (`useUsageReporter.js`). El backend acumula por stream (`$inc`) y registra slots de uptime únicos (`$addToSet`), así varias ventanas no inflan las horas. Resumen diario en `usage_daily`. Config → Monitoreo: rango de fechas, totales, tabla por día y exportación CSV.

## Notas de desarrollo

- **Branding**: "OMNIFISH VMS" (Omnifish = desarrollador, Cermaq = cliente).
- **Read-mostly del centro**: el front no agrega cámaras/NVRs/grillas a mano; se importan del centro. La única escritura al Mongo del centro es el `$set` de `active`.
- **4K**: textos/iconos de Config dimensionados para pantallas 4K.
- **Multi-screen**: cada stream abre su propia ventana en un monitor físico separado.
- **Grid overlay**: color configurable (Config → Pantallas), buena visibilidad sobre fondos submarinos.
- **Entorno real (Camancha)**: 10.1.1.229 (también 192.168.6.220). Cámaras J301–J314; PTZ probada en vivo. El enlace al centro es intermitente.
- **Trampa Mongo**: structs embebidos del driver necesitan `bson:",inline"`.
