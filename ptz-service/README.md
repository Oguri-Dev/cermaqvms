# VMS PTZ Service

Servicio de control PTZ del OMNIFISH VMS. Corre en el **equipo de compresión del centro**, junto a la MongoDB que tiene las cámaras y sus credenciales. El front nunca ve credenciales: solo envía el id o nombre de la cámara.

## Protocolos

- **ISAPI (Hikvision)** con digest auth, vía NVR (`ipNvr` + `nvrChannel`) o directo a la cámara. Es el protocolo primario (mismo orden que el sistema anterior).
- **ONVIF** (WS-Security UsernameToken digest) como fallback, directo a `ipCamera`.
- El protocolo que funciona se cachea por cámara.

## Dead-man switch

`POST /ptz/{cámara}/move` arma un temporizador de auto-stop (`STOP_TIMEOUT_MS`, default 2500 ms). El front renueva el comando cada ~800 ms mientras el operador mantiene presionado el control; si la renovación cesa (pestaña cerrada, red caída, comando enclavado), el servicio **detiene la cámara solo**.

## API (puerto 8088)

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | /health | Estado del servicio + Mongo |
| GET | /cameras | Cámaras de la base (sin credenciales) |
| POST | /ptz/{cámara}/move | `{"pan":-1..1,"tilt":-1..1,"zoom":-1..1}` — renueva el dead-man |
| POST | /ptz/{cámara}/stop | Stop inmediato |
| GET | /ptz/{cámara}/presets | Presets configurados |
| POST | /ptz/{cámara}/goto | `{"preset":"1"}` |

`{cámara}` acepta `_id` de Mongo o nombre con matching flexible (case-insensitive, sin espacios/guiones — igual que el zoom del compresor).

## Variables de entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| PTZ_PORT | 8088 | Puerto del servicio |
| MONGO_URI | mongodb://localhost:27017 | Mongo del centro |
| MONGO_DB_NAME | camancha_vsmweb | Base con la colección `cameras` |
| STOP_TIMEOUT_MS | 2500 | Dead-man: auto-stop sin renovación |
| CAMERA_TIMEOUT_MS | 4000 | Timeout HTTP hacia cámaras/NVRs |
| COMPRESSION_RESTART_CMD | `systemctl restart gst-grid` | Comando de reinicio de la compresión (separado por espacios) |
| RESTART_TIMEOUT_MS | 15000 | Timeout del comando de reinicio |

## Reinicio de la compresión

`POST /compression/restart` ejecuta `COMPRESSION_RESTART_CMD`. Como el comando por
defecto es `systemctl restart gst-grid`, el servicio PTZ necesita permiso para
ejecutarlo. Dos opciones:

**A) sudoers sin contraseña (recomendado, servicio como usuario normal):**

```bash
# /etc/sudoers.d/vms-ptz   (editar con: sudo visudo -f /etc/sudoers.d/vms-ptz)
omnifish ALL=(root) NOPASSWD: /usr/bin/systemctl restart gst-grid
```

Y configurar el comando con sudo:
```
COMPRESSION_RESTART_CMD=sudo systemctl restart gst-grid
```

**B) Correr el ptz-service como root** (en la unit: `User=root`) — más simple, menos acotado.

## Despliegue en el equipo de compresión (Linux)

```bash
# binario ya cross-compilado en este directorio
scp vms-ptz-linux usuario@<equipo-compresion>:/opt/vms-ptz/
# en el equipo:
chmod +x /opt/vms-ptz/vms-ptz-linux
MONGO_DB_NAME=camancha_vsmweb /opt/vms-ptz/vms-ptz-linux
```

Para systemd (sobrevive reinicios):

```ini
# /etc/systemd/system/vms-ptz.service
[Unit]
Description=VMS PTZ Service
After=network.target mongod.service

[Service]
ExecStart=/opt/vms-ptz/vms-ptz-linux
Environment=MONGO_DB_NAME=camancha_vsmweb
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now vms-ptz
```

## Desarrollo

El backend del VMS proxea `/api/center/ptz/*` hacia `CENTER_HOST:8088`. Para apuntar a un servicio PTZ corriendo local (sin desplegar al centro), exportar `CENTER_PTZ_HOST=localhost` antes de levantar el backend.
