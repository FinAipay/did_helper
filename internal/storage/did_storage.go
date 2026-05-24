package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"did_helper/internal/models"
)

// DIDStorageManager manages DID-based storage structure
type DIDStorageManager struct {
	baseDir string
	mu      sync.RWMutex
}

// NewDIDStorageManager creates a new DID storage manager
func NewDIDStorageManager() (*DIDStorageManager, error) {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Set base directory
	baseDir := filepath.Join(homeDir, ".did_helper")

	// Create directory if not exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &DIDStorageManager{
		baseDir: baseDir,
	}, nil
}

// getDIDDirectoryName converts DID to directory name (replace : with -)
// Example: did:finai:agents:0x123 -> did-finai-agents-0x123
func getDIDDirectoryName(did string) string {
	return strings.ReplaceAll(did, ":", "-")
}

// getDIDPath returns the DID directory path
func (s *DIDStorageManager) getDIDPath(did string) string {
	dirName := getDIDDirectoryName(did)
	return filepath.Join(s.baseDir, dirName)
}

// getFilePath returns file path within DID directory
func (s *DIDStorageManager) getFilePath(did, filename string) string {
	return filepath.Join(s.getDIDPath(did), filename)
}

// EnsureDIDDirectory creates DID directory if not exists
func (s *DIDStorageManager) EnsureDIDDirectory(did string) error {
	didPath := s.getDIDPath(did)

	// Check if directory already exists
	if _, err := os.Stat(didPath); err == nil {
		return nil // Directory already exists
	}

	// Create directory with lock protection
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.ensureDIDDirectoryInternal(did)
}

// ensureDIDDirectoryInternal creates DID directory without acquiring lock (caller must hold lock)
func (s *DIDStorageManager) ensureDIDDirectoryInternal(did string) error {
	didPath := s.getDIDPath(did)

	// Double-check after acquiring lock
	if _, err := os.Stat(didPath); err == nil {
		return nil // Another goroutine created it
	}

	return os.MkdirAll(didPath, 0700)
}

// readJSON reads a JSON file from DID directory
func (s *DIDStorageManager) readJSON(did, filename string, data interface{}) error {
	filePath := s.getFilePath(did, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filename)
	}

	// Read file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// Parse JSON
	if err := json.Unmarshal(fileData, data); err != nil {
		return fmt.Errorf("failed to parse JSON %s: %w", filename, err)
	}

	return nil
}

// writeJSON writes to a JSON file in DID directory
func (s *DIDStorageManager) writeJSON(did, filename string, data interface{}) error {
	filePath := s.getFilePath(did, filename)

	// Serialize JSON (formatted output)
	fileData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize JSON %s: %w", filename, err)
	}

	// Write file (permission set to owner read/write only)
	if err := os.WriteFile(filePath, fileData, 0600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	return nil
}

// writeText writes text content to a file in DID directory
func (s *DIDStorageManager) writeText(did, filename, content string) error {
	filePath := s.getFilePath(did, filename)

	// Write file (permission set to owner read/write only)
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	return nil
}

