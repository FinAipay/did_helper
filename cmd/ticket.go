package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"did_helper/internal/client"
	"did_helper/internal/models"
	"did_helper/internal/storage"
	"did_helper/internal/wallet"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	ticketDidParam      string
	challenge           string
	signature           string
	ticketChallengeOnly bool
	output              string
)

// ticketCmd is the Ticket management command
var ticketCmd = &cobra.Command{
	Use:   "ticket",
	Short: "Manage ticket credentials",
	Long:  "View and manage ticket credentials with DID-based storage",
}

// ticketShowCmd shows details of a specific ticket
var ticketShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show ticket information",
	RunE: func(cmd *cobra.Command, args []string) error {
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, err := sm.ResolveDID(ticketDidParam)
		if err != nil {
			return fmt.Errorf("failed to resolve DID: %w", err)
		}

		ticket, err := sm.GetTicket(did)
		if err != nil {
			return fmt.Errorf("ticket not found for DID: %s", did)
		}

		fmt.Printf("Ticket Information:\n")
		fmt.Printf("====================\n")
		fmt.Printf("DID: %s\n", did)
		fmt.Printf("Ticket: %s\n", "existence")
		fmt.Printf("Expires: %s\n", ticket.Expire)

		return nil
	},
}

var ticketCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new ticket (deprecated, use 'challenge' and 'verify' instead)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("this command is deprecated. Please use 'ticket challenge' and 'ticket verify' instead")
	},
}

