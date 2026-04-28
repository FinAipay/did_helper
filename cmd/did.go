package cmd

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"did_helper/internal/client"
	"did_helper/internal/models"
	"did_helper/internal/storage"

	"github.com/spf13/cobra"
)

var (
	didParam           string
	didEntityType      string
	didEntityId        string
	didExpectedHL      string
	original           bool
	didUpdateFile      string
	didAddKeys         []string
	didRevokeKeys      []string
	didAddServices     []string
	didRemoveServices  []string
	didUpdateMetadata  string
	didForce           bool
)

// didCmd is the DID management command
var didCmd = &cobra.Command{
	Use:   "did",
	Short: "Manage DID documents",
	Long:  "View and manage DID documents with DID-based storage",
}

// didCreateCmd creates a new DID document
var didCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create DID document from existing key",
	Long: `Create a DID document using an existing key from the import directory.
This command will request the DID service to register the DID.

Example:
  ./did_helper did create --entity-type agents --entity-id 0x1234...
  ./did_helper did create --entity-type users --entity-id pubkey123...`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if didEntityType == "" {
			return fmt.Errorf("entity type is required (--entity-type users|agents|devices|services|orgs|assets)")
		}

		if didEntityId == "" {
			return fmt.Errorf("entity ID is required (--entity-id)")
		}

		// Initialize managers
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		importMgr, err := storage.NewImportStorageManager()
		if err != nil {
			return err
		}

		// Find matching key in import directory (using entityId directly, no prefix)
		publicKey := strings.ToLower(didEntityId)

		if !importMgr.KeyExistsInImport(publicKey) {
			return fmt.Errorf("key not found in import directory for entity ID: %s\n\nAvailable keys:\n%s",
				didEntityId, listImportedKeys(importMgr))
		}

		fmt.Printf("Creating DID document...\n")
		fmt.Printf("Entity Type: %s\n", didEntityType)
		fmt.Printf("Entity ID: %s\n", didEntityId)
		fmt.Println()

		// read config
		config, err := sm.LoadConfig()
		req := map[string]interface{}{
			"entityType": didEntityType,
			"entityId":   didEntityId,
		}
		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.Post(config.API+"/api/v1/dids", req)
		err = httpClient.VerifyResponse(statusCode, res, err)
		if err != nil {
			return err
		}

		resp := &models.APIResponse{}
		if err := json.Unmarshal([]byte(res), resp); err != nil {
			return fmt.Errorf("Success but failed to parse: %w\n\rReturn Data:%s", err, res)
		}

		switch didEntityType {
		case "agents":
			query := resp.Data.(map[string]interface{})["metadata_uri"].(string)
			fmt.Printf("Create success %s", resp.Message)
			fmt.Printf("\r\nQuery uri: %s", query)
			go func() {
				time.Sleep(50 * time.Second)
				code, res, err := httpClient.Get(query)
				if code != 200 || err != nil {
					return
				}
				doc := &models.DIDDocument{}
				if err := json.Unmarshal([]byte(res), doc); err != nil {
					return
				}
				sm.SaveDIDDocument(doc.ID, doc)

			}()
			return nil
		default:
			doc := &models.DIDDocument{}
			data, err := json.Marshal(resp.Data)
			if err != nil {
				return fmt.Errorf("Success but failed to parse: %w\n\rReturn Data:%s", err, res)
			}
			if err := json.Unmarshal(data, doc); err != nil {
				return fmt.Errorf("Success but failed to parse: %w\n\rReturn Data:%s", err, res)
			}
			sm.SaveDIDDocument(doc.ID, doc)
			fmt.Println("✓ DID document created successfully!")
			fmt.Printf("DID: %s\n", doc.ID)
			fmt.Printf("Entity Type: %s\n", didEntityType)
			fmt.Printf("Entity ID: %s\n", didEntityId)

			fmt.Println("\n📁 Storage location:")
			fmt.Printf("   ~/.did_helper/%s/\n", didToDirName(doc.ID))

			fmt.Println("\n💡 Tips:")
			fmt.Println("  - View DID: did_helper did show --did " + doc.ID)
			fmt.Println("  - List all DIDs: did_helper did list")
			fmt.Println("  - Switch default: did_helper use " + doc.ID)
		}

		return nil
	},
}

var didQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query DID document",
	Long: `Query a DID document .
This command will request the DID service to register the DID.

Example:
  ./did_helper did query --did did:finai:users:0x3bcc...`,
	RunE: func(cmd *cobra.Command, args []string) error {

		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}
		config, err := sm.LoadConfig()
		if err != nil {
			return err
		}

		did, err := sm.ResolveDID(didParam)
		if err != nil {
			return fmt.Errorf("failed to resolve DID: %w", err)
		}
		http := client.NewHTTPClient()
		statusCode, res, err := http.Get(config.API + "/api/v1/dids/" + did)
		err = http.VerifyResponse(statusCode, res, err)
		if err != nil {
			return err
		}
		resp := &models.APIResponse{}
		err = json.Unmarshal([]byte(res), resp)
		if err != nil {
			return fmt.Errorf("Success but failed to parse: %w\n\rReturn Data:%s", err, res)
		}

		doc := &models.DIDDocument{}
		data, err := json.Marshal(resp.Data)
		if err != nil {
			return fmt.Errorf("Success but failed to parse: %w\n\rReturn Data:%s", err, res)
		}
		if err := json.Unmarshal(data, doc); err != nil {
			return fmt.Errorf("Success but failed to parse: %w\n\rReturn Data:%s", err, res)
		}
		sm.SaveDIDDocument(doc.ID, doc)
		fmt.Println("✓ DID document loaded successfully!")
		// Format and display DID document
		if original {
			jsonData, err := json.MarshalIndent(resp.Data, "", "    ")
			if err != nil {
				return err
			}
			fmt.Println(string(jsonData))
		} else {
			fmt.Printf("DID Document Details:\n")
			fmt.Printf("ID: %s\n", doc.ID)
			fmt.Printf("Entity Type: %s\n", strings.Split(doc.ID, ":")[2])
			fmt.Printf("Deactivate: %t\n", doc.DIDDocumentMetadata.Deactivated)
			fmt.Printf("Created: %s\n", doc.DIDDocumentMetadata.Created)
			fmt.Printf("Updated: %s\n", doc.DIDDocumentMetadata.Updated)
			fmt.Printf("Verification Methods (%d):\n", len(doc.VerificationMethod))
			for i, vm := range doc.VerificationMethod {
				fmt.Printf("  %d. ID: %s\n", i+1, vm.ID)
				fmt.Printf("     Type: %s\n", vm.Type)
				fmt.Printf("     Purpose: %s\n", vm.FinAIPurpose)
				if vm.EthereumAddress != "" {
					fmt.Printf("     Ethereum Address: %s\n", vm.EthereumAddress)
				}
				if vm.PublicKeyBase58 != "" {
					fmt.Printf("     Public Key (Base58): %s\n", vm.PublicKeyBase58)
				}
				if vm.PublicKeyMultibase != "" {
					fmt.Printf("     Public Key (Multibase): %s\n", vm.PublicKeyMultibase)
				}
				fmt.Printf("     Primary Key: %v\n", vm.FinAIIsPrimary)
				fmt.Println()
			}

			fmt.Printf("Service Endpoints (%d):\n", len(doc.Service))
			for i, svc := range doc.Service {
				fmt.Printf("  %d. ID: %s\n", i+1, svc.ID)
				fmt.Printf("     Type: %s\n", svc.Type)
				fmt.Printf("     Description: %s\n", svc.Description)
				fmt.Println()
			}
		}

		return nil
	},
}

func listImportedKeys(importMgr *storage.ImportStorageManager) string {
	keys, err := importMgr.ListImportedKeys()
	if err != nil || len(keys) == 0 {
		return "  (no keys found in import directory)"
	}

	result := ""
	for i, key := range keys {
		result += fmt.Sprintf("  %d. %s\n", i+1, key)
	}
	return result
}

