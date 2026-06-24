# OMNIFISH VMS — Cermaq

Sistema de **Video Management** para centros de cultivo de salmón de Cermaq, desarrollado por **Omnifish**.

Permite operar de forma remota el muro de monitores (*wall*) de una sala de control: visualizar el mosaico de cámaras de un centro, maximizar cualquier cámara con doble click, controlar domos PTZ y vigilar el consumo del enlace satelital.

## ¿Qué hace?

El VMS **no comprime video**: se apoya en el compresor que ya existe en el centro (**GST-Grid**), que entrega un único stream compuesto (mosaico) por WebRTC. Esto ahorra ancho de banda sobre el enlace Starlink. El front se encarga de:

1. **Mostrar el mosaico** compuesto vía WHEP (un solo flujo por pantalla).
2. **Zoom in-stream**: doble click en una celda maximiza esa cámara dentro del mismo flujo (instantáneo, sin reconexión) usando el API del compresor; el front solo oculta la grilla.
3. **Control PTZ**: mover domos y llamar presets desde la propia celda.
4. **Reinicio remoto** de la compresión y arranque ordenado del muro.
5. **Monitoreo de uso**: horas de operación y datos consumidos por stream, para vigilar el enlace satelital.

### Topología

```
Centro/pontón (en el mar)                 Sala de control (Puerto Montt)
┌────────────────────────────┐            ┌──────────────────────────┐
│ Cámaras / NVRs             │            │  Frontend (React)        │
│ Compresor GST-Grid  :8087  │  Starlink  │  Backend  (Go)    :8080  │
│ WHEP (video)        :8889  │◄──VPN─────►│                          │
│ MongoDB del centro :27017  │            │  MongoDB local    :27017 │
│ ptz-service         :8088  │            │  Muro de monitores       │
└────────────────────────────┘            └──────────────────────────┘
```

## Stack

| Capa | Tecnología |
|------|------------|
| Frontend | React 19 · Vite 8 · React Router 7 |
| Backend | Go · Chi router · driver MongoDB |
| Servicio PTZ | Go (módulo aparte, corre en el equipo del centro) |
| Base de datos | MongoDB (local + del centro) |
| Video | WebRTC vía WHEP |
| Autenticación | JWT (HS256) + bcrypt, roles admin/operador |

## Estructura del repositorio

```
frontend/      SPA React (visor, configuración, login, ventanas del muro)
backend/       API REST en Go (auth, integración con el centro, monitoreo)
ptz-service/   Servicio PTZ + control de servicios — corre en el equipo Linux del centro
docs/          Spec del compresor (api_http.md) y formato del sistema legado
```

## Cómo levantarlo

Requisitos: Go 1.21+, Node 18+, MongoDB.

```bash
# Backend (puerto 8080)
cd backend
go run .

# Frontend (puerto 5173, accesible desde otros equipos de la red)
cd frontend
npm install
npm run dev -- --host
```

Al primer arranque se crea un usuario administrador semilla (cambiarlo después de entrar). La conexión al centro se configura desde la propia interfaz (Config → Servidor) o por variables de entorno.

El `ptz-service` se despliega por separado en el equipo de compresión del centro — ver [`ptz-service/README.md`](ptz-service/README.md).

### Variables de entorno principales (backend)

| Variable | Descripción |
|----------|-------------|
| `MONGO_URI` / `MONGO_DB_NAME` | Base de datos local (config, usuarios, historial) |
| `CENTER_MONGO_URI` / `CENTER_DB_NAME` | Mongo del centro (lectura de `screen_configuration`) |
| `CENTER_HOST` | Host del centro: zoom del compresor (:8087) y WHEP (:8889) |
| `AUTH_SECRET` | Secreto de firma JWT (si se omite, se genera y persiste) |

## Funcionalidades

- **Visor del muro** con layouts configurables y una ventana del navegador por monitor físico (Multi-Screen Window Placement API).
- **Zoom in-stream** y **controles PTZ** por celda (con *dead-man switch*: si el comando de movimiento deja de renovarse, la cámara se detiene sola).
- **Reconexión automática** de los streams WHEP con detección de imagen congelada (resuelve el congelamiento tras reiniciar la compresión, sin recargar).
- **Configuración**: dispositivos, grillas, streams, mapeo de pantallas, color de la grilla, usuarios y conexión al centro.
- **Bootstrap del muro**: al abrir la app reinicia los servicios del centro en orden, una sola vez por sesión.
- **Monitoreo del enlace Starlink**: medición del consumo real por stream (vía estadísticas WebRTC) y horas de operación, con reporte por rango de fechas y exportación a CSV.

## Notas

- Pensado para operar en **red local cerrada** sobre la VPN del centro; no está endurecido para exposición directa a internet.
- El compresor del centro (GST-Grid) y su documento `screen_configuration` son **inmutables** desde el VMS: el sistema los lee y solo realiza una escritura puntual y segura (activar/detener una pantalla).
- Documentación interna y convenciones de desarrollo en [`CLAUDE.md`](CLAUDE.md).

---

*OMNIFISH VMS — Omnifish (desarrollador) · Cermaq (cliente).*
