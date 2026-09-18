package state

import (
	"sync"
)

// activeOrders almacena la asociación Mesa (string) -> sync.Map (OrderID -> []byte)
var activeOrders sync.Map

// SaveOrderToCache guarda o actualiza un pedido en formato JSON para una mesa.
func SaveOrderToCache(tableNumber string, orderID string, orderJSON []byte) {
	if tableNumber == "" || orderID == "" {
		return
	}

	val, _ := activeOrders.LoadOrStore(tableNumber, &sync.Map{})
	tableMap := val.(*sync.Map)
	tableMap.Store(orderID, orderJSON)
}

// RemoveOrderFromCache elimina un pedido específico de la mesa.
func RemoveOrderFromCache(tableNumber string, orderID string) {
	if val, ok := activeOrders.Load(tableNumber); ok {
		tableMap := val.(*sync.Map)
		tableMap.Delete(orderID)
	}
}

// ClearOrdersForTable limpia todos los pedidos asociados a una mesa.
func ClearOrdersForTable(tableNumber string) {
	activeOrders.Delete(tableNumber)
}

// GetOrdersForTable obtiene todos los pedidos activos para una mesa.
func GetOrdersForTable(tableNumber string) [][]byte {
	var list [][]byte
	if val, ok := activeOrders.Load(tableNumber); ok {
		tableMap := val.(*sync.Map)
		tableMap.Range(func(key, value interface{}) bool {
			if orderBytes, ok := value.([]byte); ok {
				list = append(list, orderBytes)
			}
			return true
		})
	}
	return list
}
