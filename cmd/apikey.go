package cmd

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"did_helper/internal/client"
	"did_helper/internal/models"
	"did_helper/internal/storage"

	"github.com/spf13/cobra"
)

var (
	apikeyDID      string
	apikeyService  string
	apikeyToRevoke string
)

// apikeyCmd is the API Key management command
var apikeyCmd = &cobra.Command{
	Use:   "apikey",
	Short: "Manage API keys",
	Long:  "Create, list, and revoke API keys for DID authentication",
}

// apikeyCreateCmd creates a new API key
var apikeyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API key",
	Long: `Create a new API key for a DID. The secret will be shown only once!

Example:
  ./did_helper apikey create --did did:finai:users:0x123... --service-name finai-x402
  ./did_helper apikey create --did did:finai:users:0x123...  # Auto-generates service name`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, err := sm.ResolveDID(apikeyDID)
		if err != nil {
			return fmt.Errorf("failed to resolve DID: %w", err)
		}

		// Load ticket
		ticket, err := sm.LoadTicket(did)
		if err != nil {
			return fmt.Errorf("ticket not found or failed to load: %w\n💡 Please run 'did_helper ticket challenge' and 'did_helper ticket verify' first", err)
		}

		// Generate service name if not provided
		serviceName := apikeyService
		if serviceName == "" {
			serviceName = generateRandomServiceName()
			fmt.Printf("ℹ️  Service name not provided, auto-generated: %s\n\n", serviceName)
		}

		config, err := sm.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Prepare request
		createReq := map[string]interface{}{
			"service_name": serviceName,
		}

		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.PostWithTicket(config.API+"/api/v1/api-keys", createReq, ticket.Ticket)
		if err != nil {
			return fmt.Errorf("failed to create API key: %w", err)
		}

		if statusCode != 201 && statusCode != 200 {
			return fmt.Errorf("failed to create API key (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("creation failed: %s", apiResp.Message)
		}

		// Parse API key data
		var keyData models.APIKeyCreateResponse
		dataBytes, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(dataBytes, &keyData); err != nil {
			return fmt.Errorf("failed to parse API key data: %w", err)
		}

		// Display results
		fmt.Println("✅ API key created successfully!")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("Service Name: %s\n", keyData.ServiceName)
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println()

		// Save to apikey.json
		existingKeys, _ := sm.LoadAPIKeys(did)

		newKey := models.APIKeyInfo{
			ID:          len(existingKeys) + 1,
			DID:         did,
			ServiceName: keyData.ServiceName,
			APIKey:      keyData.APIKey,
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		existingKeys = append(existingKeys, newKey)
		if err := sm.SaveAPIKeys(did, existingKeys); err != nil {
			fmt.Printf("⚠️  Warning: Failed to save API key to file: %v\n", err)
		} else {
			fmt.Println("💾 API key information saved to apikey.json")
		}

		return nil
	},
}

// apikeyListCmd lists all API keys
var apikeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all API keys",
	Long:  `List all API keys associated with a DID.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, err := sm.ResolveDID(apikeyDID)
		if err != nil {
			return fmt.Errorf("failed to resolve DID: %w", err)
		}

		// Load ticket
		ticket, err := sm.LoadTicket(did)
		if err != nil {
			return fmt.Errorf("ticket not found or failed to load: %w\n💡 Please run 'did_helper ticket challenge' and 'did_helper ticket verify' first", err)
		}

		config, err := sm.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.GetWithTicket(config.API+"/api/v1/api-keys", ticket.Ticket)
		if err != nil {
			return fmt.Errorf("failed to list API keys: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("failed to list API keys (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("query failed: %s", apiResp.Message)
		}

		// Parse API keys list
		var keys []models.APIKeyInfo
		dataBytes, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(dataBytes, &keys); err != nil {
			return fmt.Errorf("failed to parse API keys: %w", err)
		}

		if len(keys) == 0 {
			fmt.Println("No API keys found")
			return nil
		}

		fmt.Printf("🔑 API Keys for DID: %s\n", did)
		fmt.Printf("Found %d key(s):\n\n", len(keys))

		for i, key := range keys {
			fmt.Printf("%d. ID: %d\n", i+1, key.ID)
			fmt.Printf("   Service: %s\n", key.ServiceName)
			fmt.Printf("   API Key: %s\n", key.APIKey)
			fmt.Printf("   Active:  %v\n", key.IsActive)
			fmt.Printf("   Created: %s\n", key.CreatedAt.Format("2006-01-02 15:04:05"))
			if key.LastUsedAt != nil {
				fmt.Printf("   Last Used: %s\n", key.LastUsedAt.Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("   Last Used: Never\n")
			}
			fmt.Println()
		}

		// Update local storage
		if err := sm.SaveAPIKeys(did, keys); err != nil {
			fmt.Printf("⚠️  Warning: Failed to update local API key cache: %v\n", err)
		}

		return nil
	},
}

// apikeyRevokeCmd revokes an API key
var apikeyRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke an API key",
	Long: `Revoke (delete) an API key. This operation is irreversible.

