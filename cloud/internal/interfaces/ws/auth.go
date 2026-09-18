package ws

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"github.com/tablehub/cloud/internal/application/tunnel"
)

const (
	challengeSize      = 32
	authTimeout        = 30 * time.Second
	eventAuthChallenge = "auth_challenge"
	eventAuthResponse  = "auth_response"
)

type AuthResult struct {
	HubID          string
	OrganizationID string
}

func closeWithReason(conn *websocket.Conn, code int, reason string) {
	closeMsg := websocket.FormatCloseMessage(code, reason)
	conn.WriteMessage(websocket.CloseMessage, closeMsg)
	conn.Close()
}

func GenerateChallenge() (string, error) {
	b := make([]byte, challengeSize)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate challenge: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func AuthenticateHub(conn *websocket.Conn, provider KeyProvider) (*AuthResult, error) {
	challengeHex, err := GenerateChallenge()
	if err != nil {
		return nil, err
	}

	challengeMsg := tunnel.TunnelMessage{
		Event:     eventAuthChallenge,
		Timestamp: time.Now().UnixMilli(),
		Payload:   ChallengePayload{ChallengeHex: challengeHex},
	}

	if err := conn.SetWriteDeadline(time.Now().Add(authTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	if err := conn.WriteJSON(challengeMsg); err != nil {
		return nil, fmt.Errorf("failed to send challenge: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(authTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	_, rawMsg, err := conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read auth response: %w", err)
	}

	var authMsg tunnel.TunnelMessage
	if err := json.Unmarshal(rawMsg, &authMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal auth message: %w", err)
	}

	if authMsg.Event != eventAuthResponse {
		closeWithReason(conn, websocket.CloseNormalClosure, "expected auth_response event")
		return nil, fmt.Errorf("unexpected event: %s", authMsg.Event)
	}

	payloadBytes, err := json.Marshal(authMsg.Payload)
	if err != nil {
		closeWithReason(conn, websocket.CloseNormalClosure, "invalid payload")
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	var authPayload AuthResponsePayload
	if err := json.Unmarshal(payloadBytes, &authPayload); err != nil {
		closeWithReason(conn, websocket.CloseNormalClosure, "invalid auth payload")
		return nil, fmt.Errorf("failed to unmarshal auth payload: %w", err)
	}

	if authPayload.HubID == "" {
		log.Warn().Msg("hub authentication failed: missing hub_id")
		closeWithReason(conn, 4001, "missing hub_id")
		return nil, fmt.Errorf("missing hub_id")
	}

	if authPayload.SignatureHex == "" {
		closeWithReason(conn, 4001, "missing signature_hex")
		return nil, fmt.Errorf("missing signature_hex")
	}

	pubKeyBytes, dbOrgID, err := provider.GetHubInfo(authPayload.HubID)
	if err != nil {
		closeWithReason(conn, websocket.CloseNormalClosure, "unknown hub")
		return nil, fmt.Errorf("failed to get hub info for hub %s: %w", authPayload.HubID, err)
	}

	sigBytes, err := hex.DecodeString(authPayload.SignatureHex)
	if err != nil {
		closeWithReason(conn, websocket.CloseNormalClosure, "invalid signature encoding")
		return nil, fmt.Errorf("invalid signature hex: %w", err)
	}

	msgToVerify, err := hex.DecodeString(challengeHex)
	if err != nil {
		closeWithReason(conn, websocket.CloseNormalClosure, "invalid challenge encoding")
		return nil, fmt.Errorf("invalid challenge hex: %w", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(pubKeyBytes), msgToVerify, sigBytes) {
		closeWithReason(conn, websocket.CloseNormalClosure, "invalid signature")
		return nil, fmt.Errorf("signature verification failed for hub %s", authPayload.HubID)
	}

	log.Info().
		Str("hub_id", authPayload.HubID).
		Str("organization_id", dbOrgID).
		Msg("hub authenticated successfully")

	successMsg := tunnel.TunnelMessage{
		Event:     "auth_success",
		Timestamp: time.Now().UnixMilli(),
	}
	if err := conn.WriteJSON(successMsg); err != nil {
		return nil, fmt.Errorf("failed to send auth success: %w", err)
	}

	return &AuthResult{
		HubID:          authPayload.HubID,
		OrganizationID: dbOrgID,
	}, nil
}
