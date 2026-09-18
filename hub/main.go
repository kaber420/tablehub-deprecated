package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/tablehub/hub/internal/db"
	"github.com/tablehub/hub/internal/events"
	"github.com/tablehub/hub/internal/web"
	"github.com/tablehub/hub/pkg/discovery"
)

func main() {
	// 0. Cargar el archivo .env si existe para inicializar variables de entorno
	loadEnv(".env")

	// Leer configuraciones iniciales desde variables de entorno para los valores por defecto
	defaultDataDir := getEnvStr("DATA_DIR", "./data")
	defaultDbPath := getEnvStr("DB_PATH", "")
	defaultPort := getEnvStr("PORT", "8080")
	defaultNatsPort := getEnvInt("NATS_PORT", 4222)
	defaultMqttPort := getEnvInt("MQTT_PORT", 1883)
	defaultCloudURL := getEnvStr("CLOUD_URL", "")
	defaultRestaurantID := getEnvStr("RESTAURANT_ID", "test-restaurant-001")

	// 1. Parsear banderas de configuración (las banderas CLI tienen prioridad sobre .env)
	dataDir := flag.String("data-dir", defaultDataDir, "Directorio base para almacenar datos (SQLite, JetStream)")
	dbPath := flag.String("db", defaultDbPath, "Ruta al archivo de base de datos SQLite (si está vacío se usará {data-dir}/tablehub.db)")
	webPort := flag.String("port", defaultPort, "Puerto para el servidor web de administración local")
	natsPort := flag.Int("nats-port", defaultNatsPort, "Puerto para el servidor NATS local")
	mqttPort := flag.Int("mqtt-port", defaultMqttPort, "Puerto para el broker MQTT local")
	cloudURL := flag.String("cloud", defaultCloudURL, "URL del WebSocket del servidor SaaS en la nube (dejar vacío para deshabilitar)")
	restaurantID := flag.String("restaurant", defaultRestaurantID, "ID de este restaurante para autenticación")
	resetAdmin := flag.Bool("reset-admin", false, "Borrar la contraseña de administrador para forzar el modo Setup inicial")
	flag.Parse()

	log.Println("=== Iniciando Maitre Hub ===")

	// Resolver ruta de la base de datos
	resolvedDbPath := *dbPath
	if resolvedDbPath == "" {
		resolvedDbPath = filepath.Join(*dataDir, "tablehub.db")
	}

	// 1. Inicializar la Base de Datos SQLite
	absDbPath, err := filepath.Abs(resolvedDbPath)
	if err != nil {
		log.Fatalf("Error al resolver ruta de la base de datos: %v", err)
	}
	log.Printf("Inicializando base de datos local en: %s", absDbPath)
	if err = db.InitDB(absDbPath); err != nil {
		log.Fatalf("Fallo crítico al inicializar base de datos: %v", err)
	}

	// 1.1 Crear directorios necesarios para el CDN
	if err = os.MkdirAll(filepath.Join(*dataDir, "cdn", "assets"), 0755); err != nil {
		log.Fatalf("Error al crear directorios del CDN: %v", err)
	}

	// 1.5. Manejar restablecimiento de administrador si la bandera está activa
	if *resetAdmin {
		log.Println("--- MODO RESET ADMINISTRADOR ---")
		err := db.SaveSetting("admin_password_hash", "")
		if err != nil {
			log.Fatalf("Error al borrar contraseña de administrador: %v", err)
		}
		log.Println("Contraseña borrada con éxito.")
		log.Println("Por favor, reinicia el servidor sin la bandera -reset-admin y entra a la web para hacer el Setup inicial.")
		os.Exit(0)
	}

	// 1.6. Inicializar módulo de autenticación
	web.InitAuth()

	// Configurar callback para redirección de eventos sin ciclo de importaciones
	events.BroadcastCallback = web.BroadcastEvent

	// 2. Iniciar el Hub de WebSockets locales para el panel
	go web.LocalHub.Run()

	// 3. Iniciar NATS Server embebido (MQTT Bridge + JetStream)
	log.Println("Iniciando NATS Server embebido con JetStream y MQTT...")
	if err := events.StartEmbeddedServer(*dataDir, *mqttPort, *natsPort); err != nil {
		log.Fatalf("Fallo crítico al iniciar NATS embebido: %v", err)
	}

	// 3.5. Inicializar conexión cliente a NATS JetStream y consumidor durable
	log.Println("Conectando cliente local a NATS JetStream...")
	if err := events.InitJetStream(*natsPort); err != nil {
		events.ShutdownEmbeddedServer()
		log.Fatalf("Fallo crítico al inicializar JetStream: %v", err)
	}

	if err := events.StartConsumer(); err != nil {
		events.CloseClient()
		events.ShutdownEmbeddedServer()
		log.Fatalf("Fallo crítico al arrancar consumidor JetStream: %v", err)
	}

	// 4. Levantar el Servidor Web Local (HTTP + WS)
	go web.StartWebServer(*webPort)

	// 5. Iniciar la conexión persistente con la nube en segundo plano (Túnel)
	if *cloudURL != "" {
		db.SaveSetting("cloud_url", *cloudURL)
		db.SaveSetting("cloud_enabled", "true")
	}
	if *restaurantID != "" {
		db.SaveSetting("restaurant_id", *restaurantID)
	}

	if hubID := getEnvStr("HUB_ID", ""); hubID != "" {
		db.SaveSetting("hub_id", hubID)
	}
	
	if privKey := getEnvStr("HUB_PRIVATE_KEY", ""); privKey != "" {
		db.SaveSetting("private_key", privKey)
	}
	
	// Generar llaves al inicio si no existen (así el frontend puede mostrar la llave pública inmediatamente)
	web.InitCloudKeys()

	// 5.5. Iniciar servicio de descubrimiento UDP para dispositivos ESP32
	udpDiscovery := discovery.NewUDPListener(9999, *mqttPort)
	if err := udpDiscovery.Start(); err != nil {
		log.Printf("Advertencia: No se pudo iniciar el servicio de descubrimiento UDP: %v", err)
	} else {
		defer udpDiscovery.Stop()
	}

	web.RestartCloudConnection()

	// 6. Escuchar señales de terminación para apagado limpio
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Apagando Maitre Hub de manera limpia...")
	
	// Cierre en orden seguro: consumidor -> cliente -> servidor -> base de datos
	events.CloseClient()
	events.ShutdownEmbeddedServer()

	if db.DB != nil {
		db.DB.Close()
		log.Println("Base de datos SQLite cerrada.")
	}
	log.Println("=== Hub Apagado ===")
}

// loadEnv carga pares CLAVE=VALOR desde un archivo de texto en variables de entorno.
func loadEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return // Ignorar silenciosamente si no existe
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			os.Setenv(key, val)
		}
	}
}

// getEnvStr obtiene una variable de entorno de tipo string o retorna un valor por defecto.
func getEnvStr(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvInt obtiene una variable de entorno de tipo entero o retorna un valor por defecto.
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
