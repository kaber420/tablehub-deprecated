package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vmihailenco/msgpack/v5"
)

type CatalogPayload struct {
	Action         string            `msgpack:"action"`
	CatalogVersion string            `msgpack:"catalog_version"`
	Categories     []CatalogCategory `msgpack:"categories"`
	Items          []CatalogItem     `msgpack:"items"`
}

type CatalogCategory struct {
	ID   int    `msgpack:"id"`
	Name string `msgpack:"name"`
	Icon string `msgpack:"icon"`
}

type CatalogItem struct {
	ID          string  `msgpack:"id"`
	Name        string  `msgpack:"name"`
	Description string  `msgpack:"description"`
	Price       float32 `msgpack:"price"`
	CategoryID  int     `msgpack:"category_id"`
	ImageHash   string  `msgpack:"image_hash"`
	Icon        string  `msgpack:"icon"`
}

func main() {
	// Connect to SQLite to get an active MAC address
	db, err := sql.Open("sqlite3", "../data/tablehub.db")
	if err != nil {
		log.Fatalf("Error abriendo SQLite: %v", err)
	}
	defer db.Close()

	var macAddress string
	err = db.QueryRow("SELECT mac_address FROM devices LIMIT 1").Scan(&macAddress)
	if err != nil {
		log.Printf("Advertencia: No se encontró dispositivo en DB (%v). Usando MAC detectada: 14:C1:9F:4D:7F:D8", err)
		macAddress = "14:C1:9F:4D:7F:D8"
	}

	opts := mqtt.NewClientOptions().AddBroker("tcp://127.0.0.1:1883")
	opts.SetClientID("test_catalog_publisher")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Error conectando a MQTT local: %v", token.Error())
	}
	defer client.Disconnect(250)

	payload := CatalogPayload{
		Action:         "sync_catalog",
		CatalogVersion: "rev_v4_clean",
		Categories: []CatalogCategory{
			{ID: 1, Name: "Comida", Icon: "LV_SYMBOL_IMAGE"},
			{ID: 2, Name: "Bebidas", Icon: "LV_SYMBOL_IMAGE"},
			{ID: 3, Name: "Postres", Icon: "LV_SYMBOL_IMAGE"},
		},
		Items: []CatalogItem{
			{ID: "item_1", Name: "Burger Clásica", Description: "Carne de res, queso, lechuga y tomate", Price: 120.0, CategoryID: 1, ImageHash: "img_burger", Icon: "LV_SYMBOL_IMAGE"},
			{ID: "item_2", Name: "Papas Fritas", Description: "Papas a la francesa crujientes", Price: 45.0, CategoryID: 1, ImageHash: "img_fries", Icon: "LV_SYMBOL_IMAGE"},
			{ID: "item_3", Name: "Refresco Cola", Description: "Lata 355ml", Price: 35.0, CategoryID: 2, ImageHash: "img_soda", Icon: "LV_SYMBOL_IMAGE"},
			{ID: "item_4", Name: "Cheesecake", Description: "Rebanada de pastel de queso con fresa", Price: 75.0, CategoryID: 3, ImageHash: "img_cake", Icon: "LV_SYMBOL_IMAGE"},
		},
	}

	// Serialización MsgPack binaria
	data, err := msgpack.Marshal(payload)
	if err != nil {
		log.Fatalf("Error serializando MsgPack: %v", err)
	}
	topic := fmt.Sprintf("tablehub/device/%s/sync", macAddress)

	fmt.Printf("Publicando nuevo catálogo MsgPack (%s) al tópico: %s (Retained = true)\n", payload.CatalogVersion, topic)
	
	// retained = true para persistir el mensaje en el broker
	token := client.Publish(topic, 0, true, data)
	token.Wait()
	
	if token.Error() != nil {
		log.Fatalf("Error al publicar: %v", token.Error())
	}
	
	time.Sleep(500 * time.Millisecond)
	fmt.Println("¡Catálogo MsgPack publicado exitosamente!")
}
