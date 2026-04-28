package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"did_helper/internal/models"
)

// ImportStorageManager manages imported keys/wallets storage in ~/.did_helper/import/
type ImportStorageManager struct {
	baseDir string // ~/.did_helper/import
}

// NewImportStorageManager creates import storage manager
func NewImportStorageManager() (*ImportStorageManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".did_helper", "import")
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create import directory: %w", err)
	}

	return &ImportStorageManager{baseDir: baseDir}, nil
}

// getImportPath returns the import directory path for a given public key/address
func (m *ImportStorageManager) getImportPath(publicKey string) string {
	// Use lowercase publicKey directly as directory name, keep 0x prefix for ETH addresses
	dirName := strings.ToLower(publicKey)
	// Ensure ETH addresses have 0x prefix
	if !strings.HasPrefix(dirName, "0x") && len(dirName) == 40 {
		dirName = "0x" + dirName
	}
	return filepath.Join(m.baseDir, dirName)
}

// SaveETHWallet saves ETH wallet to import directory
// keystoreJSON should be the raw JSON bytes from go-ethereum keystore to preserve number types
func (m *ImportStorageManager) SaveETHWallet(address string, keystoreJSON []byte, metadata *models.KeyMetadata, password string) error {
	// Normalize address: ensure lowercase with 0x prefix
	cleanAddress := strings.ToLower(address)
	if !strings.HasPrefix(cleanAddress, "0x") {
		cleanAddress = "0x" + cleanAddress
	}
	importPath := m.getImportPath(cleanAddress)

	// Create directory with secure permissions
	if err := os.MkdirAll(importPath, 0700); err != nil {
		return fmt.Errorf("failed to create import directory: %w", err)
	}

	// Save keystore.json (use raw JSON to preserve number types)
	ksPath := filepath.Join(importPath, "keystore.json")
	if err := os.WriteFile(ksPath, keystoreJSON, 0600); err != nil {
		return fmt.Errorf("failed to write keystore: %w", err)
	}

	// Save wallet.json (metadata only, no private key)
	metaPath := filepath.Join(importPath, "wallet.json")
	metaData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metaPath, metaData, 0600); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Save password.txt if provided
	if password != "" {
		passPath := filepath.Join(importPath, "password.txt")
		if err := os.WriteFile(passPath, []byte(password), 0600); err != nil {
			return fmt.Errorf("failed to write password: %w", err)
		}
	}

	return nil
}

// GetETHWallet loads ETH wallet from import directory
func (m *ImportStorageManager) GetETHWallet(address string) (*models.Keystore, *models.KeyMetadata, error) {
	// Normalize address: ensure lowercase with 0x prefix
	cleanAddress := strings.ToLower(address)
	if !strings.HasPrefix(cleanAddress, "0x") {
		cleanAddress = "0x" + cleanAddress
	}
	importPath := m.getImportPath(cleanAddress)

	// Load keystore
	var keystore models.Keystore
	ksPath := filepath.Join(importPath, "keystore.json")
	ksData, err := os.ReadFile(ksPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read keystore: %w", err)
	}
	if err := json.Unmarshal(ksData, &keystore); err != nil {
		return nil, nil, fmt.Errorf("failed to parse keystore: %w", err)
	}

	// Load metadata
	var metadata models.KeyMetadata
	metaPath := filepath.Join(importPath, "wallet.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read metadata: %w", err)
	}
	if err := json.Unmarshal(metaData, &metadata); err != nil {
		return nil, nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &keystore, &metadata, nil
}

// SaveX25519Key saves X25519 key pair to import directory
func (m *ImportStorageManager) SaveX25519Key(publicKey, privateKey, entityType, entityId string) error {
	importPath := m.getImportPath(publicKey)

	// Create directory with secure permissions
	if err := os.MkdirAll(importPath, 0700); err != nil {
		return fmt.Errorf("failed to create import directory: %w", err)
	}

	// Save keypair.json (both public and private key)
	type KeyPair struct {
		PublicKey  string `json:"public_key"`
		PrivateKey string `json:"private_key"`
		Type       string `json:"type"`
	}

	keyPair := KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Type:       "x25519",
	}

	kpPath := filepath.Join(importPath, "keypair.json")
	kpData, err := json.MarshalIndent(keyPair, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keypair: %w", err)
	}
	if err := os.WriteFile(kpPath, kpData, 0600); err != nil {
		return fmt.Errorf("failed to write keypair: %w", err)
	}

	// Save metadata.json
	metadata := map[string]interface{}{
		"type":        "x25519",
		"public_key":  publicKey,
		"entity_type": entityType,
		"entity_id":   entityId,
		"created_at":  time.Now(),
	}

	metaPath := filepath.Join(importPath, "metadata.json")
	metaData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metaPath, metaData, 0600); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// GetX25519Key loads X25519 key pair from import directory
