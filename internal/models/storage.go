package models

import (
	"time"
)

// Config represents the global configuration
type Config struct {
	Default         DefaultConfig `json:"default"`
	API             string        `json:"api"`
	LastVerify      string        `json:"last_verify"`
	AmountLimit     string        `json:"amount_limit"`
	ChallengeAmount string        `json:"challage_amount"`
	
	// X402 Payment Configuration
	X402API              string                      `json:"x402_api,omitempty"`
	EIP712Networks       map[string]EIP712NetworkConfig `json:"eip712_networks,omitempty"`
	DefaultNetwork       string                      `json:"default_network,omitempty"`
	PaymentConfirmation  PaymentConfirmationConfig   `json:"payment_confirmation,omitempty"`
}

// DefaultConfig represents default DID configuration
type DefaultConfig struct {
	DID  string `json:"did"`
	Name string `json:"name"`
}

// Keystore represents Ethereum standard keystore format
type Keystore struct {
	Address string          `json:"address"`
	Crypto  CryptoInfo      `json:"crypto"`
	ID      string          `json:"id"`
	Version int             `json:"version"`
}

// CryptoInfo represents encryption information in keystore
type CryptoInfo struct {
	Cipher       string                 `json:"cipher"`
	CipherParams CipherParams           `json:"cipherparams"`
	Ciphertext   string                 `json:"ciphertext"`
	KDF          string                 `json:"kdf"`
	KDFParams    KDFParams              `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

// CipherParams represents cipher parameters
type CipherParams struct {
	IV string `json:"iv"`
}

// KDFParams represents key derivation function parameters
type KDFParams struct {
	C     int    `json:"c"`
	DKLen int    `json:"dklen"`
	PRF   string `json:"prf"`
	Salt  string `json:"salt"`
}

// KeyMetadata represents key metadata (NO private key)
type KeyMetadata struct {
	Type      string    `json:"type"`
	Address   string    `json:"address"`
	PublicKey string    `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
}

// WalletMetadata is an alias for backward compatibility
type WalletMetadata = KeyMetadata

// APIKeyData represents API key information
type APIKeyData struct {
	ServiceName string `json:"service_name"`
	APIKey      string `json:"api_key"`
	APISecret   string `json:"api_secret"`
}

// TicketData represents ticket credential
type TicketData struct {
	Ticket string `json:"ticket"`
	Expire string `json:"expire"`
}
