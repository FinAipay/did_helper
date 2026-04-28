package wallet

import (
	"encoding/hex"
	"fmt"

	"did_helper/internal/models"

	"github.com/gagliardetto/solana-go"
)

// GenerateSOLWallet generates a Solana wallet
func GenerateSOLWallet(password string) (*models.Wallet, error) {
	// Use solana-go's NewWallet to generate new wallet (creates random key pair)
	wal := solana.NewWallet()

	// Get address and private key
	address := wal.PublicKey().String()
	privateKeyBytes := wal.PrivateKey

	// Encrypt private key
	var encryptedData models.EncryptedData
	var err error
	if password != "" {
		encryptedData, err = EncryptPrivateKey(privateKeyBytes, password)
		if err != nil {
			return nil, fmt.Errorf("Failed to encrypt private key: %w", err)
		}
	} else {
		// No password, store but mark as unencrypted
		encryptedData = models.EncryptedData{
			Ciphertext: hex.EncodeToString(privateKeyBytes),
			Salt:       "",
			Nonce:      "",
			Algorithm:  "none",
		}
	}

	// Create wallet object
	solWallet := &models.Wallet{
		ID:          generateWalletID(address),
		Type:        models.WalletTypeSOL,
		Address:     address,
		PublicKey:   address, // Solana public key is the address
		PrivateKey:  encryptedData,
		IsEncrypted: password != "",
		CreatedAt:   getCurrentTime(),
	}

	return solWallet, nil
}

// ValidateSOLAddress validates Solana address format
func ValidateSOLAddress(address string) bool {
	_, err := solana.PublicKeyFromBase58(address)
	return err == nil
}
