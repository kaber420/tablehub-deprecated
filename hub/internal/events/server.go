package events

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

var (
	natsServer *server.Server
	mqttPort   int
	natsPort   int
)

type natsLogger struct{}

func (l *natsLogger) Noticef(format string, v ...any) { log.Printf("[NATS] "+format, v...) }
func (l *natsLogger) Warnf(format string, v ...any)   { log.Printf("[NATS WARN] "+format, v...) }
func (l *natsLogger) Fatalf(format string, v ...any)  { log.Fatalf("[NATS FATAL] "+format, v...) }
func (l *natsLogger) Errorf(format string, v ...any)  { log.Printf("[NATS ERROR] "+format, v...) }
func (l *natsLogger) Debugf(format string, v ...any)  { log.Printf("[NATS DEBUG] "+format, v...) }
func (l *natsLogger) Tracef(format string, v ...any)  { log.Printf("[NATS TRACE] "+format, v...) }

// StartEmbeddedServer inicia un servidor NATS con JetStream y MQTT embebidos.
func StartEmbeddedServer(dataDir string, mPort int, nPort int) error {
	mqttPort = mPort
	natsPort = nPort

	opts := &server.Options{
		ServerName: "tablehub-nats",
		Host:       "0.0.0.0",
		Port:       nPort,
		JetStream:  true,
		StoreDir:   filepath.Join(dataDir, "jetstream"),
		MQTT: server.MQTTOpts{
			Host: "0.0.0.0",
			Port: mPort,
		},
		NoLog: true, // Usamos nuestro propio logger canalizado a Go logs
	}

	var err error
	natsServer, err = server.NewServer(opts)
	if err != nil {
		return fmt.Errorf("error al inicializar NATS server: %w", err)
	}

	// Establecer el custom logger para canalizar a través de log.Printf
	natsServer.SetLogger(&natsLogger{}, false, false)

	// Arrancar el servidor en una goroutine
	go natsServer.Start()

	// Bloquear hasta que esté listo para conexiones (máximo 10 segundos)
	if !natsServer.ReadyForConnections(10 * time.Second) {
		return fmt.Errorf("el servidor NATS embebido no se inició a tiempo")
	}

	log.Printf("NATS Server embebido iniciado. Puerto NATS: %d, Puerto MQTT Bridge: %d", natsPort, mqttPort)
	return nil
}

// ShutdownEmbeddedServer detiene el servidor de NATS limpiamente.
func ShutdownEmbeddedServer() {
	if natsServer != nil {
		log.Println("Apagando NATS Server embebido...")
		defer func() {
			if r := recover(); r != nil {
				log.Printf("NATS Server embebido finalizado (recuperado de advertencia durante apagado: %v)", r)
			}
		}()
		natsServer.Shutdown()
		natsServer.WaitForShutdown()
		log.Println("NATS Server embebido apagado.")
	}
}
