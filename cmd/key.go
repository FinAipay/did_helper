package cmd

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"syscall"

	"did_helper/internal/storage"
	"did_helper/internal/wallet"

	"github.com/gagliardetto/solana-go"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	keyType       string
	keyPassword   string
	keyPromptPass bool
	keyDidParam   string
)

// keyCmd is the cryptographic key management command
var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage cryptographic keys",
	Long:  "Generate and manage ETH, Solana, and X25519 key pairs with DID-based storage",
}

// keyGenerateCmd generates a new key pair
var keyGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new key pair",
	Long: `Generate a new cryptographic key pair and save it to the import directory.
The key can later be used to create a DID document with any entity type.

Example:
  ./did_helper key generate --type ethereum --password "mypass"
  ./did_helper key generate --type solana
  ./did_helper key generate --type x25519`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if keyType == "" {
			return fmt.Errorf("key type is required (--type ethereum|solana|x25519)")
		}

		// Handle password (only for ethereum)
		password := keyPassword

		// If interactive password input is needed (ethereum only)
		if keyPromptPass || (keyType == "ethereum" && password == "") {
			fmt.Print("Enter encryption password: ")
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			fmt.Println()
			password = string(bytePassword)

			if password == "" {
				fmt.Println("⚠️  Warning: No password set. Key will have lower security!")
				fmt.Print("Continue? (y/N): ")
				reader := bufio.NewReader(os.Stdin)
				confirm, _ := reader.ReadString('\n')
				confirm = strings.TrimSpace(confirm)
				if confirm != "y" && confirm != "Y" {
					return fmt.Errorf("operation cancelled")
				}
			} else {
				// Validate password strength
				if err := wallet.ValidatePassword(password); err != nil {
					fmt.Printf("⚠️  Weak password: %v\n", err)
					fmt.Print("Continue with weak password? (y/N): ")
					reader := bufio.NewReader(os.Stdin)
					confirm, _ := reader.ReadString('\n')
					confirm = strings.TrimSpace(confirm)
					if confirm != "y" && confirm != "Y" {
						return fmt.Errorf("operation cancelled")
					}
				}

				// Confirm password
				fmt.Print("Confirm password: ")
				byteConfirmPassword, err := term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return fmt.Errorf("failed to read confirmation password: %w", err)
				}
				fmt.Println()

				if password != string(byteConfirmPassword) {
					return fmt.Errorf("passwords do not match")
				}
			}
		}

		// Generate key based on type
		switch keyType {
		case "ethereum", "eth":
			return generateETHKey(password)
		case "solana", "sol":
			return generateSOLKey()
		case "x25519":
			return generateX25519Key()
		default:
			return fmt.Errorf("unsupported key type: %s (supported: ethereum, solana, x25519)", keyType)
		}
	},
}

