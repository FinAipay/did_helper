package models

import (
	"time"
)

// Ticket credential structure
type Ticket struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Value     string    `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ChallengeResponse represents the response from /api/v1/challenge/:did
type ChallengeResponse struct {
	Challenge        string `json:"challenge"`
	ExpiresAt        string `json:"expiresAt"`
	CreatedAt        string `json:"createdAt"`
	BrowserExtension string `json:"browserExtension"`
}

// VerifyChallengeRequest represents the request to /api/v1/verify/challenge
type VerifyChallengeRequest struct {
	DID       string `json:"did"`
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
}

// VerifyChallengeResponse represents the response from /api/v1/verify/challenge
type VerifyChallengeResponse struct {
	Ticket string `json:"ticket"`
	Expire string `json:"expire,omitempty"` // Optional, may not be returned by server
}

// StoredTicket represents the ticket stored locally
type StoredTicket struct {
	Ticket  string `json:"ticket"`
	Created string `json:"created"`
	Expire  string `json:"expire"`
}

// ReputationResponse represents the response from /api/v1/reputation/:did
type ReputationResponse struct {
	Level      string `json:"level"`
	TotleScore int    `json:"totle_score"`
}

// APIKeyCreateResponse represents the response from POST /api/v1/api-keys
type APIKeyCreateResponse struct {
	ServiceName string `json:"service_name"`
	APIKey      string `json:"api_key"`
	APISecret   string `json:"api_secret"`
}

// APIKeyInfo represents an API key entry
type APIKeyInfo struct {
	ID          int        `json:"id"`
	DID         string     `json:"did"`
	ServiceName string     `json:"service_name"`
	APIKey      string     `json:"api_key"`
	APISecret   string     `json:"api_secret"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
}
