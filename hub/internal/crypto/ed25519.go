package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateKeyPair genera un par de llaves Ed25519 (Pública y Privada).
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("error al generar llaves Ed25519: %w", err)
	}
	return pub, priv, nil
}

// Sign firma un mensaje utilizando la llave privada Ed25519.
func Sign(privateKey ed25519.PrivateKey, message []byte) []byte {
	return ed25519.Sign(privateKey, message)
}

// Verify verifica si la firma de un mensaje es válida usando la llave pública Ed25519.
func Verify(publicKey ed25519.PublicKey, message []byte, signature []byte) bool {
	return ed25519.Verify(publicKey, message, signature)
}

// EncodeKeyToHex convierte una llave (pública o privada) a su representación hexadecimal.
func EncodeKeyToHex(key []byte) string {
	return hex.EncodeToString(key)
}

// DecodeKeyFromHex decodifica una llave desde su representación hexadecimal.
func DecodeKeyFromHex(hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("error al decodificar llave hex: %w", err)
	}
	return key, nil
}