// SaveConfig saves global configuration
func (s *DIDStorageManager) SaveConfig(config *models.Config) error {
	configPath := filepath.Join(s.baseDir, "config.json")

	fileData, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(configPath, fileData, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// LoadConfig loads global configuration
func (s *DIDStorageManager) LoadConfig() (*models.Config, error) {
	configPath := filepath.Join(s.baseDir, "config.json")

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config
		return &models.Config{
			Default: models.DefaultConfig{
				DID:  "",
				Name: "",
			},
			API:             "https://didapi.finai.network",
			LastVerify:      "",
			AmountLimit:     "1 USDC",
			ChallengeAmount: "0.5 USDC",
		}, nil
	}

	// Read config
	fileData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config models.Config
	if err := json.Unmarshal(fileData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// GetDefaultDID gets default DID from config
func (s *DIDStorageManager) GetDefaultDID() (string, error) {
	config, err := s.LoadConfig()
	if err != nil {
		return "", err
	}

	if config.Default.DID == "" {
		return "", fmt.Errorf("no default DID configured")
	}

	return config.Default.DID, nil
}

// ResolveDID resolves DID (use provided or default from config)
func (s *DIDStorageManager) ResolveDID(did string) (string, error) {
	if did != "" {
		return did, nil
	}

	return s.GetDefaultDID()
}

// SaveDIDDocument saves DID document
func (s *DIDStorageManager) SaveDIDDocument(did string, doc *models.DIDDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveDIDDocumentInternal(did, doc)
}

// saveDIDDocumentInternal saves DID document without acquiring lock (caller must hold lock)
func (s *DIDStorageManager) saveDIDDocumentInternal(did string, doc *models.DIDDocument) error {
	// Ensure directory exists (use internal version to avoid deadlock)
	if err := s.ensureDIDDirectoryInternal(did); err != nil {
		return err
	}

	// Filename uses directory naming convention
	filename := getDIDDirectoryName(did) + ".json"
	return s.writeJSON(did, filename, doc)
}

// GetDIDDocument loads DID document
func (s *DIDStorageManager) GetDIDDocument(did string) (*models.DIDDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filename := getDIDDirectoryName(did) + ".json"
	var doc models.DIDDocument

	if err := s.readJSON(did, filename, &doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

// SaveKeystore saves Ethereum keystore
func (s *DIDStorageManager) SaveKeystore(did string, keystore *models.Keystore) error {
	if err := s.EnsureDIDDirectory(did); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writeJSON(did, "keystore.json", keystore)
}

// GetKeystore loads Ethereum keystore
func (s *DIDStorageManager) GetKeystore(did string) (*models.Keystore, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keystore models.Keystore
	if err := s.readJSON(did, "keystore.json", &keystore); err != nil {
		return nil, err
	}

	return &keystore, nil
}

// SaveKeyMetadata saves key metadata (NO private key)
func (s *DIDStorageManager) SaveKeyMetadata(did string, metadata *models.KeyMetadata) error {
	if err := s.EnsureDIDDirectory(did); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writeJSON(did, "wallet.json", metadata)
}

// GetKeyMetadata loads key metadata
func (s *DIDStorageManager) GetKeyMetadata(did string) (*models.KeyMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var metadata models.KeyMetadata
	if err := s.readJSON(did, "wallet.json", &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// KeyExists checks if key already exists for a DID
func (s *DIDStorageManager) KeyExists(did string) bool {
	didPath := s.getDIDPath(did)

	// Check if wallet.json or keystore.json exists
	walletPath := filepath.Join(didPath, "wallet.json")
	keystorePath := filepath.Join(didPath, "keystore.json")

	_, walletErr := os.Stat(walletPath)
	_, keystoreErr := os.Stat(keystorePath)

	// If either file exists, key already exists
	return walletErr == nil || keystoreErr == nil
}

// SaveWalletMetadata saves wallet metadata (NO private key or mnemonic)
func (s *DIDStorageManager) SaveWalletMetadata(did string, metadata *models.WalletMetadata) error {
	if err := s.EnsureDIDDirectory(did); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writeJSON(did, "wallet.json", metadata)
}

// GetWalletMetadata loads wallet metadata
func (s *DIDStorageManager) GetWalletMetadata(did string) (*models.WalletMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var metadata models.WalletMetadata
	if err := s.readJSON(did, "wallet.json", &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// SaveAPIKey saves API key information
func (s *DIDStorageManager) SaveAPIKey(did string, apikey *models.APIKeyData) error {
	if err := s.EnsureDIDDirectory(did); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writeJSON(did, "apikey.json", apikey)
}

// GetAPIKey loads API key information
func (s *DIDStorageManager) GetAPIKey(did string) (*models.APIKeyData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var apikey models.APIKeyData
	if err := s.readJSON(did, "apikey.json", &apikey); err != nil {
		return nil, err
	}

	return &apikey, nil
}

// SaveTicket saves ticket credential
func (s *DIDStorageManager) SaveTicket(did string, ticket *models.TicketData) error {
	if err := s.EnsureDIDDirectory(did); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writeJSON(did, "ticket.json", ticket)
}

// GetTicket loads ticket credential
func (s *DIDStorageManager) GetTicket(did string) (*models.TicketData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ticket models.TicketData
	if err := s.readJSON(did, "ticket.json", &ticket); err != nil {
		return nil, err
	}

	return &ticket, nil
}

// SavePassword saves password hint/note (plaintext, no CLI access)
func (s *DIDStorageManager) SavePassword(did, password string) error {
	if err := s.EnsureDIDDirectory(did); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.writeText(did, "password.txt", password)
}

// ListDIDs lists all DID directories
func (s *DIDStorageManager) ListDIDs() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var dids []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "did-") {
			// Convert directory name back to DID format
			did := strings.ReplaceAll(entry.Name(), "-", ":")
			dids = append(dids, did)
		}
	}

	return dids, nil
}

// DIDExists checks if DID directory exists
func (s *DIDStorageManager) DIDExists(did string) bool {
	didPath := s.getDIDPath(did)
	_, err := os.Stat(didPath)
	return err == nil
}

// InitializeDefaultConfig initializes default config if not exists
func (s *DIDStorageManager) InitializeDefaultConfig() error {
	config, err := s.LoadConfig()
	if err != nil {
		return err
	}

	// If config is empty, initialize it
	if config.Default.DID == "" && config.API == "" {
		config.API = "http://localhost:8080"
		config.AmountLimit = "1 USDC"
		config.ChallengeAmount = "0.5 USDC"
		
		// Initialize x402 configuration
		config.X402API = "https://x402.finai.network/testnet/base-sepolia"
		config.DefaultNetwork = "base-sepolia"
		
		// Initialize EIP-712 networks
		config.EIP712Networks = map[string]models.EIP712NetworkConfig{
			"base-sepolia": {
				Name:              "FinAI Payment",
				Version:           "1",
				ChainID:           84532,
				VerifyingContract: "0x0000000000000000000000000000000000000000",
			},
			"ethereum": {
				Name:              "FinAI Payment",
				Version:           "1",
				ChainID:           1,
				VerifyingContract: "0x0000000000000000000000000000000000000000",
			},
		}
		
		// Initialize payment confirmation
		config.PaymentConfirmation = models.PaymentConfirmationConfig{
			Enabled:       true,
			USDCThreshold: "100.00",
		}
		
		return s.SaveConfig(config)
	}

	return nil
}

// UpdateDefaultDID updates default DID in config
func (s *DIDStorageManager) UpdateDefaultDID(did, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.updateDefaultDIDInternal(did, name)
}

// updateDefaultDIDInternal updates default DID without acquiring lock (caller must hold lock)
func (s *DIDStorageManager) updateDefaultDIDInternal(did, name string) error {
	config, err := s.LoadConfig()
	if err != nil {
		return err
	}

	config.Default.DID = did
	config.Default.Name = name
	config.LastVerify = time.Now().Format("2006-01-02 15:04:05")

	return s.SaveConfig(config)
}

// UseDID sets the default DID in config
func (s *DIDStorageManager) UseDID(did string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config, err := s.LoadConfig()
	if err != nil {
		return err
	}

	// Verify DID exists
	if !s.DIDExists(did) {
		return fmt.Errorf("DID not found: %s", did)
	}

	// Extract entity ID from DID for name
	parts := strings.Split(did, ":")
	entityId := ""
	if len(parts) >= 4 {
		entityId = parts[3]
	}

	config.Default.DID = did
	config.Default.Name = entityId
	config.LastVerify = time.Now().Format("2006-01-02 15:04:05")

	return s.SaveConfig(config)
}

// CreateDIDDocument creates DID document by requesting DID service
// Returns the created DID identifier
// func (s *DIDStorageManager) CreateDIDDocument(entityType, entityId string, publicKey string, keyType string) (string, error) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	// Step 1: Verify key exists in import directory
// 	importMgr, err := NewImportStorageManager()
// 	if err != nil {
// 		return "", fmt.Errorf("failed to initialize import storage: %w", err)
// 	}

// 	if !importMgr.KeyExistsInImport(keyType, publicKey) {
// 		return "", fmt.Errorf("key not found in import directory. Generate key first using 'key generate'")
// 	}

// 	// Step 2: Build DID identifier
// 	did := models.FormatDID(entityType, entityId)

// 	// Step 3: Check if DID already exists
// 	if s.DIDExists(did) {
// 		return "", fmt.Errorf("DID already exists: %s", did)
// 	}

// 	// Step 4: Create DID document structure
// 	doc, err := models.NewDIDDocument(entityType, entityId)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to create DID document: %w", err)
// 	}

// 	// Step 5: Add verification method based on key type
// 	vm := s.buildVerificationMethod(did, publicKey, keyType)
// 	doc.AddVerificationMethod(vm)

// 	// Add to appropriate relationship
// 	switch keyType {
// 	case "ethereum", "eth":
// 		doc.AddToAuthentication(vm.ID)
// 		doc.AddToAssertionMethod(vm.ID)
// 	case "x25519":
// 		doc.AddToKeyAgreement(vm.ID)
// 	}

// 	// Step 6: Save DID document (use internal version to avoid deadlock)
// 	if err := s.saveDIDDocumentInternal(did, doc); err != nil {
// 		return "", fmt.Errorf("failed to save DID document: %w", err)
// 	}

// 	// Step 7: Update default DID
// 	s.updateDefaultDIDInternal(did, entityId)

// 	return did, nil
// }

// buildVerificationMethod builds W3C-compliant verification method
func (s *DIDStorageManager) buildVerificationMethod(did, publicKey, keyType string) models.VerificationMethod {
	now := time.Now()

	var vm models.VerificationMethod

	switch keyType {
	case "ethereum", "eth":
		vm = models.VerificationMethod{
			ID:              did + "#keys-1",
			Type:            models.KeyTypeEthereum,
			Controller:      did,
			EthereumAddress: publicKey, // For ETH, publicKey is the address
			FinAIChain:      "ethereum",
			FinAINetwork:    "mainnet",
			FinAIPurpose:    models.PurposeAuthentication,
			FinAIIsPrimary:  true,
			CreatedAt:       now,
		}

	case "x25519":
		vm = models.VerificationMethod{
			ID:              did + "#keys-1",
			Type:            models.KeyTypeX25519,
			Controller:      did,
			PublicKeyBase58: publicKey,
			FinAIPurpose:    models.PurposeKeyAgreement,
			FinAIIsPrimary:  true,
			CreatedAt:       now,
		}
	}

	return vm
}

// LoadTicket loads ticket from DID directory
func (s *DIDStorageManager) LoadTicket(did string) (*models.StoredTicket, error) {
	didPath := s.getDIDPath(did)
	ticketPath := filepath.Join(didPath, "ticket.json")

	data, err := os.ReadFile(ticketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ticket: %w", err)
	}

	var ticket models.StoredTicket
	if err := json.Unmarshal(data, &ticket); err != nil {
		return nil, fmt.Errorf("failed to parse ticket: %w", err)
	}

	return &ticket, nil
}

// SaveAPIKeys saves API keys to DID directory
func (s *DIDStorageManager) SaveAPIKeys(did string, keys []models.APIKeyInfo) error {
	didPath := s.getDIDPath(did)
	apikeyPath := filepath.Join(didPath, "apikey.json")

	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal API keys: %w", err)
	}

	if err := os.WriteFile(apikeyPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write API keys: %w", err)
	}

	return nil
}

// LoadAPIKeys loads API keys from DID directory
func (s *DIDStorageManager) LoadAPIKeys(did string) ([]models.APIKeyInfo, error) {
	didPath := s.getDIDPath(did)
	apikeyPath := filepath.Join(didPath, "apikey.json")

	// Check if file exists
	if _, err := os.Stat(apikeyPath); os.IsNotExist(err) {
		return []models.APIKeyInfo{}, nil
	}

	data, err := os.ReadFile(apikeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read API keys: %w", err)
	}

	var keys []models.APIKeyInfo
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, fmt.Errorf("failed to parse API keys: %w", err)
	}

	return keys, nil
}