// generateETHKey generates Ethereum key with keystore and saves to import directory
func generateETHKey(password string) error {
	fmt.Println("Generating Ethereum key pair...")

	// Initialize import storage manager
	importMgr, err := storage.NewImportStorageManager()
	if err != nil {
		return err
	}

	// Generate keystore and metadata (returns raw JSON)
	keystoreJSON, metadata, err := wallet.GenerateETHWalletWithKeystore(password)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Use address as identifier (lowercase)
	address := strings.ToLower(metadata.Address)

	// Check if key already exists in import directory
	if importMgr.KeyExistsInImport(address) {
		return fmt.Errorf("key already exists in import directory: %s", address)
	}

	// Save to import directory (NOT DID directory)
	if err := importMgr.SaveETHWallet(address, keystoreJSON, metadata, password); err != nil {
		return fmt.Errorf("failed to save key to import directory: %w", err)
	}

	// Display success message
	fmt.Println("\n✓ Key generated successfully!")
	fmt.Printf("========================\n")
	fmt.Printf("Type: ethereum\n")
	fmt.Printf("Address: %s\n", address)
	fmt.Printf("Public Key: %s\n", metadata.PublicKey)
	fmt.Printf("Created: %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))

	if password != "" {
		fmt.Println("\n🔐 Security:")
		fmt.Println("  - Private key encrypted in keystore.json")
		fmt.Println("  - Password saved to password.txt (file system access only)")
		fmt.Println("  - ⚠️  Cannot regenerate - backup your credentials!")
	}

	fmt.Println("\n💡 Next steps:")
	fmt.Println("   1. Create DID for agents: did_helper did create --entity-type agents --entity-id " + address)
	fmt.Println("   2. Create DID for users:  did_helper did create --entity-type users --entity-id " + address)
	fmt.Println("   3. View key info:         did_helper key show --address " + address)

	return nil
}

// generateX25519Key generates X25519 key pair and saves to import directory (no keystore, no password)
func generateX25519Key() error {
	fmt.Println("Generating X25519 key pair...")

	// Initialize import storage manager
	importMgr, err := storage.NewImportStorageManager()
	if err != nil {
		return err
	}

	// Generate X25519 key
	publicKey, privateKey, err := wallet.GenerateX25519KeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate X25519 key: %w", err)
	}

	// Check if key already exists in import directory (lowercase)
	publicKeyLower := strings.ToLower(publicKey)
	if importMgr.KeyExistsInImport(publicKeyLower) {
		return fmt.Errorf("key already exists in import directory: %s", publicKeyLower)
	}

	// Save to import directory (NOT DID directory)
	// For X25519, we don't need entityType/entityId at this stage
	if err := importMgr.SaveX25519Key(publicKeyLower, privateKey, "", ""); err != nil {
		return fmt.Errorf("failed to save key to import directory: %w", err)
	}

	// Display success message
	fmt.Println("\n✓ X25519 key generated successfully!")
	fmt.Printf("========================\n")
	fmt.Printf("Type: x25519\n")
	fmt.Printf("Public Key: %s\n", publicKey)

	fmt.Println("\n📝 Note:")
	fmt.Println("  - X25519 keys are for key agreement (encryption)")
	fmt.Println("  - No keystore or password file generated")

	fmt.Println("\n💡 Next steps:")
	fmt.Println("   1. Create DID for devices: did_helper did create --entity-type devices --entity-id " + publicKey)
	fmt.Println("   2. Create DID for services: did_helper did create --entity-type services --entity-id " + publicKey)

	return nil
}

// generateSOLKey generates Solana key pair and saves to import directory (no keystore, no password)
func generateSOLKey() error {
	fmt.Println("Generating Solana key pair...")

	// Initialize import storage manager
	importMgr, err := storage.NewImportStorageManager()
	if err != nil {
		return err
	}

	// Generate Solana wallet using solana-go library
	wal := solana.NewWallet()
	publicKey := wal.PublicKey().String()
	privateKeyBytes := []byte(wal.PrivateKey)
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	// Check if key already exists in import directory (lowercase)
	publicKeyLower := strings.ToLower(publicKey)
	if importMgr.KeyExistsInImport(publicKeyLower) {
		return fmt.Errorf("key already exists in import directory: %s", publicKeyLower)
	}

	// Save to import directory (NOT DID directory)
	if err := importMgr.SaveSOLWallet(publicKeyLower, privateKeyHex); err != nil {
		return fmt.Errorf("failed to save key to import directory: %w", err)
	}

	// Display success message
	fmt.Println("\n✓ Solana key generated successfully!")
	fmt.Printf("========================\n")
	fmt.Printf("Type: solana\n")
	fmt.Printf("Address: %s\n", publicKey)
	fmt.Printf("Public Key: %s\n", publicKey)

	fmt.Println("\n📝 Note:")
	fmt.Println("  - Solana keys use Ed25519 signature scheme")
	fmt.Println("  - No keystore or password file generated")
	fmt.Println("  - Private key stored in keypair.json (hex encoded)")

	fmt.Println("\n💡 Next steps:")
	fmt.Println("   1. Create DID for users:  did_helper did create --entity-type users --entity-id " + publicKey)
	fmt.Println("   2. Create DID for agents: did_helper did create --entity-type agents --entity-id " + publicKey)
	fmt.Println("   3. View key info:         did_helper key show --address " + publicKey)

	return nil
}

