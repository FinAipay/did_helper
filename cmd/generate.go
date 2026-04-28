package cmd

// var (
// 	generatePassword   string
// 	generatePromptPass bool
// 	generateEntityType string
// 	generateEntityId   string
// )

// // generateCmd is the quick generation command (two-step process)
// var generateCmd = &cobra.Command{
// 	Use:   "generate",
// 	Short: "Quickly generate a key and create DID",
// 	Long: `Quickly generate a new key pair and create DID document in one step.
// This is a convenience command that combines 'key generate' and 'did create'.

// Example:
//   ./did_helper generate --entity-type agents --password "mypass"`,
// 	RunE: func(cmd *cobra.Command, args []string) error {
// 		// Default to ethereum type
// 		keyType := "ethereum"

// 		// Default to agents if not specified
// 		if generateEntityType == "" {
// 			generateEntityType = "agents"
// 		}

// 		fmt.Printf("Step 1: Generating %s key pair...\n", keyType)

// 		// Step 1: Generate key to import directory
// 		var publicKey string
// 		var err error
// 		var importKeyType string // Key type for import directory storage

// 		switch keyType {
// 		case "ethereum", "eth":
// 			publicKey, err = generateQuickETHKey(generatePassword)
// 			importKeyType = "eth"
// 		case "x25519":
// 			publicKey, err = generateQuickX25519Key()
// 			importKeyType = "x25519"
// 		default:
// 			return fmt.Errorf("unsupported key type: %s", keyType)
// 		}

// 		if err != nil {
// 			return err
// 		}

// 		// Step 2: Create DID document
// 		fmt.Println("\nStep 2: Creating DID document...")

// 		sm, err := storage.NewDIDStorageManager()
// 		if err != nil {
// 			return err
// 		}

// 		entityId := publicKey
// 		if generateEntityId != "" {
// 			entityId = generateEntityId
// 		}

// 		did, err := sm.CreateDIDDocument(generateEntityType, entityId, publicKey, importKeyType)
// 		if err != nil {
// 			return fmt.Errorf("failed to create DID: %w", err)
// 		}

// 		// Create template files
// 		emptyAPIKey := &models.APIKeyData{}
// 		sm.SaveAPIKey(did, emptyAPIKey)

// 		emptyTicket := &models.TicketData{}
// 		sm.SaveTicket(did, emptyTicket)

// 		// Display success
// 		fmt.Println("\n✓ Generation completed successfully!")
// 		fmt.Printf("========================\n")
// 		fmt.Printf("DID: %s\n", did)
// 		fmt.Printf("Type: %s\n", keyType)
// 		fmt.Printf("Entity Type: %s\n", generateEntityType)
// 		fmt.Printf("Entity ID: %s\n", entityId)

// 		if keyType == "ethereum" {
// 			fmt.Printf("Address: %s\n", publicKey)
// 		} else {
// 			fmt.Printf("Public Key: %s\n", publicKey)
// 		}

// 		fmt.Println("\n📁 Storage locations:")
// 		fmt.Printf("   Keys: ~/.did_helper/import/eth-%s/\n", publicKey)
// 		fmt.Printf("   DID:  ~/.did_helper/%s/\n", didToDirName(did))

// 		return nil
// 	},
// }

// // generateQuickETHKey generates ETH key and returns address
// func generateQuickETHKey(password string) (string, error) {
// 	importMgr, err := storage.NewImportStorageManager()
// 	if err != nil {
// 		return "", err
// 	}

// 	// Handle password
// 	if password == "" {
// 		fmt.Print("Enter encryption password: ")
// 		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
// 		if err != nil {
// 			return "", fmt.Errorf("failed to read password: %w", err)
// 		}
// 		fmt.Println()
// 		password = string(bytePassword)

// 		if password == "" {
// 			fmt.Println("⚠️  Warning: No password set. Key will have lower security!")
// 			fmt.Print("Continue? (y/N): ")
// 			reader := bufio.NewReader(os.Stdin)
// 			confirm, _ := reader.ReadString('\n')
// 			confirm = strings.TrimSpace(confirm)
// 			if confirm != "y" && confirm != "Y" {
// 				return "", fmt.Errorf("operation cancelled")
// 			}
// 		} else {
// 			// Validate password strength
// 			if err := wallet.ValidatePassword(password); err != nil {
// 				fmt.Printf("⚠️  Weak password: %v\n", err)
// 				fmt.Print("Continue with weak password? (y/N): ")
// 				reader := bufio.NewReader(os.Stdin)
// 				confirm, _ := reader.ReadString('\n')
// 				confirm = strings.TrimSpace(confirm)
// 				if confirm != "y" && confirm != "Y" {
// 					return "", fmt.Errorf("operation cancelled")
// 				}
// 			}

// 			// Confirm password
// 			fmt.Print("Confirm password: ")
// 			byteConfirmPassword, err := term.ReadPassword(int(syscall.Stdin))
// 			if err != nil {
// 				return "", fmt.Errorf("failed to read confirmation password: %w", err)
// 			}
// 			fmt.Println()

// 			if password != string(byteConfirmPassword) {
// 				return "", fmt.Errorf("passwords do not match")
// 			}
// 		}
// 	}

// 	keystoreData, metadata, err := wallet.GenerateETHWalletWithKeystore(password)
// 	if err != nil {
// 		return "", err
// 	}

// 	address := metadata.Address

// 	if err := importMgr.SaveETHWallet(address, keystoreData, metadata, password); err != nil {
// 		return "", err
// 	}

// 	return address, nil
// }

// // generateQuickX25519Key generates X25519 key and returns public key
// func generateQuickX25519Key() (string, error) {
// 	importMgr, err := storage.NewImportStorageManager()
// 	if err != nil {
// 		return "", err
// 	}

// 	publicKey, privateKey, err := wallet.GenerateX25519KeyPair()
// 	if err != nil {
// 		return "", err
// 	}

// 	if err := importMgr.SaveX25519Key(publicKey, privateKey, "", ""); err != nil {
// 		return "", err
// 	}

// 	return publicKey, nil
// }

// // func init() {
// // 	rootCmd.AddCommand(generateCmd)

// // 	// Define flags
// // 	generateCmd.Flags().StringVarP(&generatePassword, "password", "p", "", "Encryption password")
// // 	generateCmd.Flags().BoolVarP(&generatePromptPass, "prompt-password", "P", false, "Interactive password input")
// // 	generateCmd.Flags().StringVarP(&generateEntityType, "entity-type", "e", "agents", "Entity type (users|agents|devices|services|orgs|assets)")
// // 	generateCmd.Flags().StringVarP(&generateEntityId, "entity-id", "i", "", "Entity ID (optional, defaults to address)")
// // }
