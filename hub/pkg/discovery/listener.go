package discovery

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
)

type DiscoverRequest struct {
	Cmd       string `json:"cmd"`
	DeviceMac string `json:"device_mac"`
}

type DiscoverResponse struct {
	Status   string `json:"status"`
	HubIP    string `json:"hub_ip"`
	MqttPort int    `json:"mqtt_port"`
}

type UDPListener struct {
	port     int
	mqttPort int
	conn     *net.UDPConn
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewUDPListener(port int, mqttPort int) *UDPListener {
	return &UDPListener{
		port:     port,
		mqttPort: mqttPort,
		stopChan: make(chan struct{}),
	}
}

func (l *UDPListener) Start() error {
	addr := net.UDPAddr{
		Port: l.port,
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp4", &addr)
	if err != nil {
		return fmt.Errorf("error escuchando UDP en puerto %d: %w", l.port, err)
	}
	l.conn = conn

	log.Printf("[UDP Discovery] Escuchando solicitudes de descubrimiento ESP32 en puerto %d...", l.port)

	l.wg.Add(1)
	go l.listenLoop()

	return nil
}

func (l *UDPListener) listenLoop() {
	defer l.wg.Done()
	buf := make([]byte, 1024)

	for {
		select {
		case <-l.stopChan:
			return
		default:
		}

		n, remoteAddr, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-l.stopChan:
				return
			default:
				log.Printf("[UDP Discovery] Error leyendo paquete UDP: %v", err)
				continue
			}
		}

		var req DiscoverRequest
		if err := json.Unmarshal(buf[:n], &req); err != nil {
			continue
		}

		if req.Cmd == "DISCOVER_HUB" {
			log.Printf("[UDP Discovery] Petición recibida de %s (MAC: %s)", remoteAddr.String(), req.DeviceMac)

			localIP := getLocalIPForRemote(remoteAddr.IP)
			resp := DiscoverResponse{
				Status:   "IAM_HUB",
				HubIP:    localIP,
				MqttPort: l.mqttPort,
			}

			data, err := json.Marshal(resp)
			if err != nil {
				log.Printf("[UDP Discovery] Error serializando respuesta: %v", err)
				continue
			}

			_, err = l.conn.WriteToUDP(data, remoteAddr)
			if err != nil {
				log.Printf("[UDP Discovery] Error enviando respuesta a %s: %v", remoteAddr.String(), err)
			} else {
				log.Printf("[UDP Discovery] Respuesta enviada a %s con IP Hub: %s", remoteAddr.String(), localIP)
			}
		}
	}
}

func (l *UDPListener) Stop() {
	close(l.stopChan)
	if l.conn != nil {
		l.conn.Close()
	}
	l.wg.Wait()
	log.Println("[UDP Discovery] Listener detenido.")
}

func getLocalIPForRemote(remoteIP net.IP) string {
	conn, err := net.Dial("udp", remoteIP.String()+":80")
	if err != nil {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return "127.0.0.1"
		}
		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
