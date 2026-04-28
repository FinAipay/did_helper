package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"did_helper/internal/models"
)

// StorageManager manages local file storage
type StorageManager struct {
	baseDir string
	mu      sync.RWMutex
}

// NewStorageManager creates a new storage manager
func NewStorageManager() (*StorageManager, error) {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("Failed to get user home directory: %w", err)
	}

	homeDir = "./"
	// Set base directory
	baseDir := filepath.Join(homeDir, ".did_helper")

	// Create directory if not exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create config directory: %w", err)
	}

	return &StorageManager{
		baseDir: baseDir,
	}, nil
}

// getFilePath returns the file path
func (s *StorageManager) getFilePath(filename string) string {
	return filepath.Join(s.baseDir, filename)
}

// readJSON reads a JSON file
func (s *StorageManager) readJSON(filename string, data interface{}) error {
	filePath := s.getFilePath(filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File does not exist, return empty list
		return nil
	}

	// Read file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("Failed to read file %s: %w", filename, err)
	}

	// Parse JSON
	if err := json.Unmarshal(fileData, data); err != nil {
		return fmt.Errorf("Failed to parse JSON %s: %w", filename, err)
	}

	return nil
}

// writeJSON writes to a JSON file
func (s *StorageManager) writeJSON(filename string, data interface{}) error {
	filePath := s.getFilePath(filename)

	// Serialize JSON (formatted output)
	fileData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to serialize JSON %s: %w", filename, err)
	}

	// Write file (permission set to owner read/write only)
	if err := os.WriteFile(filePath, fileData, 0600); err != nil {
		return fmt.Errorf("Failed to write file %s: %w", filename, err)
	}

	return nil
}

// SaveDIDDocument saves a DID document
func (s *StorageManager) SaveDIDDocument(did *models.DIDDocument) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read existing list
	var dids []models.DIDDocument
	if err := s.readJSON("dids.json", &dids); err != nil {
		return err
	}

	// Check if already exists
	found := false
	for i, d := range dids {
		if d.ID == did.ID {
			dids[i] = *did
			found = true
			break
		}
	}

	// If not exists, add it
	if !found {
		dids = append(dids, *did)
	}

	// Write to file
	return s.writeJSON("dids.json", dids)
}

// GetDIDDocuments gets all DID documents
func (s *StorageManager) GetDIDDocuments() ([]models.DIDDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var dids []models.DIDDocument
	if err := s.readJSON("dids.json", &dids); err != nil {
		return nil, err
	}

	if dids == nil {
		dids = []models.DIDDocument{}
	}

	return dids, nil
}

// GetDIDDocumentByID gets a DID document by ID
func (s *StorageManager) GetDIDDocumentByID(id string) (*models.DIDDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var dids []models.DIDDocument
	if err := s.readJSON("dids.json", &dids); err != nil {
		return nil, err
	}

	for _, d := range dids {
		if d.ID == id {
			return &d, nil
		}
	}

	return nil, fmt.Errorf("DID document not found: %s", id)
}

// SaveTicket saves a ticket
func (s *StorageManager) SaveTicket(ticket models.Ticket) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read existing list
	var tickets []models.Ticket
	if err := s.readJSON("tickets.json", &tickets); err != nil {
		return err
	}

	// Check if already exists
	found := false
	for i, t := range tickets {
		if t.ID == ticket.ID {
			tickets[i] = ticket
			found = true
			break
		}
	}

	// If not exists, add it
	if !found {
		tickets = append(tickets, ticket)
	}

	// Write to file
	return s.writeJSON("tickets.json", tickets)
}

// GetTickets gets all tickets
func (s *StorageManager) GetTickets() ([]models.Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tickets []models.Ticket
	if err := s.readJSON("tickets.json", &tickets); err != nil {
		return nil, err
	}

	if tickets == nil {
		tickets = []models.Ticket{}
	}

	return tickets, nil
}

// GetTicketByID gets a ticket by ID
func (s *StorageManager) GetTicketByID(id string) (*models.Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tickets []models.Ticket
	if err := s.readJSON("tickets.json", &tickets); err != nil {
		return nil, err
	}

	for _, t := range tickets {
		if t.ID == id {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("Ticket not found: %s", id)
}

// SaveAPIKey saves an API key
func (s *StorageManager) SaveAPIKey(apikey models.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read existing list
	var apikeys []models.APIKey
	if err := s.readJSON("api_keys.json", &apikeys); err != nil {
		return err
	}

	// Check if already exists
	found := false
	for i, k := range apikeys {
		if k.ID == apikey.ID {
			apikeys[i] = apikey
			found = true
			break
		}
	}

	// If not exists, add it
	if !found {
		apikeys = append(apikeys, apikey)
	}

	// Write to file
	return s.writeJSON("api_keys.json", apikeys)
}

// GetAPIKeys gets all API keys
func (s *StorageManager) GetAPIKeys() ([]models.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var apikeys []models.APIKey
	if err := s.readJSON("api_keys.json", &apikeys); err != nil {
		return nil, err
	}

	if apikeys == nil {
		apikeys = []models.APIKey{}
	}

	return apikeys, nil
}

// GetAPIKeyByID gets an API key by ID
func (s *StorageManager) GetAPIKeyByID(id string) (*models.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var apikeys []models.APIKey
	if err := s.readJSON("api_keys.json", &apikeys); err != nil {
		return nil, err
	}

	for _, k := range apikeys {
		if k.ID == id {
			return &k, nil
		}
	}

	return nil, fmt.Errorf("API key not found: %s", id)
}

// SaveWallet saves a wallet
func (s *StorageManager) SaveWallet(wallet models.Wallet) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read existing list
	var wallets []models.Wallet
	if err := s.readJSON("wallets.json", &wallets); err != nil {
		return err
	}

	// Check if already exists
	found := false
	for i, w := range wallets {
		if w.ID == wallet.ID {
			wallets[i] = wallet
			found = true
			break
		}
	}

	// If not exists, add it
	if !found {
		wallets = append(wallets, wallet)
	}

	// Write to file
	return s.writeJSON("wallets.json", wallets)
}

// GetWallets gets all wallets
func (s *StorageManager) GetWallets() ([]models.Wallet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var wallets []models.Wallet
	if err := s.readJSON("wallets.json", &wallets); err != nil {
		return nil, err
	}

	if wallets == nil {
		wallets = []models.Wallet{}
	}

	return wallets, nil
}

// GetWalletByID gets a wallet by ID
func (s *StorageManager) GetWalletByID(id string) (*models.Wallet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var wallets []models.Wallet
	if err := s.readJSON("wallets.json", &wallets); err != nil {
		return nil, err
	}

	for _, w := range wallets {
		if w.ID == id {
			return &w, nil
		}
	}

	return nil, fmt.Errorf("Wallet not found: %s", id)
}

// DeleteWalletByID deletes a wallet by ID
func (s *StorageManager) DeleteWalletByID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var wallets []models.Wallet
	if err := s.readJSON("wallets.json", &wallets); err != nil {
		return err
	}

	// Find and delete
	found := false
	for i, w := range wallets {
		if w.ID == id {
			wallets = append(wallets[:i], wallets[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("Wallet not found: %s", id)
	}

	// Write to file
	return s.writeJSON("wallets.json", wallets)
}