// didListCmd lists all DID documents
var didListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all DID documents",
	RunE: func(cmd *cobra.Command, args []string) error {
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		dids, err := sm.ListDIDs()
		if err != nil {
			return fmt.Errorf("failed to list DIDs: %w", err)
		}

		if len(dids) == 0 {
			fmt.Println("No DID documents found")
			return nil
		}

		fmt.Printf("Found %d DID document(s):\n\n", len(dids))
		for i, did := range dids {
			doc, err := sm.GetDIDDocument(did)
			if err != nil {
				fmt.Printf("%d. DID: %s (document not available)\n", i+1, did)
				continue
			}

			fmt.Printf("%d. DID: %s\n", i+1, doc.ID)
			fmt.Printf("   Entity Type: %s\n", strings.Split(doc.ID, ":")[2])
			fmt.Printf("   Status: %s\n", doc.DIDDocumentMetadata.Deactivated)
			fmt.Printf("   Created: %s\n", doc.DIDDocumentMetadata.Created)
			fmt.Printf("   Updated: %s\n", doc.DIDDocumentMetadata.Updated)
			fmt.Printf("   hl: %s\n", doc.DIDDocumentMetadata.HashLink)
			fmt.Printf("   Version ID: %s\n", doc.DIDDocumentMetadata.VersionID)
			fmt.Printf("   Verification Methods: %d\n", len(doc.VerificationMethod))
			fmt.Printf("   Service Endpoints: %d\n", len(doc.Service))
			fmt.Println()
		}

		return nil
	},
}

// didShowCmd shows details of a specific DID document
var didShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show details of a specific DID document",
	RunE: func(cmd *cobra.Command, args []string) error {
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, err := sm.ResolveDID(didParam)
		if err != nil {
			return fmt.Errorf("failed to resolve DID: %w", err)
		}

		doc, err := sm.GetDIDDocument(did)
		if err != nil {
			return fmt.Errorf("DID document not found: %w", err)
		}

		// Format and display DID document
		fmt.Printf("DID Document Details:\n")
		fmt.Printf("========================\n")
		fmt.Printf("ID: %s\n", doc.ID)
		fmt.Printf("Entity Type: %s\n", strings.Split(doc.ID, ":")[2])
		fmt.Printf("Status: %s\n", doc.DIDDocumentMetadata.Deactivated)
		fmt.Printf("Created: %s\n", doc.DIDDocumentMetadata.Created)
		fmt.Printf("Updated: %s\n", doc.DIDDocumentMetadata.Updated)
		fmt.Printf("Verification Methods (%d):\n", len(doc.VerificationMethod))
		for i, vm := range doc.VerificationMethod {
			fmt.Printf("  %d. ID: %s\n", i+1, vm.ID)
			fmt.Printf("     Type: %s\n", vm.Type)
			fmt.Printf("     Purpose: %s\n", vm.FinAIPurpose)
			if vm.EthereumAddress != "" {
				fmt.Printf("     Ethereum Address: %s\n", vm.EthereumAddress)
			}
			if vm.PublicKeyBase58 != "" {
				fmt.Printf("     Public Key (Base58): %s\n", vm.PublicKeyBase58)
			}
			if vm.PublicKeyMultibase != "" {
				fmt.Printf("     Public Key (Multibase): %s\n", vm.PublicKeyMultibase)
			}
			fmt.Printf("     Primary Key: %v\n", vm.FinAIIsPrimary)
			fmt.Println()
		}

		fmt.Printf("Service Endpoints (%d):\n", len(doc.Service))
		for i, svc := range doc.Service {
			fmt.Printf("  %d. ID: %s\n", i+1, svc.ID)
			fmt.Printf("     Type: %s\n", svc.Type)
			fmt.Printf("     Description: %s\n", svc.Description)
			fmt.Println()
		}

		return nil
	},
}