// ticketChallengeCmd generates a challenge and signs it
var ticketChallengeCmd = &cobra.Command{
	Use:   "challenge",
	Short: "Generate challenge and sign it",
	Long: `Generate a challenge from the DID service and sign it.

If the private key exists in the import directory, it will be signed automatically.
Otherwise, JavaScript code will be output for manual signing with browser wallet.

Example:
  ./did_helper ticket challenge --did did:finai:users:0x1234...
  ./did_helper ticket challenge --did did:finai:users:0x1234... --output challenge.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, err := sm.ResolveDID(ticketDidParam)
		if err != nil {
			return fmt.Errorf("failed to resolve DID: %w", err)
		}

		importMgr, err := storage.NewImportStorageManager()
		if err != nil {
			return err
		}

		// Extract entityId from DID (format: did:finai:{type}:{entityId})
		parts := strings.Split(did, ":")
		if len(parts) != 4 {
			return fmt.Errorf("invalid DID format: %s", did)
		}
		entityId := strings.ToLower(parts[3])

		fmt.Printf("🔐 Generating challenge for DID: %s\n", did)
		fmt.Printf("📋 Entity ID: %s\n\n", entityId)

		// Step 1: Generate challenge
		config, err := sm.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.Post(fmt.Sprintf("%s/api/v1/challenge/%s", config.API, did), nil)
		err = httpClient.VerifyResponse(statusCode, res, err)
		if err != nil {
			return err
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse challenge response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("challenge generation failed: %s", apiResp.Message)
		}

		// Parse challenge data
		challengeData := &models.ChallengeResponse{}
		dataBytes, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(dataBytes, challengeData); err != nil {
			return fmt.Errorf("failed to parse challenge data: %w", err)
		}

		fmt.Printf("✅ Challenge generated\n")
		fmt.Printf("📝 Challenge: %s\n", challengeData.Challenge)
		fmt.Printf("⏰ Expires at: %s\n\n", challengeData.ExpiresAt)

		// Step 2: Check if private key exists in import directory
		if importMgr.KeyExistsInImport(entityId) {
			fmt.Println("🔑 Private key found in import directory, signing automatically...")

			// Try to sign with ETH key first
			sig, err := signChallengeWithETHKey(challengeData.Challenge, entityId)
			if err == nil {
				signature = sig
				fmt.Println("✅ Signed with ETH key")
			} else {
				// Try Solana key
				fmt.Println("ℹ️  ETH key not available, trying Solana...")
				sig, err = signChallengeWithSOLKey(challengeData.Challenge, entityId)
				if err == nil {
					signature = sig
					fmt.Println("✅ Signed with Solana key")
				} else {
					// Try X25519 decryption
					fmt.Println("ℹ️  Solana key not available, trying X25519...")
					sig, err = decryptChallengeWithX25519(challengeData.Challenge, entityId)
					if err != nil {
						return fmt.Errorf("failed to sign/decrypt challenge: %w", err)
					}
					signature = sig
					fmt.Println("✅ Decrypted with X25519 key")
				}
			}

			fmt.Printf("\n🔐 Signature: %s\n", signature)
			fmt.Println("\n💡 You can now use 'ticket verify' command to verify this signature:")
			fmt.Printf("   ./did_helper ticket verify --did %s --challenge '%s' --signature '%s'\n",
				did, challengeData.Challenge, signature)

		} else {
			// No private key found, generate HTML file for manual signing
			fmt.Println("💡 Instructions:")
			fmt.Printf("⏰ **1. Quick verification of browser wallet(recommend):** %s", challengeData.BrowserExtension)
			fmt.Println("	Open this link in browser for quick verification")

			fmt.Println("⚠️  Private key NOT found in import directory.")
			fmt.Println("📋  Generating HTML file for browser wallet signing...\n")

			jsCode := generateSigningJS(did, challengeData.Challenge)
			if err != nil {
				return fmt.Errorf("failed to generate HTML file: %w", err)
			}

			fmt.Printf("✅ JS Code generated successfully!\n")
			fmt.Println("2. Manual verification")
			fmt.Printf("📁 %s\n\n", jsCode)
			fmt.Println("Step:")
			fmt.Println("   1. Open the browser console(F12))")
			fmt.Println("   2. Paste the above code into the console and press Enter to run it")
			fmt.Println("   3. Paste the `Signature` into the agent chat box")
			fmt.Printf("   ./did_helper ticket verify --did %s --challenge '%s' --signature '<YOUR_SIGNATURE>'\n",
				did, challengeData.Challenge)
		}

		return nil
	},
}

// ticketVerifyCmd verifies the signature and obtains ticket
var ticketVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify signature and obtain ticket",
	Long: `Verify the challenge signature and obtain a ticket.

Example:
  ./did_helper ticket verify --did did:finai:users:0x1234... --challenge <challenge> --signature <signature>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if ticketDidParam == "" || challenge == "" || signature == "" {
			return fmt.Errorf("--did, --challenge, and --signature are all required")
		}

		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, err := sm.ResolveDID(ticketDidParam)
		if err != nil {
			return fmt.Errorf("failed to resolve DID: %w", err)
		}

		fmt.Printf("📤 Verifying signature...\n")
		fmt.Printf("📋 DID: %s\n", did)
		fmt.Printf("🔐 Challenge: %s...\n", truncateString(challenge, 30))
		fmt.Printf("✍️  Signature: %s...\n\n", truncateString(signature, 30))

		// Call verify API
		config, err := sm.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Try nested format: { "data": { "did": ..., "challenge": ..., "signature": ... } }
		verifyReq := map[string]interface{}{
			"did":       did,
			"challenge": challenge,
			"signature": signature,
		}

		reqBody, _ := json.Marshal(verifyReq)

		// Debug: print request body
		fmt.Printf("📤 Request Body: %s\n", string(reqBody))
		fmt.Printf("📍 API Endpoint: %s/api/v1/verify/challenge\n\n", config.API)

		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.Post(config.API+"/api/v1/verify/challenge", verifyReq)

		// Debug: print response
		fmt.Printf("📥 Response Status: %d\n", statusCode)
		fmt.Printf("📥 Response Body: %s\n\n", res)

		if err != nil || statusCode != 200 {
			return fmt.Errorf("verification failed (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse verification response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("verification failed: %s", apiResp.Message)
		}

		// Parse ticket data
		ticketData := &models.VerifyChallengeResponse{}
		dataBytes, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(dataBytes, ticketData); err != nil {
			return fmt.Errorf("failed to parse ticket data: %w", err)
		}

		fmt.Println("✅ Verification successful!")
		// Calculate expire time
		expire := ticketData.Expire
		if expire == "" {
			// Default to 6 months if not provided
			expire = calculateDefaultExpire()
			fmt.Printf("⏰ Expire: %s (default 6 months)\n", expire)
		} else {
			fmt.Printf("⏰ Expire: %s\n", expire)
		}

		// Save ticket to storage
		if err := saveTicketToStorage(sm, did, ticketData.Ticket, expire); err != nil {
			return fmt.Errorf("failed to save ticket: %w", err)
		}

		fmt.Println("\n✓ You can now use this ticket to access services!")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(ticketCmd)
	ticketCmd.AddCommand(ticketShowCmd)
	// ticketCmd.AddCommand(ticketCreateCmd)
	ticketCmd.AddCommand(ticketChallengeCmd)
	ticketCmd.AddCommand(ticketVerifyCmd)

	ticketShowCmd.Flags().StringVarP(&ticketDidParam, "did", "d", "", "DID identifier (default: from config)")
	ticketChallengeCmd.Flags().StringVarP(&ticketDidParam, "did", "d", "", "DID identifier (default: from config)")
	ticketChallengeCmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (optional)")
	ticketVerifyCmd.Flags().StringVarP(&ticketDidParam, "did", "d", "", "DID identifier (default: from config)")
	ticketVerifyCmd.Flags().StringVarP(&challenge, "challenge", "c", "", "Challenge string")
	ticketVerifyCmd.Flags().StringVarP(&signature, "signature", "s", "", "Signature string")
}

// ============================================================================
// Helper Functions
// ============================================================================

func signChallengeWithETHKey(challenge, entityId string) (string, error) {
	importMgr, _ := storage.NewImportStorageManager()

	// Try multiple methods to get private key

	// Method 1: Try to load directly encrypted private key (if exists)
	privateKeyHex, err := loadEncryptedPrivateKey(importMgr, entityId)
	if err == nil {
		fmt.Println("✅ Loaded private key from encrypted file")
	} else {
		// Method 2: Load keystore and decrypt with password
		fmt.Println("ℹ️  Encrypted private key not found, trying keystore...")
		privateKeyHex, err = loadAndDecryptKeystore(importMgr, entityId)
		if err != nil {
			return "", fmt.Errorf("failed to get private key: %w", err)
		}
	}

	// Sign challenge using personal_sign
	signature, err := wallet.PersonalSign(privateKeyHex, challenge)
	if err != nil {
		return "", fmt.Errorf("failed to sign challenge: %w", err)
	}

	return signature, nil
}

// loadEncryptedPrivateKey tries to load directly encrypted private key
func loadEncryptedPrivateKey(importMgr *storage.ImportStorageManager, entityId string) (string, error) {
	// Check if private_key.enc exists
	privKeyPath := filepath.Join(importMgr.GetImportPath(entityId), "private_key.enc")
	if _, err := os.Stat(privKeyPath); os.IsNotExist(err) {
		return "", fmt.Errorf("encrypted private key file not found")
	}

	// Read encrypted private key
	encData, err := os.ReadFile(privKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read encrypted private key: %w", err)
	}

	// Try to decrypt without password (if stored unencrypted)
	// Or use password from password.txt
	password := ""
	passPath := importMgr.GetPasswordPath(entityId)
	if passData, err := os.ReadFile(passPath); err == nil {
		password = strings.TrimSpace(string(passData))
	}

	// Decrypt the private key
	var privateKeyHex string
	if password != "" {
		decrypted, err := wallet.DecryptData(string(encData), password)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt private key: %w", err)
		}
		privateKeyHex = string(decrypted)
	} else {
		// Assume it's stored as plain hex (less secure but convenient)
		privateKeyHex = string(encData)
	}

	return privateKeyHex, nil
}

