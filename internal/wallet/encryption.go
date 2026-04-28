package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"did_helper/internal/models"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptPrivateKey encrypts private key using AES-256-GCM
func EncryptPrivateKey(privateKey []byte, password string) (models.EncryptedData, error) {
	// Generate random salt (32 bytes)
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return models.EncryptedData{}, fmt.Errorf("Failed to generate salt: %w", err)
	}

	// Derive key from password (PBKDF2, 100000 iterations)
	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return models.EncryptedData{}, fmt.Errorf("Failed to create AES cipher: %w", err)
	}

	// Generate random nonce (12 bytes)
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return models.EncryptedData{}, fmt.Errorf("Failed to generate nonce: %w", err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return models.EncryptedData{}, fmt.Errorf("Failed to create GCM mode: %w", err)
	}

	// Encrypt data
	ciphertext := aesGCM.Seal(nil, nonce, privateKey, nil)

	return models.EncryptedData{
		Ciphertext: hex.EncodeToString(ciphertext),
		Salt:       hex.EncodeToString(salt),
		Nonce:      hex.EncodeToString(nonce),
		Algorithm:  "AES-256-GCM",
	}, nil
}

// DecryptPrivateKey decrypts private key
func DecryptPrivateKey(encrypted models.EncryptedData, password string) ([]byte, error) {
	// Decode ciphertext, salt, and nonce
	ciphertext, err := hex.DecodeString(encrypted.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode ciphertext: %w", err)
	}

	salt, err := hex.DecodeString(encrypted.Salt)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode salt: %w", err)
	}

	nonce, err := hex.DecodeString(encrypted.Nonce)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode nonce: %w", err)
	}

	// Derive key from password
	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("Failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("Failed to create GCM mode: %w", err)
	}

	// Decrypt data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("Decryption failed, password may be incorrect: %w", err)
	}

	return plaintext, nil
}
