package wallet

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"did_helper/internal/models"

	"golang.org/x/crypto/curve25519"
)

// GenerateX25519KeyPair generates an X25519 key pair (simple version)
func GenerateX25519KeyPair() (publicKey, privateKey string, err error) {
	// Generate private key (32 bytes)
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Derive public key from private key
	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &priv)

	// Encode to hex strings
	publicKey = hex.EncodeToString(pub[:])
	privateKey = hex.EncodeToString(priv[:])

	return publicKey, privateKey, nil
}

// GenerateX25519Key generates X25519 key pair with encryption support
func GenerateX25519Key(password string) (*models.Wallet, error) {
	// Generate Ed25519 key pair (for signing)
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 key: %w", err)
	}

	// Derive X25519 public key from Ed25519 private key
	var publicKey [32]byte
	var privateKeyArray [32]byte
	copy(privateKeyArray[:], priv[:32])
	curve25519.ScalarBaseMult(&publicKey, &privateKeyArray)

	// Serialize keys
	privateKeyHex := hex.EncodeToString(priv)
	publicKeyHex := hex.EncodeToString(publicKey[:])

	// Encrypt private key
	var encryptedData models.EncryptedData
	if password != "" {
		encryptedData, err = EncryptPrivateKey(priv, password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
	} else {
		// No password, store but mark as unencrypted
		encryptedData = models.EncryptedData{
			Ciphertext: privateKeyHex,
			Salt:       "",
			Nonce:      "",
			Algorithm:  "none",
		}
	}

	// Create wallet object
	wallet := &models.Wallet{
		ID:          generateWalletID(publicKeyHex),
		Type:        models.WalletTypeX25519,
		Address:     publicKeyHex, // X25519 uses public key as address
		PublicKey:   publicKeyHex,
		PrivateKey:  encryptedData,
		Algorithm:   "X25519KeyAgreementKey2019",
		IsEncrypted: password != "",
		CreatedAt:   getCurrentTime(),
	}

	return wallet, nil
}

// X25519Decrypt decrypts data using X25519 private key
func X25519Decrypt(privateKeyHex string, encryptedData []byte) ([]byte, error) {
	// Decode private key
	privBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privBytes) != 32 {
		return nil, fmt.Errorf("invalid private key length: expected 32 bytes, got %d", len(privBytes))
	}

	// For X25519 challenge-response, we assume the challenge is encrypted with our public key
	// In a real scenario, this would use proper authenticated encryption
	// For now, we'll just return the data as-is (placeholder for actual decryption logic)
	
	// TODO: Implement proper X25519 decryption with symmetric key derivation
	// This requires the sender to encrypt with our public key and include necessary metadata
	
	return encryptedData, nil
}