// loadAndDecryptKeystore loads keystore and decrypts with password
func loadAndDecryptKeystore(importMgr *storage.ImportStorageManager, entityId string) (string, error) {
	// Load keystore as raw JSON to avoid type conversion issues
	importPath := importMgr.GetImportPath(entityId)
	ksPath := filepath.Join(importPath, "keystore.json")
	keystoreJSON, err := os.ReadFile(ksPath)
	if err != nil {
		return "", fmt.Errorf("failed to read keystore file: %w", err)
	}

	// Read password if exists
	password := ""
	passPath := importMgr.GetPasswordPath(entityId)
	if passData, err := os.ReadFile(passPath); err == nil {
		password = strings.TrimSpace(string(passData))
		fmt.Println("✅ Found password in password.txt")
	}

	if password == "" {
		// Prompt for password
		fmt.Print("⚠️  Password not found. Enter keystore password: ")
		passBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		password = string(passBytes)
		fmt.Println()
	}

	// Decrypt private key from raw keystore JSON
	privateKeyHex, err := wallet.DecryptKeystoreFromRawJSON(keystoreJSON, password)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt keystore: %w", err)
	}

	return privateKeyHex, nil
}

func decryptChallengeWithX25519(challenge, entityId string) (string, error) {
	importMgr, _ := storage.NewImportStorageManager()

	// Load X25519 keypair
	pubKey, privKey, err := importMgr.GetX25519Key(entityId)
	if err != nil {
		return "", fmt.Errorf("failed to load X25519 key: %w", err)
	}
	_ = pubKey

	// For X25519, we need to decrypt the challenge
	// The challenge should be encrypted with our public key
	challengeBytes, err := hex.DecodeString(challenge)
	if err != nil {
		return "", fmt.Errorf("failed to decode challenge: %w", err)
	}

	// Decrypt using X25519
	decrypted, err := wallet.X25519Decrypt(privKey, challengeBytes)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt challenge: %w", err)
	}

	return hex.EncodeToString(decrypted), nil
}