// didVerifyCmd verifies DID document integrity using hashlink
var didVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify DID document integrity",
	Long: `Verify the integrity of a DID document by calculating and comparing its hashlink.

This command calculates the SHA256 hash of the DID document (excluding hl and versionId fields)
and compares it with the expected hashlink to detect any tampering.

Example:
  # Verify with expected hashlink
  ./did_helper did verify --did did:finai:users:0x1234... --hl zQmX7K9J2L4M6N8P0R3S5T7V9W1X3Y5Z7A9B1C3D5E7F9G1H3
  
  # Calculate current hashlink without comparison
  ./did_helper did verify --did did:finai:users:0x1234...`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, err := sm.ResolveDID(didParam)
		if err != nil {
			return fmt.Errorf("failed to resolve DID: %w", err)
		}

		doc, err := sm.GetDIDDocument(did)
		if err != nil {
			return fmt.Errorf("DID document not found: %w", err)
		}

		fmt.Printf("🔐 Verifying DID document integrity...\n")
		fmt.Printf("📄 Document ID: %s\n", doc.ID)
		fmt.Printf("📊 Excluding fields: hl, versionId\n\n")

		// Convert DID document to map for hash calculation
		docMap, err := convertDIDDocToMap(doc)
		if err != nil {
			return fmt.Errorf("failed to convert document: %w", err)
		}

		// Calculate hashlink
		calculatedHL, err := calculateHashLink(docMap)
		if err != nil {
			return fmt.Errorf("failed to calculate hashlink: %w", err)
		}

		fmt.Printf("✅ Calculated hl: %s\n", calculatedHL)

		// If expected HL is provided, compare
		if didExpectedHL != "" {
			fmt.Printf("🎯 Expected hl:   %s\n\n", didExpectedHL)

			match := calculatedHL == didExpectedHL

			fmt.Println(strings.Repeat("=", 60))
			if match {
				fmt.Println("✅ Document integrity verified successfully!")
				fmt.Println("✓ The document has NOT been tampered with")
			} else {
				fmt.Println("❌ Document hash mismatch detected!")
				fmt.Println("⚠️  WARNING: Document may have been tampered or is outdated")
				fmt.Println(strings.Repeat("=", 60))
				fmt.Printf("  Expected hl:   %s\n", didExpectedHL)
				fmt.Printf("  Calculated hl: %s\n", calculatedHL)
				fmt.Println(strings.Repeat("=", 60))
				return fmt.Errorf("hashlink verification failed")
			}
			fmt.Println(strings.Repeat("=", 60))
		} else {
			fmt.Println("\n💡 Tip: Use --hl flag to verify against expected hashlink")
		}

		return nil
	},
}

