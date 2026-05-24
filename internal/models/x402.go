package models

import (
	"time"
)

// EIP712NetworkConfig represents EIP-712 domain configuration for a specific network
type EIP712NetworkConfig struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	ChainID           int64  `json:"chainId"`
	VerifyingContract string `json:"verifyingContract"`
}

// PaymentConfirmationConfig represents payment confirmation threshold configuration
type PaymentConfirmationConfig struct {
	Enabled       bool   `json:"enabled"`
	USDCThreshold string `json:"usdc_threshold"`
}

// X402Config extends Config with x402-specific fields
type X402Config struct {
	X402API              string                      `json:"x402_api"`
	EIP712Networks       map[string]EIP712NetworkConfig `json:"eip712_networks"`
	DefaultNetwork       string                      `json:"default_network"`
	PaymentConfirmation  PaymentConfirmationConfig   `json:"payment_confirmation"`
}

// PaymentRequest represents a custom payment creation request
type PaymentRequest struct {
	Amount           string `json:"amount"`
	RecipientAddress string `json:"recipientAddress"`
}

// OrderResponse represents an order from the API
type OrderResponse struct {
	OrderID          string    `json:"order_id"`
	Status           string    `json:"status"`
	Amount           string    `json:"amount"`
	RecipientAddress string    `json:"recipient_address"`
	PayerAddress     string    `json:"payer_address,omitempty"`
	TxHash           string    `json:"tx_hash,omitempty"`
	Network          string    `json:"network"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ExpiresAt        time.Time `json:"expires_at,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// SigningRequirements represents EIP-712 signing requirements for an order
type SigningRequirements struct {
	Domain      map[string]interface{} `json:"domain"`
	Types       map[string]interface{} `json:"types"`
	PrimaryType string                 `json:"primaryType"`
	Message     map[string]interface{} `json:"message"`
}

// ProcessPaymentRequest represents the payment processing request with signature
type ProcessPaymentRequest struct {
	Address   string              `json:"address"`
	Signature string              `json:"signature"`
	Payload   SigningRequirements `json:"payload"`
	Network   string              `json:"network"`
}

// OrderStatus represents the current status of an order
type OrderStatus struct {
	OrderID   string    `json:"order_id"`
	Status    string    `json:"status"`
	TxHash    string    `json:"tx_hash,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	Message   string    `json:"message,omitempty"`
}

// OrderListResponse represents the response from listing orders
type OrderListResponse struct {
	Orders []OrderResponse `json:"orders"`
	Total  int             `json:"total"`
	Page   int             `json:"page,omitempty"`
	Limit  int             `json:"limit,omitempty"`
}
