package ws

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/tablehub/cloud/internal/application/tunnel"
)

var errUnknownHub = errors.New("unknown hub")

type MockKeyProvider struct {
	keys map[string]ed25519.PublicKey
}

func NewMockKeyProvider() *MockKeyProvider {
	return &MockKeyProvider{
		keys: make(map[string]ed25519.PublicKey),
	}
}

func (m *MockKeyProvider) GetHubInfo(hubID string) ([]byte, string, error) {
	key, ok := m.keys[hubID]
	if !ok {
		return nil, "", errUnknownHub
	}
	return []byte(key), "test-org-123", nil
}

func (m *MockKeyProvider) AddKey(hubID string, key ed25519.PublicKey) {
	m.keys[hubID] = key
}

func dialWS(t *testing.T, serverURL string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	return conn
}

func readChallenge(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	_, rawMsg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read challenge: %v", err)
	}

	var challengeMsg tunnel.TunnelMessage
	if err := json.Unmarshal(rawMsg, &challengeMsg); err != nil {
		t.Fatalf("failed to unmarshal challenge: %v", err)
	}

	if challengeMsg.Event != eventAuthChallenge {
		t.Fatalf("expected event %s, got %s", eventAuthChallenge, challengeMsg.Event)
	}

	payloadBytes, _ := json.Marshal(challengeMsg.Payload)
	var challengePayload ChallengePayload
	if err := json.Unmarshal(payloadBytes, &challengePayload); err != nil {
		t.Fatalf("failed to unmarshal challenge payload: %v", err)
	}

	return challengePayload.ChallengeHex
}

func signChallenge(t *testing.T, privKey ed25519.PrivateKey, challengeHex string) string {
	t.Helper()
	challengeBytes, _ := hex.DecodeString(challengeHex)
	signature := ed25519.Sign(privKey, challengeBytes)
	return hex.EncodeToString(signature)
}

func sendAuthResponse(t *testing.T, conn *websocket.Conn, hubID, organizationID, signatureHex string) {
	t.Helper()
	authResponse := tunnel.TunnelMessage{
		Event:     eventAuthResponse,
		Timestamp: time.Now().UnixMilli(),
		Payload: AuthResponsePayload{
			HubID:          hubID,
			OrganizationID: organizationID,
			SignatureHex:   signatureHex,
		},
	}
	if err := conn.WriteJSON(authResponse); err != nil {
		t.Fatalf("failed to send auth response: %v", err)
	}
}

func TestGenerateChallenge(t *testing.T) {
	challenge, err := GenerateChallenge()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(challenge) != 64 {
		t.Errorf("expected challenge length 64, got %d", len(challenge))
	}

	if _, err := hex.DecodeString(challenge); err != nil {
		t.Errorf("challenge is not valid hex: %v", err)
	}

	challenge2, _ := GenerateChallenge()
	if challenge == challenge2 {
		t.Error("two consecutive challenges should not be identical")
	}
}

func TestAuthenticateHub_ValidSignature(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	hubID := "test-hub-001"

	provider := NewMockKeyProvider()
	provider.AddKey(hubID, pubKey)

	authDone := make(chan error, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			authDone <- err
			return
		}
		defer conn.Close()

		result, err := AuthenticateHub(conn, provider)
		if err != nil {
			authDone <- err
			return
		}
		if result.HubID != hubID {
			authDone <- errors.New("wrong hub_id: " + result.HubID)
			return
		}
		authDone <- nil
	}))
	defer server.Close()

	wsConn := dialWS(t, server.URL)
	defer wsConn.Close()

	challengeHex := readChallenge(t, wsConn)
	sigHex := signChallenge(t, privKey, challengeHex)
	sendAuthResponse(t, wsConn, hubID, "organization-001", sigHex)

	select {
	case err := <-authDone:
		if err != nil {
			t.Fatalf("authentication failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out waiting for auth result")
	}
}

func TestAuthenticateHub_InvalidSignature(t *testing.T) {
	pubKey, _, _ := ed25519.GenerateKey(rand.Reader)
	hubID := "test-hub-002"

	provider := NewMockKeyProvider()
	provider.AddKey(hubID, pubKey)

	authDone := make(chan error, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			authDone <- err
			return
		}
		defer conn.Close()

		_, err = AuthenticateHub(conn, provider)
		if err == nil {
			authDone <- errors.New("expected authentication to fail with invalid signature")
		} else {
			authDone <- nil
		}
	}))
	defer server.Close()

	wsConn := dialWS(t, server.URL)
	defer wsConn.Close()

	challengeHex := readChallenge(t, wsConn)

	_, wrongPrivKey, _ := ed25519.GenerateKey(rand.Reader)
	wrongSigHex := signChallenge(t, wrongPrivKey, challengeHex)
	sendAuthResponse(t, wsConn, hubID, "organization-001", wrongSigHex)

	select {
	case err := <-authDone:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out")
	}
}

func TestAuthenticateHub_UnknownHub(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)

	provider := NewMockKeyProvider()
	provider.AddKey("other-hub", pubKey)

	authDone := make(chan error, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			authDone <- err
			return
		}
		defer conn.Close()

		_, err = AuthenticateHub(conn, provider)
		if err == nil {
			authDone <- errors.New("expected authentication to fail for unknown hub")
		} else {
			authDone <- nil
		}
	}))
	defer server.Close()

	wsConn := dialWS(t, server.URL)
	defer wsConn.Close()

	challengeHex := readChallenge(t, wsConn)
	sigHex := signChallenge(t, privKey, challengeHex)
	sendAuthResponse(t, wsConn, "unknown-hub", "organization-001", sigHex)

	select {
	case err := <-authDone:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out")
	}
}

func TestAuthenticateHub_TruncatedSignature(t *testing.T) {
	pubKey, _, _ := ed25519.GenerateKey(rand.Reader)
	hubID := "test-hub-003"

	provider := NewMockKeyProvider()
	provider.AddKey(hubID, pubKey)

	authDone := make(chan error, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			authDone <- err
			return
		}
		defer conn.Close()

		_, err = AuthenticateHub(conn, provider)
		if err == nil {
			authDone <- errors.New("expected authentication to fail with truncated signature")
		} else {
			authDone <- nil
		}
	}))
	defer server.Close()

	wsConn := dialWS(t, server.URL)
	defer wsConn.Close()

	_ = readChallenge(t, wsConn)
	sendAuthResponse(t, wsConn, hubID, "organization-001", "abcdef1234")

	select {
	case err := <-authDone:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out")
	}
}
