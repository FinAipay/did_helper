package wallet

import (
	"encoding/hex"
	"fmt"

	"github.com/gagliardetto/solana-go"
)

// SignMessageWithSOL signs a message using Solana Ed25519 private key
func SignMessageWithSOL(privateKeyHex string, message string) (string, error) {
	// Decode private key from hex
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}

	// Create Solana private key from bytes
	var privateKey solana.PrivateKey
	copy(privateKey[:], privateKeyBytes)

	// Sign the message directly (standard Ed25519 signature)
	signature, err := privateKey.Sign([]byte(message))
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	// Return signature as hex string with 0x prefix
	return "0x" + hex.EncodeToString(signature[:]), nil
}

// ValidateSOLPrivateKey validates Solana private key format
func ValidateSOLPrivateKey(privateKeyHex string) bool {
	bytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return false
	}
	if len(bytes) != 64 {
		return false
	}
	var privateKey solana.PrivateKey
	copy(privateKey[:], bytes)
	// Try to get public key to validate
	_ = privateKey.PublicKey()
	return true
}
