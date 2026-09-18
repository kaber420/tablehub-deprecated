package ws

// ChallengePayload is the WebSocket auth challenge sent to a hub.
type ChallengePayload struct {
	ChallengeHex string `json:"challenge_hex"`
}

// AuthResponsePayload is the hub's response to the auth challenge.
type AuthResponsePayload struct {
	OrganizationID string `json:"organization_id"`
	HubID          string `json:"hub_id"`
	PublicKey      string `json:"public_key,omitempty"`
	SignatureHex   string `json:"signature_hex"`
}

// KeyProvider abstracts the retrieval of hub public keys for authentication.
type KeyProvider interface {
	GetHubInfo(hubID string) (publicKey []byte, orgID string, err error)
}