// keyShowCmd shows key public information from import directory
var keyShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show key public information",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if address flag is provided
		if keyDidParam == "" {
			return fmt.Errorf("address is required (use --address)")
		}

		importMgr, err := storage.NewImportStorageManager()
		if err != nil {
			return err
		}

		// Try to load ETH wallet first
		_, metadata, err := importMgr.GetETHWallet(keyDidParam)
		if err == nil {
			// ETH wallet found
			fmt.Printf("Key Information:\n")
			fmt.Printf("====================\n")
			fmt.Printf("Type: ethereum\n")
			fmt.Printf("Address: %s\n", metadata.Address)
			fmt.Printf("Public Key: %s\n", metadata.PublicKey)
			fmt.Printf("Created: %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Println("   Never attempt to read files from a directory(Any Agent or Service)")
			return nil
		}

		// If not ETH, try X25519
		publicKey, _, err := importMgr.GetX25519Key(keyDidParam)
		if err == nil {
			// X25519 key found
			fmt.Printf("Key Information:\n")
			fmt.Printf("====================\n")
			fmt.Printf("Type: x25519\n")
			fmt.Printf("Public Key: %s\n", publicKey)

			fmt.Println("   Never attempt to read files from a directory(Any Agent or Service)")
			return nil
		}

		// If not X25519, try Solana
		solPubKey, _, err := importMgr.GetSOLWallet(keyDidParam)
		if err == nil {
			// Solana wallet found
			fmt.Printf("Key Information:\n")
			fmt.Printf("====================\n")
			fmt.Printf("Type: solana\n")
			fmt.Printf("Address: %s\n", solPubKey)
			fmt.Printf("Public Key: %s\n", solPubKey)

			fmt.Println("   Never attempt to read files from a directory(Any Agent or Service)")
			return nil
		}

		return fmt.Errorf("key not found for address/public key: %s", keyDidParam)
	},
}

// keyListCmd lists all keys from import directory
var keyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		importMgr, err := storage.NewImportStorageManager()
		if err != nil {
			return err
		}

		keys, err := importMgr.ListImportedKeys()
		if err != nil {
			return fmt.Errorf("failed to list keys: %w", err)
		}

		if len(keys) == 0 {
			fmt.Println("No keys found in import directory")
			return nil
		}

		fmt.Printf("Found %d key(s) in import directory:\n\n", len(keys))
		for i, keyDir := range keys {
			fmt.Printf("%d. %s\n", i+1, keyDir)

			// Try to get more info based on key type
			if strings.HasPrefix(keyDir, "0x") && len(keyDir) == 42 {
				// Ethereum address (0x + 40 hex chars)
				address := keyDir
				_, metadata, err := importMgr.GetETHWallet(address)
				if err == nil {
					fmt.Printf("   Type: ethereum\n")
					fmt.Printf("   Address: %s\n", maskAddr(metadata.Address))
					fmt.Printf("   Created: %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
				}
			} else if strings.HasPrefix(keyDir, "x25519-") {
				fmt.Printf("   Type: x25519\n")
			} else {
				// Assume Solana (Base58 encoded public key)
				_, _, err := importMgr.GetSOLWallet(keyDir)
				if err == nil {
					fmt.Printf("   Type: solana\n")
				}
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(keyCmd)
	keyCmd.AddCommand(keyGenerateCmd)
	keyCmd.AddCommand(keyShowCmd)
	keyCmd.AddCommand(keyListCmd)

	// Define flags for key generate
	keyGenerateCmd.Flags().StringVarP(&keyType, "type", "t", "", "Key type (ethereum|x25519)")
	keyGenerateCmd.Flags().StringVarP(&keyPassword, "password", "p", "", "Encryption password (required for ethereum)")
	keyGenerateCmd.Flags().BoolVarP(&keyPromptPass, "prompt-password", "P", false, "Interactive password input")

	// Define flags for key show
	keyShowCmd.Flags().StringVarP(&keyDidParam, "address", "a", "", "Address or public key to show")
}
