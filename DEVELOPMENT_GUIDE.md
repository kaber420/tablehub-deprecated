# Guía de Desarrollo y Ejecución (Tablehub)

El ecosistema de Tablehub está compuesto por varias partes (Hub, Panel Web, Nube y Firmware). Esta guía te servirá como referencia rápida para saber cómo compilar y arrancar cada uno de los componentes de forma independiente.

---

## 1. Panel de Administración Local (Frontend - Svelte)

Esta es la interfaz gráfica que corre en el navegador para configurar el Hub. Está hecha con Svelte y Vite.

- **Ubicación:** `/hub/web`
- **Requisitos:** Node.js, pnpm (o npm).

### Comandos útiles:
```bash
cd hub/web

# Instalar dependencias (solo la primera vez)
pnpm install

# Modo Desarrollo (Hot-Reload)
pnpm run dev

# Compilar para Producción (Genera la carpeta /dist que el backend Go va a servir)
pnpm run build
```

---

## 2. El Hub Local (Backend - Go)

Este es el "cerebro" local del restaurante. Sirve los archivos del panel de administración (la carpeta `dist` que compilaste en el paso anterior), actúa como broker local MQTT/NATS embebido para recibir telemetría de los ESP32, y se conecta al servidor en la nube mediante WebSockets seguros.

- **Ubicación:** `/hub`
- **Requisitos:** Go (Golang).

### Banderas de la CLI (Opciones de Configuración):
Al arrancar el Hub se pueden pasar las siguientes banderas:
*   `--data-dir`: Directorio base para almacenar la base de datos SQLite y datos de persistencia de JetStream (por defecto `./data`).
*   `--db`: Ruta directa a la base de datos SQLite (si está vacía, se usará `{data-dir}/tablehub.db`).
*   `--port`: Puerto del servidor de administración local (por defecto `8080`).
*   `--nats-port`: Puerto para el servidor NATS local (por defecto `4222`).
*   `--mqtt-port`: Puerto para el broker MQTT local de los ESP32 (por defecto `1883`).
*   `--cloud`: Dirección del WebSocket de la Nube SaaS.
*   `--restaurant`: ID identificador del restaurante.
*   `-reset-admin`: Restablece la contraseña del administrador para forzar el flujo de configuración inicial en la web.

### Comandos útiles:
```bash
cd hub

# Instalar o descargar dependencias de Go
go mod tidy

# Ejecutar en modo desarrollo directamente
go run main.go

# Compilar el binario para producción
go build -o bin/hub main.go

# Ejecutar el binario compilado
./bin/hub --mqtt-port 1883 --nats-port 4222 --data-dir ./data
```

> [!NOTE]
> **Tamaño del binario y compilación:** Al embeber el NATS Server en el binario del Hub de Go, el peso total del binario ejecutable y el tiempo requerido para el `go build` en producción se incrementarán sustancialmente.

> **Modo Reset**: Si alguna vez olvidas la contraseña del panel web local, puedes ejecutar el Hub con la bandera especial:
> `./bin/hub -reset-admin=true`

---

## 3. Servidor en la Nube SaaS (Backend - Go)

Este es el servidor global que recibe conexiones WebSocket desde múltiples Hubs locales en diferentes restaurantes para sincronizarlos.

- **Ubicación:** `/cloud`
- **Requisitos:** Go (Golang).

### Comandos útiles:
```bash
cd cloud

# Ejecutar en modo desarrollo directamente
go run ./cmd/server serve

# Compilar el binario para producción
go build -o bin/server ./cmd/server

# Ejecutar el binario compilado
./bin/server serve
```

---

## 4. Dispositivos Físicos "TablePads" (Firmware - C++/PlatformIO)

Este es el código que corre dentro de los microcontroladores (ESP32) que van en las mesas del restaurante.

- **Ubicación:** `/firmware`
- **Requisitos:** PlatformIO (Instalado normalmente como extensión de VS Code o CLI) o Python para usar el entorno virtual pre-configurado (`venv`).

### Comandos útiles:
```bash
cd firmware

# Compilar el firmware sin subirlo a una placa
pio run

# Compilar y subir el firmware por USB al ESP32
pio run -t upload

# Subir firmware y abrir el monitor serial para ver los logs en tiempo real
pio run -t upload -t monitor
```

> **Nota**: Si no tienes la CLI de PlatformIO instalada globalmente de forma nativa, asegúrate de activar el entorno virtual interno (`source venv/bin/activate`) antes de correr los comandos de `pio`.

---

## Resumen del Flujo de Trabajo Normal

Si vas a desarrollar una característica completa que toque todo el Hub, normalmente abrirás terminales separadas y harás lo siguiente:

1. Terminal 1: `cd hub/web && pnpm run build` (O puedes usar `pnpm run dev` si estás probando solo diseño).
2. Terminal 2: `cd hub && go run main.go`
3. Abres el navegador en `http://localhost:8080`.
