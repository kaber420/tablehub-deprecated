package crypto

import (
	"crypto/ed25519"
	"bytes"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("No se esperaba error al generar las llaves: %v", err)
	}

	if len(pub) != ed25519.PublicKeySize {
		t.Errorf("Tamaño incorrecto de llave pública: esperado %d, obtenido %d", ed25519.PublicKeySize, len(pub))
	}

	if len(priv) != ed25519.PrivateKeySize {
		t.Errorf("Tamaño incorrecto de llave privada: esperado %d, obtenido %d", ed25519.PrivateKeySize, len(priv))
	}
}

func TestSignAndVerify(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Error al generar las llaves: %v", err)
	}

	message := []byte("este es un desafio aleatorio de tablehub 12345")
	signature := Sign(priv, message)

	if !Verify(pub, message, signature) {
		t.Error("La firma debería ser válida para el mensaje original")
	}

	// Verificar con mensaje alterado
	alteredMessage := []byte("este es un desafio aleatorio de tablehub 12346")
	if Verify(pub, alteredMessage, signature) {
		t.Error("La firma NO debería ser válida para un mensaje alterado")
	}

	// Verificar con otra llave pública
	otherPub, _, _ := GenerateKeyPair()
	if Verify(otherPub, message, signature) {
		t.Error("La firma NO debería ser válida con una llave pública diferente")
	}
}

func TestHexEncoding(t *testing.T) {
	pub, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Error al generar las llaves: %v", err)
	}

	hexStr := EncodeKeyToHex(pub)
	decoded, err := DecodeKeyFromHex(hexStr)
	if err != nil {
		t.Fatalf("Error al decodificar la llave hex: %v", err)
	}

	if !bytes.Equal(pub, decoded) {
		t.Error("La llave decodificada no coincide con la original")
	}
}
