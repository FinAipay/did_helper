package models

import (
	"time"
)

type WalletType string

const (
	WalletTypeETH    WalletType = "ethereum"
	WalletTypeSOL    WalletType = "solana"
	WalletTypeX25519 WalletType = "x25519"
)

type EncryptedData struct {
	Ciphertext string `json:"ciphertext"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Algorithm  string `json:"algorithm"`
}

type Wallet struct {
	ID          string        `json:"id"`
	Type        WalletType    `json:"type"`
	Address     string        `json:"address"`
	PublicKey   string        `json:"public_key"`
	PrivateKey  EncryptedData `json:"private_key"`
	Algorithm   string        `json:"algorithm,omitempty"`
	IsEncrypted bool          `json:"is_encrypted"`
	CreatedAt   time.Time     `json:"created_at"`
}
