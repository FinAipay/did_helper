package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// SignEIP712 signs data using EIP-712 standard
// privateKeyHex: hex-encoded private key (without 0x prefix)
// domain: EIP-712 domain separator configuration
// types: type definitions for the structured data
// message: the actual message to sign
// Returns signature as hex string with 0x prefix
func SignEIP712(privateKeyHex string, domain map[string]interface{}, 
                types map[string]interface{}, message map[string]interface{}) (string, error) {
	
	// Parse private key
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// Build TypedData structure from JSON
	typedData, err := buildTypedDataFromMaps(domain, types, message)
	if err != nil {
		return "", fmt.Errorf("failed to build typed data: %w", err)
	}

	// Calculate hash of the typed data
	hash, _, err := apitypes.TypedDataAndHash(*typedData)
	if err != nil {
		return "", fmt.Errorf("failed to calculate typed data hash: %w", err)
	}

	// Sign the hash
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	// Convert signature to hex string with 0x prefix
	signatureHex := "0x" + hex.EncodeToString(signature)

	return signatureHex, nil
}

// buildTypedDataFromMaps constructs apitypes.TypedData from interface maps
func buildTypedDataFromMaps(domain map[string]interface{}, types map[string]interface{}, 
                            message map[string]interface{}) (*apitypes.TypedData, error) {
	
	// Serialize and deserialize to ensure proper types
	domainJSON, err := json.Marshal(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal domain: %w", err)
	}
	typesJSON, err := json.Marshal(types)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal types: %w", err)
	}
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	var domainData apitypes.TypedDataDomain
	var typesData apitypes.Types
	var messageData apitypes.TypedDataMessage

	if err := json.Unmarshal(domainJSON, &domainData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal domain: %w", err)
	}
	if err := json.Unmarshal(typesJSON, &typesData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal types: %w", err)
	}
	if err := json.Unmarshal(messageJSON, &messageData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Determine PrimaryType
	primaryType := ""
	if pt, ok := types["primaryType"].(string); ok {
		primaryType = pt
	} else {
		// Default to first non-EIP712Domain type
		for typeName := range typesData {
			if typeName != "EIP712Domain" {
				primaryType = typeName
				break
			}
		}
	}

	typedData := &apitypes.TypedData{
		Types:       typesData,
		PrimaryType: primaryType,
		Domain:      domainData,
		Message:     messageData,
	}

	return typedData, nil
}

// PersonalSignSimple signs a message using Ethereum's personal_sign method
// This is simpler than EIP-712 and used for challenge-response
func PersonalSignSimple(privateKeyHex string, message string) (string, error) {
	// Parse private key
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// Hash the message with Ethereum's personal_sign prefix
	hash := accounts.TextHash([]byte(message))

	// Sign the hash
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	// Convert to hex with 0x prefix
	return "0x" + hex.EncodeToString(signature), nil
}
