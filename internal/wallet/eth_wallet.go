package wallet

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	"did_helper/internal/models"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// GenerateETHWallet generates an Ethereum wallet
func GenerateETHWallet(password string) (*models.Wallet, error) {
	// Generate private key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("Failed to generate private key: %w", err)
	}

	// Get public key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Failed to convert public key type")
	}

	// Get Ethereum address
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	// Serialize private key
	privateKeyBytes := crypto.FromECDSA(privateKey)
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	// Encrypt private key
	var encryptedData models.EncryptedData
	if password != "" {
		encryptedData, err = EncryptPrivateKey([]byte(privateKeyHex), password)
		if err != nil {
			return nil, fmt.Errorf("Failed to encrypt private key: %w", err)
		}
	} else {
		// No password, store but mark as unencrypted
		encryptedData = models.EncryptedData{
			Ciphertext: privateKeyHex,
			Salt:       "",
			Nonce:      "",
			Algorithm:  "none",
		}
	}

	// Create wallet object
	wallet := &models.Wallet{
		ID:          generateWalletID(address),
		Type:        models.WalletTypeETH,
		Address:     address,
		PublicKey:   hex.EncodeToString(crypto.CompressPubkey(publicKeyECDSA)),
		PrivateKey:  encryptedData,
		IsEncrypted: password != "",
		CreatedAt:   getCurrentTime(),
	}

	return wallet, nil
}

// GetETHAddressFromPrivateKey gets Ethereum address from private key
func GetETHAddressFromPrivateKey(privateKeyHex string) (string, error) {
	// Decode private key
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("Failed to decode private key: %w", err)
	}

	// Parse private key
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return "", fmt.Errorf("Failed to parse private key: %w", err)
	}

	// Get address
	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	return address, nil
}

// ValidateETHAddress validates Ethereum address format
func ValidateETHAddress(address string) bool {
	return common.IsHexAddress(address)
}

// PersonalSign signs a message using Ethereum's personal_sign method
func PersonalSign(privateKeyHex string, message string) (string, error) {
	// Decode private key
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}

	// Parse private key
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// Hash the message with Ethereum prefix
	hash := crypto.Keccak256Hash([]byte("\x19Ethereum Signed Message:\n" + fmt.Sprint(len(message)) + message))

	// Sign the hash
	signature, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	// Convert to hex string (add 0x prefix)
	return "0x" + hex.EncodeToString(signature), nil
}