// signChallengeWithSOLKey signs challenge with Solana Ed25519 private key
func signChallengeWithSOLKey(challenge, entityId string) (string, error) {
	importMgr, _ := storage.NewImportStorageManager()

	// Load Solana keypair
	pubKey, privKey, err := importMgr.GetSOLWallet(entityId)
	if err != nil {
		return "", fmt.Errorf("failed to load Solana key: %w", err)
	}
	_ = pubKey

	// Sign challenge using Solana Ed25519
	signature, err := wallet.SignMessageWithSOL(privKey, challenge)
	if err != nil {
		return "", fmt.Errorf("failed to sign challenge: %w", err)
	}

	return signature, nil
}

func generateSigningJS(did, challenge string) string {
	return fmt.Sprintf(`async function verifyDID() {
    // Try MetaMask (Ethereum) first
    if (typeof ethereum !== 'undefined') {
        try {
            console.log('🦊 Detecting MetaMask...');
            // Request connection to MetaMask
            const accounts = await ethereum.request({ 
                method: 'eth_requestAccounts' 
            });
            
            console.log('✅ Connected account:', accounts[0]);
            console.log('📝 Please confirm this is your address: %s');
            
            // Challenge content
            const challenge = "%s";
            
            // Sign challenge with MetaMask using personal_sign
            const signature = await ethereum.request({
                method: 'personal_sign',
                params: [challenge, accounts[0]],
            });
            
            console.log('\\n🔐 Signature result:');
            console.log('Signature:', signature);
            console.log('\\n📋 Challenge ID:');
            console.log(challenge);
            
            console.log('\\n⚠️  Please copy the "Signature" value above and provide it to the assistant for verification!');
            
            return signature;
            
        } catch (error) {
            console.error('❌ MetaMask signing failed:', error);
        }
    }
    
    // Try Phantom (Solana)
    if (typeof solana !== 'undefined' && solana.isPhantom) {
        try {
            console.log('👻 Detecting Phantom Wallet...');
            // Connect to Phantom
            const resp = await solana.connect();
            const publicKey = resp.publicKey.toString();
            
            console.log('✅ Connected account:', publicKey);
            console.log('📝 Please confirm this is your address: %s');
            
            // Challenge content
            const challenge = "%s";
            const message = new TextEncoder().encode(challenge);
            
            // Sign message with Phantom
            const signedMessage = await solana.signMessage(message, "utf8");
            
            // Convert signature to hex string with 0x prefix
            const signature = '0x' + Array.from(signedMessage.signature)
                .map(b => b.toString(16).padStart(2, '0'))
                .join('');
            
            console.log('\\n🔐 Signature result:');
            console.log('Signature:', signature);
            console.log('\\n📋 Challenge ID:');
            console.log(challenge);
            
            console.log('\\n⚠️  Please copy the "Signature" value above and provide it to the assistant for verification!');
            
            return signature;
            
        } catch (error) {
            console.error('❌ Phantom signing failed:', error);
        }
    }
    
    // No wallet detected
    console.error('❌ No compatible wallet detected!');
    console.error('Please install MetaMask (for Ethereum) or Phantom (for Solana) extension!');
}
verifyDID();`, did, challenge, did, challenge)
}

func saveTicketToStorage(sm *storage.DIDStorageManager, did, ticket, expire string) error {
	// Use TicketData instead of StoredTicket
	ticketData := &models.TicketData{
		Ticket: ticket,
		Expire: expire,
	}

	return sm.SaveTicket(did, ticketData)
}

func calculateDefaultExpire() string {
	// Default to 6 months from now
	return time.Now().Add(time.Hour * 24 * 30 * 6).Format(time.RFC3339)
}

func currentTimeRFC3339() string {
	return time.Now().Format(time.RFC3339)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func didToDirName(did string) string {
	// Simple conversion for directory name, replacing colons with underscores or similar
	return strings.ReplaceAll(did, ":", "_")
}
