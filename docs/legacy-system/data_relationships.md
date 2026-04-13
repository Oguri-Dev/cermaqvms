# Relaciones de Datos del Sistema Anterior

## Jerarquía del Centro (Ejemplo: Pollollo)

```
Centro: Pollollo (192.200.53.1/32)
├── Módulo 1
│   ├── Línea 1: Jaula 101, Jaula 102
│   ├── Línea 2: Jaula 103, Jaula 104
│   ├── Línea 3: Jaula 105, Jaula 106
│   └── Línea 4: Jaula 107, Jaula 108
└── Módulo 2
    ├── Línea 1: Jaula 201, Jaula 202
    ├── Línea 2: Jaula 203, Jaula 204
    ├── Línea 3: Jaula 205, Jaula 206
    └── Línea 4: Jaula 207, Jaula 208
```

## Dispositivos

### NVR 1 (192.200.60.3)
- Usuario: admin
- 28 cámaras submarinas (canales 1-28)
- Cada jaula tiene 2 cámaras: A y B

### NVR 2 (192.200.60.4)
- Usuario: admin
- 4 cámaras PTZ (canales 9-12)
  - PTZ1 100 (192.200.60.51)
  - PTZ2 100 (192.200.60.52)
  - PTZ1 200 (192.200.60.53)
  - PTZ2 200 (192.200.60.54)

## Configuración de Pantallas

### Pantalla 1 (pantalla1) — Módulo 1
- Grilla: 4x4 (16 cámaras submarinas)
- Resolución: 1920x1080
- FPS: 20, GOP: 25, Bitrate: 3000
- selectFlow: 1 (NVR primario)
- hardwareEncoding: 4 (CPU dec + GPU enc)
- Layout de celdas (por columnas):
  ```
  J101 A  |  J103 A  |  J105 A  |  J107 A
  J101 B  |  J103 B  |  J105 B  |  J107 B
  J102 A  |  J104 A  |  J106 A  |  J108 A
  J102 B  |  J104 B  |  J106 B  |  J108 B
  ```

### Pantalla 2 (pantalla2) — Módulo 2
- Grilla: 4x4 (12 cámaras submarinas + 4 vacías)
- Resolución: 1920x1080
- FPS: 20, GOP: 25, Bitrate: 3000
- selectFlow: 1 (NVR primario)
- hardwareEncoding: 3 (GPU dec + CPU comp + GPU enc)
- Layout de celdas:
  ```
  J201 A  |  J203 A  |  J205 A  |  (vacía)
  J201 B  |  J203 B  |  J205 B  |  (vacía)
  J202 A  |  J204 A  |  J206 A  |  (vacía)
  J202 B  |  J204 B  |  J206 B  |  (vacía)
  ```

### Pantalla 3 (pantalla3) — PTZ/Domos
- Grilla: 2x2 (4 cámaras PTZ)
- Resolución: 1920x1080
- FPS: 20, GOP: 25, Bitrate: 3000
- selectFlow: 1 (NVR primario)
- hardwareEncoding: 4 (CPU dec + GPU enc)
- Layout:
  ```
  PTZ1 100  |  PTZ1 200
  PTZ2 100  |  PTZ2 200
  ```

## screen_view (Vista de pantallas)
Agrupa las 3 screen_configurations en una vista llamada "Default":
- Pantalla 1 (Módulo 1 submarinas)
- Pantalla 2 (Módulo 2 submarinas)
- Pantalla 3 (PTZ/Domos)

## Relaciones clave
1. **Centro** → Módulos → Líneas → Jaulas
2. **NVR** contiene cámaras (con credenciales compartidas)
3. **Cámara** pertenece a una jaula y a un NVR
4. **screen_configuration** = Stream (grilla + asignación de cámaras + compresión)
5. **screen_view** = Pantallas (agrupa múltiples screen_configurations)
