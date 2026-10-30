package authsvc

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// parsePublicKeyPEM parses PEM encoded PKIX Ed25519 public key ("PUBLIC KEY" block).
func parsePublicKeyPEM(data []byte) (ed25519.PublicKey, error) {
	der, err := decodePEM(data, "PUBLIC KEY")
	if err != nil {
		return nil, fmt.Errorf("decode pem: %w", err)
	}

	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse pkix public key: %w", err)
	}

	edKey, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("unexpected public key type %T", key)
	}

	return edKey, nil
}

// parsePrivateKeyPEM parses PEM encoded PKCS8 Ed25519 private key ("PRIVATE KEY" block).
func parsePrivateKeyPEM(data []byte) (ed25519.PrivateKey, error) {
	der, err := decodePEM(data, "PRIVATE KEY")
	if err != nil {
		return nil, fmt.Errorf("decode pem: %w", err)
	}

	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse pkcs8 private key: %w", err)
	}

	edKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("unexpected private key type %T", key)
	}

	return edKey, nil
}

func decodePEM(data []byte, blockType string) ([]byte, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no pem block found")
	}

	if block.Type != blockType {
		return nil, fmt.Errorf("unexpected pem block type %q, want %q", block.Type, blockType)
	}

	return block.Bytes, nil
}
