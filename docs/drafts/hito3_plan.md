# Hito 3: Infraestructura de Mensajería (NATS JetStream + MQTT)

Este plan detalla los pasos de código exactos para integrar NATS embebido en el Hub (`hub/`) para que funcione como el broker local de MQTT para los ESP32, y cómo el backend de Go consumirá estos eventos usando JetStream.

## Resumen del Objetivo
Reemplazar la dependencia de un broker MQTT externo por un servidor NATS embebido en el binario del Hub de Go. El Hub habilitará el puerto `1883` para recibir conexiones MQTT de los ESP32, y usará su propio cliente nativo de NATS para consumir esos eventos de forma persistente a través de JetStream.

## Decisiones de Diseño Tomadas
- **Versiones de Dependencias (Estables y Actuales):**
  - NATS Server: `github.com/nats-io/nats-server/v2` en versión **`v2.10.24`**.
  - NATS Go Client: `github.com/nats-io/nats.go` en versión **`v1.52.0`**.
- **Almacenamiento:** Los eventos de JetStream se guardarán en disco (`FileStore`) en `{data-dir}/jetstream` para garantizar persistencia absoluta ante apagones.
- **Transición de MQTT a NATS en Go:** Se eliminará `internal/mqtt/client.go` y se creará `internal/events/`. El backend de Go usará puramente NATS JetStream para consumir y NATS para publicar, dejando el protocolo MQTT exclusivo para la conexión física de los ESP32.
- **QoS 1 Requerido:** JetStream requiere obligatoriamente **QoS 1** para persistir mensajes MQTT en el Stream. El firmware de los ESP32 deberá configurarse para publicar eventos con QoS 1.
- **Abstracción de Conexión:** El paquete `web` no debe importar la librería de NATS directamente. La comunicación se canalizará a través del paquete `events` mediante helpers expuestos.
- **Impacto en Binario:** Embeber `nats-server` incrementa el peso del binario final y el tiempo de compilación. Se documentará este aspecto en la guía de desarrollo.

---

## Pasos Propuestos

### 1. Dependencias y Limpieza de go.mod
- Ejecutar `go get github.com/nats-io/nats-server/v2@v2.10.24`
- Ejecutar `go get github.com/nats-io/nats.go@v1.52.0`
- Eliminar el uso de `paho.mqtt.golang` y ejecutar `go mod tidy` para limpiar `go.mod` y `go.sum` de dependencias no utilizadas.

### 2. Nuevo Paquete de Eventos (`internal/events`)
Crearemos un módulo centralizado para manejar el servidor embebido y la lógica de JetStream.

#### Nuevo Archivo: `hub/internal/events/server.go`
- Configurará un `server.Options` de NATS.
  - *Nota de correctitud:* Usar la estructura moderna de `MQTTOpts` dentro de `server.Options` (compatible con la versión `v2.10.24`):
    ```go
    opts := &server.Options{
        // ... otras opciones ...
        JetStream: true,
        StoreDir: filepath.Join(dataDir, "jetstream"),
        MQTT: server.MQTTOpts{
            Host: "0.0.0.0",
            Port: mqttPort,
        },
    }
    ```
- Redirigirá el logger interno de NATS para que escriba a través del logger estándar de Go (`log.Printf`) y evitar logs duplicados o desordenados.
- Expondrá una función `StartEmbeddedServer(dataDir string, mqttPort int, natsPort int)` que arrancará el servidor en una goroutine y bloqueará hasta que esté listo usando `server.ReadyForConnections()`.
- Expondrá una función `ShutdownEmbeddedServer()` para realizar un apagado seguro (`natsServer.Shutdown()`).

#### Nuevo Archivo: `hub/internal/events/jetstream.go`
- Creará una conexión cliente nativa de NATS (`nats.Connect(...)`).
  - *Nota de robustez:* Implementar reintentos con timeout en la conexión inicial por si el listener de NATS embebido tarda en levantar.
- Creará el contexto de JetStream gestionando errores de inicialización (`js, err := nc.JetStream()`).
- Definirá un Stream persistente llamado `TABLE_EVENTS` configurado con límites específicos:
  - `Subjects`: `tablehub.table.>` (wildcard multinivel para soportar futuras jerarquías).
  - `Storage`: `file.Storage` (FileStore).
  - `MaxAge`: `72h` (los datos se persistirán en base de datos local mediante el consumidor).
  - `MaxBytes`: `1GB`.
  - `Replicas`: `1` (nodo embebido local).
