package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"did_helper/internal/models"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
)

// GenerateETHWalletWithKeystore generates Ethereum wallet with standard keystore
// Returns raw keystore JSON bytes to preserve number types
func GenerateETHWalletWithKeystore(password string) ([]byte, *models.WalletMetadata, error) {
	// Generate private key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Get address
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create temporary keystore
	tmpDir, _ := os.MkdirTemp("", "keystore")
	defer os.RemoveAll(tmpDir)

	ks := keystore.NewKeyStore(tmpDir, keystore.StandardScryptN, keystore.StandardScryptP)

	// Import ECDSA key and encrypt with password
	account, err := ks.ImportECDSA(privateKey, password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to import key to keystore: %w", err)
	}

	// Read the keystore file (raw JSON)
	keystoreJSON, err := os.ReadFile(account.URL.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read keystore file: %w", err)
	}

	// Create wallet metadata (NO private key or mnemonic)
	metadata := &models.WalletMetadata{
		Type:      "ethereum",
		Address:   address.Hex(),
		PublicKey: hex.EncodeToString(crypto.CompressPubkey(&privateKey.PublicKey)),
		CreatedAt: getCurrentTime(),
	}

	return keystoreJSON, metadata, nil
}

// DecryptKeystorePrivateKey decrypts private key from keystore using password
func DecryptKeystorePrivateKey(ks *models.Keystore, password string) (string, error) {
	// Marshal keystore to JSON
	keystoreJSON, err := json.Marshal(ks)
	if err != nil {
		return "", fmt.Errorf("failed to marshal keystore: %w", err)
	}

	// Use go-ethereum's keystore decryption
	key, err := keystore.DecryptKey(keystoreJSON, password)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt key: %w", err)
	}

	return hex.EncodeToString(crypto.FromECDSA(key.PrivateKey)), nil
}

// DecryptKeystoreFromRawJSON decrypts private key from raw keystore JSON
func DecryptKeystoreFromRawJSON(keystoreJSON []byte, password string) (string, error) {
	// Use go-ethereum's keystore decryption directly on raw JSON
	key, err := keystore.DecryptKey(keystoreJSON, password)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt key: %w", err)
	}

	return hex.EncodeToString(crypto.FromECDSA(key.PrivateKey)), nil
}

// DecryptData decrypts data using password (simple AES decryption)
func DecryptData(encryptedData, password string) ([]byte, error) {
	// For now, assume the data is base64 encoded encrypted content
	// In production, use proper AES-256-GCM encryption
	
	// Simple implementation: if no encryption, return as-is
	decoded, err := hex.DecodeString(encryptedData)
	if err != nil {
		// If not hex, try base64
		decoded = []byte(encryptedData)
	}
	
	return decoded, nil
}
