package ws

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/tablehub/cloud/internal/application/tunnel"
)

func createTestHubConn(t *testing.T, hubID string, config WSConfig) *HubConn {
	t.Helper()
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		hubConn := NewHubConn(conn, hubID, "", config, nil)
		_ = hubConn
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}

	return NewHubConn(wsConn, hubID, "", config, nil)
}

func TestHubRegistry_StoreAndGet(t *testing.T) {
	registry := NewHubRegistry()

	hubConn := createTestHubConn(t, "hub-001", DefaultWSConfig())

	registry.Store("hub-001", "", hubConn)
	defer registry.Delete("hub-001")

	retrieved := registry.Get("hub-001")
	if retrieved == nil {
		t.Fatal("expected to retrieve connection, got nil")
	}

	// Verify the sender is the same HubConn we stored.
	if retrieved != tunnel.Sender(hubConn) {
		t.Error("expected to get back the same sender")
	}
}

func TestHubRegistry_Delete(t *testing.T) {
	registry := NewHubRegistry()

	hubConn := createTestHubConn(t, "hub-002", DefaultWSConfig())

	registry.Store("hub-002", "", hubConn)
	registry.Delete("hub-002")

	retrieved := registry.Get("hub-002")
	if retrieved != nil {
		t.Error("expected nil after delete, got connection")
	}
}

func TestHubRegistry_ReplaceStaleConnection(t *testing.T) {
	registry := NewHubRegistry()

	hubConn1 := createTestHubConn(t, "hub-003", DefaultWSConfig())
	hubConn2 := createTestHubConn(t, "hub-003", DefaultWSConfig())

	registry.Store("hub-003", "", hubConn1)
	registry.Store("hub-003", "", hubConn2)
	defer registry.Delete("hub-003")

	retrieved := registry.Get("hub-003")
	if retrieved == nil {
		t.Fatal("expected to retrieve connection, got nil")
	}

	if retrieved != tunnel.Sender(hubConn2) {
		t.Error("expected second connection to replace first")
	}
}

func TestHubRegistry_Shutdown(t *testing.T) {
	registry := NewHubRegistry()

	for i := 0; i < 10; i++ {
		conn := createTestHubConn(t, fmt.Sprintf("hub-%d", i), DefaultWSConfig())
		registry.Store(fmt.Sprintf("hub-%d", i), "", conn)
	}

	hubIDs := registry.Shutdown()

	if len(hubIDs) != 10 {
		t.Errorf("expected 10 hub IDs from shutdown, got %d", len(hubIDs))
	}

	for i := 0; i < 10; i++ {
		retrieved := registry.Get(fmt.Sprintf("hub-%d", i))
		if retrieved != nil {
			t.Errorf("expected nil after shutdown for hub-%d, got connection", i)
		}
	}
}

func TestHubRegistry_ConcurrentStoreGetDelete(t *testing.T) {
	registry := NewHubRegistry()

	const numGoroutines = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			hubID := fmt.Sprintf("concurrent-hub-%d", id)
			hubConn := createTestHubConn(t, hubID, DefaultWSConfig())
			registry.Store(hubID, "", hubConn)
		}(i)

		go func(id int) {
			defer wg.Done()
			hubID := fmt.Sprintf("concurrent-hub-%d", id)
			registry.Get(hubID)
		}(i)

		go func(id int) {
			defer wg.Done()
			hubID := fmt.Sprintf("concurrent-hub-%d", id)
			registry.Delete(hubID)
		}(i)
	}

	wg.Wait()
}

func TestHubRegistry_ConcurrentReadWrite(t *testing.T) {
	registry := NewHubRegistry()

	const numWriters = 50
	const numReaders = 50
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numWriters + numReaders)

	for i := 0; i < numWriters; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				hubID := fmt.Sprintf("rw-hub-%d-%d", id, j)
				hubConn := createTestHubConn(t, hubID, DefaultWSConfig())
				registry.Store(hubID, "", hubConn)
			}
		}(i)
	}

	for i := 0; i < numReaders; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				hubID := fmt.Sprintf("rw-hub-%d-%d", id, j)
				registry.Get(hubID)
			}
		}(i)
	}

	wg.Wait()
}

func TestHubRegistry_ConcurrentReplace(t *testing.T) {
	registry := NewHubRegistry()

	const numGoroutines = 100
	const hubID = "shared-hub"

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			hubConn := createTestHubConn(t, hubID, DefaultWSConfig())
			registry.Store(hubID, "", hubConn)
		}(i)
	}

	wg.Wait()

	retrieved := registry.Get(hubID)
	if retrieved == nil {
		t.Fatal("expected to retrieve connection after concurrent replace")
	}
}

func TestHubRegistry_ConcurrentShutdown(t *testing.T) {
	registry := NewHubRegistry()

	const numConnections = 100

	for i := 0; i < numConnections; i++ {
		hubConn := createTestHubConn(t, fmt.Sprintf("shutdown-hub-%d", i), DefaultWSConfig())
		registry.Store(fmt.Sprintf("shutdown-hub-%d", i), "", hubConn)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		registry.Shutdown()
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < numConnections; i++ {
			hubID := fmt.Sprintf("shutdown-hub-%d", i)
			registry.Get(hubID)
		}
	}()

	wg.Wait()
}