- Implementará suscripciones de tipo **Push** con **Ack manual** (`ManualAck()`) y un consumidor durable (`hub-worker`).
- En el callback de consumo:
  - Validar y procesar el mensaje.
  - Asegurar llamar **siempre** a `msg.Ack()` en caso de éxito o de error no-reintentable para no bloquear la cola de procesamiento.
  - Utilizar `msg.NakWithDelay(...)` o similar con límites de reintento ante fallos temporales (ej. problemas de escritura en base de datos).
- Expondrá un helper de publicación global para ser consumido por otros paquetes (como `web`):
  ```go
  func Publish(subject string, data []byte) error {
      // Publica usando el cliente nativo de NATS previamente inicializado
  }
  ```

### 3. Refactorización del Main, Web Server y Documentación

#### Modificar: `hub/main.go`
- Eliminar el flag obsoleto `-mqtt`.
- Agregar nuevos flags a la CLI:
  - `--nats-port` (por defecto `4222`)
  - `--mqtt-port` (por defecto `1883`)
  - `--data-dir` (por defecto `./data`)
- Mantener el flag existente `-reset-admin`.
- Importar y arrancar el servidor embebido de NATS en una goroutine llamando a `events.StartEmbeddedServer()`.
- Asegurar la sincronización de readiness antes de conectar el cliente JetStream.
- En el manejador de señales del sistema (`SIGINT`/`SIGTERM`), implementar el **orden de apagado seguro**:
  1. Drenar consumidor JetStream (`sub.Drain()`).
  2. Cerrar la conexión cliente NATS (`nc.Close()`).
  3. Apagar el servidor NATS (`events.ShutdownEmbeddedServer()`).
  4. Cerrar la conexión a la base de datos SQLite (`db.Close()`).

#### Modificar: `hub/internal/web/server.go`
- En el endpoint/goroutine que gestiona la sesión en la nube (`handleCloudSession`), sustituir el placeholder de publicación MQTT por una llamada al helper de publicación del paquete `events`:
  ```go
  events.Publish("tablehub.table.5.order", data) // NATS mapeará esto a MQTT de forma transparente para el ESP32 suscrito a "tablehub/table/5/order"
  ```

#### Eliminar: `hub/internal/mqtt/client.go`
- Eliminar este archivo completamente.

#### Modificar: `DEVELOPMENT_GUIDE.md`
- Actualizar la guía de desarrollo eliminando la referencia al flag `-mqtt` y documentando los nuevos flags (`--nats-port`, `--mqtt-port`, `--data-dir`).
- Añadir sección sobre el aumento del tamaño del binario y tiempos de compilación al integrar NATS server en producción.

---

## Plan de Verificación

1. **Compilación e Inicio:**
   Arrancar el hub usando `go run main.go --mqtt-port 1883 --data-dir ./data`. Verificar que NATS se inicializa correctamente y el puerto 1883 queda abierto.
2. **Prueba de Ingesta (QoS 1 requerido):**
   Desde una terminal externa, publicar un mensaje simulando un ESP32 utilizando QoS 1 (`-q 1`):
   ```bash
   mosquitto_pub -h localhost -p 1883 -q 1 -t "tablehub/table/5/alert" -m '{"table":"5","kind":"waiter","timestamp":12345}'
   ```
   Verificar en los logs del Hub de Go que JetStream interceptó el evento y lo procesó.
3. **Prueba de Publicación (Salida):**
   Validar que la publicación desde el backend de Go hacia el subject `tablehub.table.5.order` es recibida por un cliente suscrito vía MQTT al tópico `tablehub/table/5/order`:
   ```bash
   mosquitto_sub -h localhost -p 1883 -t "tablehub/table/5/order"
   ```
4. **Prueba de Graceful Shutdown:**
   Enviar una señal `SIGINT` (Ctrl+C) al Hub y verificar en los logs que se ejecuta la secuencia ordenada de cierre: consumidor -> cliente NATS -> servidor NATS -> base de datos.