// didReputationCmd shows DID reputation
var didReputationCmd = &cobra.Command{
	Use:   "reputation",
	Short: "Show DID reputation",
	Long:  `Query and display the reputation score and level for a DID.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, err := sm.ResolveDID(didParam)
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
		statusCode, res, err := httpClient.GetWithTicket(config.API+"/api/v1/reputation/"+did, ticket.Ticket)
		if err != nil {
			return fmt.Errorf("failed to query reputation: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("failed to query reputation (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("query failed: %s", apiResp.Message)
		}

		// Parse reputation data
		var repData models.ReputationResponse
		dataBytes, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(dataBytes, &repData); err != nil {
			return fmt.Errorf("failed to parse reputation data: %w", err)
		}

		fmt.Printf("📊 DID Reputation\n")
		fmt.Printf("========================\n")
		fmt.Printf("DID: %s\n", did)
		fmt.Printf("Level: %s\n", repData.Level)
		fmt.Printf("Total Score: %d\n", repData.TotleScore)
		fmt.Println()

		return nil
	},
}

// didUpdateCmd updates DID document
var didUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update DID document",
	Long: `Update a DID document by adding/removing keys or services.

Examples:
  # Using JSON file
  ./did_helper did update --did did:finai:users:0x123... --request-file update.json
  
  # Add a service endpoint
  ./did_helper did update --did did:finai:users:0x123... --add-service '{"id":"#svc-1","type":"LinkedDomains","serviceEndpoint":"https://example.com"}'
  
  # Revoke a key
  ./did_helper did update --did did:finai:users:0x123... --revoke-key "#keys-1"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, err := sm.ResolveDID(didParam)
		if err != nil {
			return fmt.Errorf("failed to resolve DID: %w", err)
		}

		// Load ticket
		ticket, err := sm.LoadTicket(did)
		if err != nil {
			return fmt.Errorf("ticket not found or failed to load: %w\n💡 Please run 'did_helper ticket challenge' and 'did_helper ticket verify' first", err)
		}

		// Build update request
		updateReq := models.UpdateDIDRequest{}

		// If request file is provided, load from file
		if didUpdateFile != "" {
			fileData, err := os.ReadFile(didUpdateFile)
			if err != nil {
				return fmt.Errorf("failed to read request file: %w", err)
			}
			if err := json.Unmarshal(fileData, &updateReq); err != nil {
				return fmt.Errorf("failed to parse request file: %w", err)
			}
		} else {
			// Process command line flags
			if len(didAddKeys) > 0 {
				for _, keyJSON := range didAddKeys {
					var vm models.VerificationMethod
					if err := json.Unmarshal([]byte(keyJSON), &vm); err != nil {
						return fmt.Errorf("failed to parse add-key JSON: %w", err)
					}
					updateReq.AddKeys = append(updateReq.AddKeys, &vm)
				}
			}

			if len(didRevokeKeys) > 0 {
				updateReq.RevokeKeys = didRevokeKeys
			}

			if len(didAddServices) > 0 {
				for _, svcJSON := range didAddServices {
					var svc models.Service
					if err := json.Unmarshal([]byte(svcJSON), &svc); err != nil {
						return fmt.Errorf("failed to parse add-service JSON: %w", err)
					}
					updateReq.AddServices = append(updateReq.AddServices, &svc)
				}
			}

			if len(didRemoveServices) > 0 {
				updateReq.RemoveServices = didRemoveServices
			}

			if didUpdateMetadata != "" {
				var metadata map[string]interface{}
				if err := json.Unmarshal([]byte(didUpdateMetadata), &metadata); err != nil {
					return fmt.Errorf("failed to parse update-metadata JSON: %w", err)
				}
				updateReq.UpdateMetadata = metadata
			}
		}

		// Validate request
		if len(updateReq.AddKeys) == 0 && len(updateReq.RevokeKeys) == 0 &&
			len(updateReq.AddServices) == 0 && len(updateReq.RemoveServices) == 0 &&
			len(updateReq.UpdateMetadata) == 0 {
			return fmt.Errorf("no update operations specified")
		}

		config, err := sm.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.PutWithTicket(config.API+"/api/v1/dids/"+did, updateReq, ticket.Ticket)
		if err != nil {
			return fmt.Errorf("failed to update DID: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("failed to update DID (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("update failed: %s", apiResp.Message)
		}

		fmt.Println("✅ DID document updated successfully!")
		fmt.Printf("Message: %s\n", apiResp.Message)

		// Optionally re-query and save updated document
		fmt.Println("\n💡 Tip: Use 'did_helper did query --did " + did + "' to view the updated document")

		return nil
	},
}

// didDeactivateCmd deactivates a DID
var didDeactivateCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "Deactivate a DID",
	Long: `Deactivate (disable) a DID document. This operation is irreversible.

Example:
  ./did_helper did deactivate --did did:finai:users:0x123...
  ./did_helper did deactivate --did did:finai:users:0x123... --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, err := sm.ResolveDID(didParam)
		if err != nil {
			return fmt.Errorf("failed to resolve DID: %w", err)
		}

		// Confirmation unless --force is used
		if !didForce {
			fmt.Printf("⚠️  WARNING: Deactivating DID is IRREVERSIBLE!\n")
			fmt.Printf("DID: %s\n\n", did)
			fmt.Print("Are you sure? Type 'yes' to confirm: ")
			
			var confirmation string
			fmt.Scanln(&confirmation)
			
			if strings.ToLower(confirmation) != "yes" {
				return fmt.Errorf("deactivation cancelled")
			}
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
		statusCode, res, err := httpClient.DeleteWithTicket(config.API+"/api/v1/dids/"+did, ticket.Ticket)
		if err != nil {
			return fmt.Errorf("failed to deactivate DID: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("failed to deactivate DID (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("deactivation failed: %s", apiResp.Message)
		}

		fmt.Println("✅ DID deactivated successfully!")
		fmt.Printf("Message: %s\n", apiResp.Message)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(didCmd)
	didCmd.AddCommand(didCreateCmd)
	didCmd.AddCommand(didQueryCmd)
	didCmd.AddCommand(didListCmd)
	didCmd.AddCommand(didShowCmd)
	didCmd.AddCommand(didVerifyCmd)
	didCmd.AddCommand(didReputationCmd)
	didCmd.AddCommand(didUpdateCmd)
	didCmd.AddCommand(didDeactivateCmd)

	didCreateCmd.Flags().StringVarP(&didEntityType, "entity-type", "e", "", "Entity type (users|agents|devices|services|orgs|assets)")
	didCreateCmd.Flags().StringVarP(&didEntityId, "entity-id", "i", "", "Entity ID (address or public key)")

	didShowCmd.Flags().StringVarP(&didParam, "did", "d", "", "DID identifier (default: from config)")
	didQueryCmd.Flags().StringVarP(&didParam, "did", "d", "", "DID identifier (default: from config)")
	didQueryCmd.Flags().BoolVarP(&original, "original", "o", false, "Original output")
	didVerifyCmd.Flags().StringVarP(&didParam, "did", "d", "", "DID identifier to verify")
	didVerifyCmd.Flags().StringVarP(&didExpectedHL, "hl", "l", "", "Expected hashlink for comparison")
	
	didReputationCmd.Flags().StringVarP(&didParam, "did", "d", "", "DID identifier (default: from config)")
	
	didUpdateCmd.Flags().StringVarP(&didParam, "did", "d", "", "DID identifier (default: from config)")
	didUpdateCmd.Flags().StringVarP(&didUpdateFile, "request-file", "f", "", "JSON file containing update request")
	didUpdateCmd.Flags().StringArrayVar(&didAddKeys, "add-key", []string{}, "Add verification method (JSON format, can be used multiple times)")
	didUpdateCmd.Flags().StringArrayVar(&didRevokeKeys, "revoke-key", []string{}, "Revoke verification method by ID (can be used multiple times)")
	didUpdateCmd.Flags().StringArrayVar(&didAddServices, "add-service", []string{}, "Add service endpoint (JSON format, can be used multiple times)")
	didUpdateCmd.Flags().StringArrayVar(&didRemoveServices, "remove-service", []string{}, "Remove service endpoint by ID (can be used multiple times)")
	didUpdateCmd.Flags().StringVar(&didUpdateMetadata, "update-metadata", "", "Update metadata (JSON format)")
	
	didDeactivateCmd.Flags().StringVarP(&didParam, "did", "d", "", "DID identifier (default: from config)")
	didDeactivateCmd.Flags().BoolVar(&didForce, "force", false, "Skip confirmation prompt")
}
// ============================================================================
// Hashlink Calculation Helper Functions
// ============================================================================

// sortObjectKeys recursively sorts object keys for canonical JSON serialization
func sortObjectKeys(obj interface{}) interface{} {
	switch v := obj.(type) {
	case nil:
		return nil
	case bool, float64, int, string:
		return v
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = sortObjectKeys(item)
		}
		return result
	case map[string]interface{}:
		sortedMap := make(map[string]interface{})
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sortedMap[k] = sortObjectKeys(v[k])
		}
		return sortedMap
	default:
		return v
	}
}

// calculateHashLink calculates SHA256 hash of a document (excluding hl and versionId fields)
func calculateHashLink(document map[string]interface{}) (string, error) {
	// Create a deep copy to avoid modifying original
	docCopy := make(map[string]interface{})
	jsonData, _ := json.Marshal(document)
	json.Unmarshal(jsonData, &docCopy)

	// Exclude hl and versionId fields
	if metadata, ok := docCopy["didDocumentMetadata"].(map[string]interface{}); ok {
		delete(metadata, "hl")
		delete(metadata, "versionId")
	}

	// Sort all keys recursively for canonical form
	sortedDoc := sortObjectKeys(docCopy)

	// Canonical JSON serialization (no whitespace)
	jsonString, err := json.Marshal(sortedDoc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal document: %w", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(jsonString)

	// Encode as Base64URL (no padding)
	base64Url := base64.RawURLEncoding.EncodeToString(hash[:])

	// Add Hashlink prefix
	return "z" + base64Url, nil
}

// convertDIDDocToMap converts DIDDocument struct to map[string]interface{}
func convertDIDDocToMap(doc *models.DIDDocument) (map[string]interface{}, error) {
	// Marshal to JSON then unmarshal to map
	data, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DID document: %w", err)
	}

	var docMap map[string]interface{}
	if err := json.Unmarshal(data, &docMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	return docMap, nil
}
