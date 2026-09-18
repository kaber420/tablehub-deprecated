package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateChallenge creates a random 32-byte challenge for hub authentication.
func GenerateChallenge() ([]byte, string, error) {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, "", fmt.Errorf("generate challenge: %w", err)
	}
	return challenge, hex.EncodeToString(challenge), nil
}

// GenerateKeyPair creates a new Ed25519 key pair and returns hex-encoded strings.
func GenerateKeyPair() (pubHex, privHex string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate key pair: %w", err)
	}
	return hex.EncodeToString(pub), hex.EncodeToString(priv), nil
}

// VerifySignature validates an Ed25519 signature against a public key and message.
// publicKeyHex and signatureHex are hex-encoded strings.
func VerifySignature(publicKeyHex, signatureHex string, message []byte) (bool, error) {
	pubKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false, fmt.Errorf("decode public key: %w", err)
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid public key size: got %d, want %d", len(pubKeyBytes), ed25519.PublicKeySize)
	}

	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, fmt.Errorf("decode signature: %w", err)
	}

	pubKey := ed25519.PublicKey(pubKeyBytes)
	return ed25519.Verify(pubKey, message, sigBytes), nil
}
