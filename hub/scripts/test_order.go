package main

import (
	"encoding/json"
	"log"
	"github.com/nats-io/nats.go"
)

func main() {
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// Payload simulando lo que enviaría TastyIgniter
	payload := map[string]interface{}{
		"table_number": "mesauno", // IMPORTANTE: debe coincidir con una mesa activa en tu DB
		"order_id":     "ORD-9999",
		"customer_name": "Juan Prueba",
		"status_name":  "En Cocina",
		"items": []map[string]interface{}{
			{
				"id":           "item_1",
				"name":         "Hamburguesa Doble",
				"status_label": "En Cocina",
			},
			{
				"id":           "item_2",
				"name":         "Papas Fritas",
				"status_label": "Listo",
			},
		},
	}

	data, _ := json.Marshal(payload)
	err = nc.Publish("tenant.local.pos.order", data)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("✅ Orden de prueba publicada en NATS (tenant.local.pos.order)")
	log.Println(string(data))
}