Example:
  ./did_helper apikey revoke --did did:finai:users:0x123... --api-key pk_live_xxx`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, err := sm.ResolveDID(apikeyDID)
		if err != nil {
			return fmt.Errorf("failed to resolve DID: %w", err)
		}

		if apikeyToRevoke == "" {
			return fmt.Errorf("API key is required (--api-key)")
		}

		// Confirmation
		fmt.Printf("⚠️  WARNING: Revoking API key is IRREVERSIBLE!\n")
		fmt.Printf("DID:     %s\n", did)
		fmt.Printf("API Key: %s\n\n", apikeyToRevoke)
		fmt.Print("Are you sure? Type 'yes' to confirm: ")

		var confirmation string
		fmt.Scanln(&confirmation)

		if strings.ToLower(confirmation) != "yes" {
			return fmt.Errorf("revocation cancelled")
		}

		// Load ticket
		ticket, err := sm.LoadTicket(did)
		if err != nil {
			return fmt.Errorf("ticket not found or failed to load: %w\n💡 Please run 'did_helper ticket challenge' and 'did_helper ticket verify' first", err)
		}

		config, err := sm.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.DeleteWithTicket(config.API+"/api/v1/api-keys/"+apikeyToRevoke, ticket.Ticket)
		if err != nil {
			return fmt.Errorf("failed to revoke API key: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("failed to revoke API key (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("revocation failed: %s", apiResp.Message)
		}

		fmt.Println("✅ API key revoked successfully!")
		fmt.Printf("Message: %s\n", apiResp.Message)

		// Update local storage - remove the revoked key
		existingKeys, _ := sm.LoadAPIKeys(did)
		var updatedKeys []models.APIKeyInfo
		for _, key := range existingKeys {
			if key.APIKey != apikeyToRevoke {
				updatedKeys = append(updatedKeys, key)
			}
		}

		if err := sm.SaveAPIKeys(did, updatedKeys); err != nil {
			fmt.Printf("⚠️  Warning: Failed to update local API key cache: %v\n", err)
		}

		return nil
	},
}

// generateRandomServiceName generates a random 6-character service name
func generateRandomServiceName() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	rand.Seed(time.Now().UnixNano())
	result := make([]byte, 6)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func init() {
	rootCmd.AddCommand(apikeyCmd)
	apikeyCmd.AddCommand(apikeyCreateCmd)
	apikeyCmd.AddCommand(apikeyListCmd)
	apikeyCmd.AddCommand(apikeyRevokeCmd)

	apikeyCreateCmd.Flags().StringVarP(&apikeyDID, "did", "d", "", "DID identifier (default: from config)")
	apikeyCreateCmd.Flags().StringVarP(&apikeyService, "service-name", "s", "", "Service name (optional, auto-generated if not provided)")

	apikeyListCmd.Flags().StringVarP(&apikeyDID, "did", "d", "", "DID identifier (default: from config)")

	apikeyRevokeCmd.Flags().StringVarP(&apikeyDID, "did", "d", "", "DID identifier (default: from config)")
	apikeyRevokeCmd.Flags().StringVarP(&apikeyToRevoke, "api-key", "k", "", "API key to revoke")
}