func (m *ImportStorageManager) GetX25519Key(publicKey string) (string, string, error) {
	importPath := m.getImportPath(publicKey)

	// Load keypair
	type KeyPair struct {
		PublicKey  string `json:"public_key"`
		PrivateKey string `json:"private_key"`
		Type       string `json:"type"`
	}

	var keyPair KeyPair
	kpPath := filepath.Join(importPath, "keypair.json")
	kpData, err := os.ReadFile(kpPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read keypair: %w", err)
	}
	if err := json.Unmarshal(kpData, &keyPair); err != nil {
		return "", "", fmt.Errorf("failed to parse keypair: %w", err)
	}

	return keyPair.PublicKey, keyPair.PrivateKey, nil
}

// ListImportedKeys lists all imported keys
func (m *ImportStorageManager) ListImportedKeys() ([]string, error) {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read import directory: %w", err)
	}

	var keys []string
	for _, entry := range entries {
		if entry.IsDir() {
			keys = append(keys, entry.Name())
		}
	}
	return keys, nil
}

// KeyExistsInImport checks if key exists in import directory (by publicKey only, no type prefix)
func (m *ImportStorageManager) KeyExistsInImport(publicKey string) bool {
	importPath := m.getImportPath(publicKey)
	_, err := os.Stat(importPath)
	return err == nil
}

// GetPasswordPath returns the path to password.txt for a given key
func (m *ImportStorageManager) GetPasswordPath(publicKey string) string {
	importPath := m.getImportPath(publicKey)
	return filepath.Join(importPath, "password.txt")
}

// GetImportPath returns the import directory path for a given key
func (m *ImportStorageManager) GetImportPath(publicKey string) string {
	return m.getImportPath(publicKey)
}

// SaveSOLWallet saves Solana wallet to import directory (no encryption, like X25519)
func (m *ImportStorageManager) SaveSOLWallet(publicKey, privateKey string) error {
	importPath := m.getImportPath(publicKey)

	// Create directory with secure permissions
	if err := os.MkdirAll(importPath, 0700); err != nil {
		return fmt.Errorf("failed to create import directory: %w", err)
	}

	// Save keypair.json (both public and private key in hex format)
	type KeyPair struct {
		PublicKey  string `json:"public_key"`
		PrivateKey string `json:"private_key"`
		Type       string `json:"type"`
	}

	keyPair := KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Type:       "solana",
	}

	kpPath := filepath.Join(importPath, "keypair.json")
	kpData, err := json.MarshalIndent(keyPair, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keypair: %w", err)
	}
	if err := os.WriteFile(kpPath, kpData, 0600); err != nil {
		return fmt.Errorf("failed to write keypair: %w", err)
	}

	// Save metadata.json
	metadata := map[string]interface{}{
		"type":       "solana",
		"public_key": publicKey,
		"created_at": time.Now(),
	}

	metaPath := filepath.Join(importPath, "metadata.json")
	metaData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metaPath, metaData, 0600); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// GetSOLWallet loads Solana wallet from import directory
func (m *ImportStorageManager) GetSOLWallet(publicKey string) (string, string, error) {
	importPath := m.getImportPath(publicKey)

	// Load keypair
	type KeyPair struct {
		PublicKey  string `json:"public_key"`
		PrivateKey string `json:"private_key"`
		Type       string `json:"type"`
	}

	var keyPair KeyPair
	kpPath := filepath.Join(importPath, "keypair.json")
	kpData, err := os.ReadFile(kpPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read keypair: %w", err)
	}
	if err := json.Unmarshal(kpData, &keyPair); err != nil {
		return "", "", fmt.Errorf("failed to parse keypair: %w", err)
	}

	return keyPair.PublicKey, keyPair.PrivateKey, nil
}
