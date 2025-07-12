package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/curve25519"
)

type Encryptor struct {
	privateKey [32]byte
	publicKey  [32]byte
	gcm        cipher.AEAD
}

func NewEncryptor(privateKey [32]byte) (*Encryptor, error) {
	publicKey, err := curve25519.X25519(privateKey[:], curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	var pubKey [32]byte
	copy(pubKey[:], publicKey)

	return &Encryptor{
		privateKey: privateKey,
		publicKey:  pubKey,
	}, nil
}

func (e *Encryptor) SetupSharedSecret(peerPublicKey [32]byte) error {
	sharedSecret, err := curve25519.X25519(e.privateKey[:], peerPublicKey[:])
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

	e.gcm = gcm
	return nil
}

type EncryptResult struct {
	Nonce         string
	EncryptedData string
}

func (e *Encryptor) Encrypt(plaintext string) (*EncryptResult, error) {
	if e.gcm == nil {
		return nil, fmt.Errorf("encryptor not initialized with shared secret")
	}

	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := e.gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return &EncryptResult{
		Nonce:         base64.StdEncoding.EncodeToString(nonce),
		EncryptedData: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

func (e *Encryptor) GetPublicKey() [32]byte {
	return e.publicKey
}