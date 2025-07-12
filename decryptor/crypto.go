package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

type Decryptor struct {
	privateKey [32]byte
	publicKey  [32]byte
	gcm        cipher.AEAD
}

func NewDecryptor(privateKey [32]byte) (*Decryptor, error) {
	publicKey, err := curve25519.X25519(privateKey[:], curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	var pubKey [32]byte
	copy(pubKey[:], publicKey)

	return &Decryptor{
		privateKey: privateKey,
		publicKey:  pubKey,
	}, nil
}

func (d *Decryptor) SetupSharedSecret(peerPublicKey [32]byte) error {
	sharedSecret, err := curve25519.X25519(d.privateKey[:], peerPublicKey[:])
	if err != nil {
		return fmt.Errorf("failed to compute shared secret: %w", err)
	}

	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	d.gcm = gcm
	return nil
}

func (d *Decryptor) Decrypt(nonce, encryptedData string) (string, error) {
	if d.gcm == nil {
		return "", fmt.Errorf("decryptor not initialized with shared secret")
	}

	// Decode base64 nonce
	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	// Decode base64 encrypted data
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	// Decrypt
	plaintext, err := d.gcm.Open(nil, nonceBytes, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

func (d *Decryptor) GetPublicKey() [32]byte {
	return d.publicKey
}