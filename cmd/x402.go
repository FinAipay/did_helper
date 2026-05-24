package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"did_helper/internal/client"
	"did_helper/internal/models"
	"did_helper/internal/storage"
	"did_helper/internal/wallet"

	"github.com/spf13/cobra"
)

var (
	x402DID           string
	x402Amount        string
	x402Recipient     string
	x402OrderID       string
	x402Status        string
	x402Limit         int
	x402Network       string
	x402Force         bool
)

// x402Cmd is the x402 payment command
var x402Cmd = &cobra.Command{
	Use:   "x402",
	Short: "Manage x402 payments and orders",
	Long:  "Create payments, manage orders, and process x402 transactions with EIP-712 signatures",
}

// validateX402Prerequisites validates DID and ETH wallet existence
func validateX402Prerequisites(sm *storage.DIDStorageManager, did string) error {
	// 1. Validate DID exists
	resolvedDID, err := sm.ResolveDID(did)
	if err != nil {
		return fmt.Errorf("DID not found: %w. Please create DID first", err)
	}

	// 2. Extract entityId from DID
	parts := strings.Split(resolvedDID, ":")
	if len(parts) != 4 {
		return fmt.Errorf("invalid DID format: %s", resolvedDID)
	}
	entityId := strings.ToLower(parts[3])

	// 3. Check ETH wallet exists in import directory
	importMgr, err := storage.NewImportStorageManager()
	if err != nil {
		return fmt.Errorf("failed to initialize import manager: %w", err)
	}

	if !importMgr.KeyExistsInImport(entityId) {
		return fmt.Errorf("ETH wallet not found for entity: %s. Please generate ETH key first:\n  did_helper key generate --type ethereum --password <password>", entityId)
	}

	// 4. Verify it's ETH type (not Solana or X25519)
	_, _, err = importMgr.GetETHWallet(entityId)
	if err != nil {
		return fmt.Errorf("only ETH wallets are supported for x402 payments. Found non-ETH key for: %s", entityId)
	}

	return nil
}

// loadX402Config loads x402 configuration
func loadX402Config(sm *storage.DIDStorageManager) (*models.Config, error) {
	config, err := sm.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate x402 API is configured
	if config.X402API == "" {
		return nil, fmt.Errorf("x402_api not configured in config.json")
	}

	return config, nil
}

// checkPaymentConfirmation checks if payment requires confirmation
func checkPaymentConfirmation(config *models.Config, amount string, orderID string) error {
	if !config.PaymentConfirmation.Enabled {
		return nil
	}

	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return fmt.Errorf("invalid amount format: %w", err)
	}

	thresholdFloat, err := strconv.ParseFloat(config.PaymentConfirmation.USDCThreshold, 64)
	if err != nil {
		return fmt.Errorf("invalid threshold format in config: %w", err)
	}

	if amountFloat > thresholdFloat {
		fmt.Println("\n⚠️  ⚠️  ⚠️  HIGH VALUE PAYMENT WARNING ⚠️  ⚠️  ⚠️")
		fmt.Printf("Amount: %s USDC (threshold: %s)\n", amount, config.PaymentConfirmation.USDCThreshold)
		fmt.Println("\nThis payment exceeds the confirmation threshold.")
		fmt.Printf("Type EXACTLY: \"CONFIRM PAYMENT %s\" to proceed\n", orderID)
		fmt.Println("Type 'cancel' to abort.\n")

		var confirmation string
		fmt.Scanln(&confirmation)

		expectedConfirm := fmt.Sprintf("CONFIRM PAYMENT %s", orderID)
		if strings.ToUpper(confirmation) != expectedConfirm {
			return fmt.Errorf("payment cancelled by user")
		}

		fmt.Println("✅ Payment confirmed by user")
	}

	return nil
}

// x402PaymentCreateCmd creates a new payment order
var x402PaymentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new payment order",
	Long: `Create a new x402 payment order.

Example:
  ./did_helper x402 payment create --did did:finai:users:0x123... --amount "1.00" --recipient "0x742d..."`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if x402DID == "" {
			return fmt.Errorf("--did is required")
		}
		if x402Amount == "" {
			return fmt.Errorf("--amount is required")
		}
		if x402Recipient == "" {
			return fmt.Errorf("--recipient is required")
		}

		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		// Validate prerequisites
		if err := validateX402Prerequisites(sm, x402DID); err != nil {
			return err
		}

		// Load config
		config, err := loadX402Config(sm)
		if err != nil {
			return err
		}

		// Load ticket
		did, _ := sm.ResolveDID(x402DID)
		ticket, err := sm.LoadTicket(did)
		if err != nil {
			return fmt.Errorf("ticket not found or failed to load: %w\n💡 Please run 'did_helper ticket challenge' and 'did_helper ticket verify' first", err)
		}

		fmt.Printf("📝 Creating payment order...\n")
		fmt.Printf("Amount: %s USDC\n", x402Amount)
		fmt.Printf("Recipient: %s\n", x402Recipient)
		fmt.Println()

		// Create payment request
		paymentReq := models.PaymentRequest{
			Amount:           x402Amount,
			RecipientAddress: x402Recipient,
		}

		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.PostWithTicketAndDID(
			config.X402API+"/api/x402WebClient/v1/create-custom-payment",
			paymentReq,
			ticket.Ticket,
			did,
		)
		if err != nil {
			return fmt.Errorf("failed to create payment: %w", err)
		}

		if statusCode != 200 && statusCode != 201 {
			return fmt.Errorf("failed to create payment (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("creation failed: %s", apiResp.Message)
		}

		// Parse order data
		var order models.OrderResponse
		dataBytes, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(dataBytes, &order); err != nil {
			return fmt.Errorf("failed to parse order data: %w", err)
		}

		// Save order to local storage
		orderMgr, err := storage.NewOrderStorageManager()
		if err != nil {
			return fmt.Errorf("failed to initialize order storage: %w", err)
		}

		if err := orderMgr.SaveOrder(did, &order); err != nil {
			fmt.Printf("⚠️  Warning: Failed to save order to local storage: %v\n", err)
		} else {
			fmt.Println("💾 Order saved to local storage")
		}

		// Display results
		fmt.Println("\n✅ Payment order created successfully!")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("Order ID: %s\n", order.OrderID)
		fmt.Printf("Status: %s\n", order.Status)
		fmt.Printf("Amount: %s USDC\n", order.Amount)
		fmt.Printf("Recipient: %s\n", order.RecipientAddress)
		fmt.Printf("Network: %s\n", order.Network)
		fmt.Printf("Created: %s\n", order.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("\n💡 Next steps:")
		fmt.Printf("   - Process payment: did_helper x402 pay --did %s --order-id %s\n", did, order.OrderID)
		fmt.Printf("   - Check status: did_helper x402 order status --did %s --order-id %s\n", did, order.OrderID)

		return nil
	},
}

// x402OrderListCmd lists orders
var x402OrderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List payment orders",
	Long: `List payment orders for a DID.

Example:
  ./did_helper x402 order list --did did:finai:users:0x123...
  ./did_helper x402 order list --did did:finai:users:0x123... --status pending --limit 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if x402DID == "" {
			return fmt.Errorf("--did is required")
		}

		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		// Validate prerequisites
		if err := validateX402Prerequisites(sm, x402DID); err != nil {
			return err
		}

		did, _ := sm.ResolveDID(x402DID)

		// Load ticket
		ticket, err := sm.LoadTicket(did)
		if err != nil {
			return fmt.Errorf("ticket not found or failed to load: %w\n💡 Please run 'did_helper ticket challenge' and 'did_helper ticket verify' first", err)
		}

		config, err := loadX402Config(sm)
		if err != nil {
			return err
		}

		// Build query URL
		url := fmt.Sprintf("%s/api/x402WebClient/v1/orders", config.X402API)
		params := []string{}
		if x402Status != "" {
			params = append(params, fmt.Sprintf("status=%s", x402Status))
		}
		if x402Limit > 0 {
			params = append(params, fmt.Sprintf("limit=%d", x402Limit))
		}
		if len(params) > 0 {
			url += "?" + strings.Join(params, "&")
		}

		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.GetWithTicketAndDID(url, ticket.Ticket, did)
		if err != nil {
			return fmt.Errorf("failed to list orders: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("failed to list orders (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("query failed: %s", apiResp.Message)
		}

		// Parse orders
		var orderList models.OrderListResponse
		dataBytes, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(dataBytes, &orderList); err != nil {
			return fmt.Errorf("failed to parse orders: %w", err)
		}

		if len(orderList.Orders) == 0 {
			fmt.Println("No orders found")
			return nil
		}

		fmt.Printf("📋 Orders for DID: %s\n", did)
		fmt.Printf("Found %d order(s):\n\n", len(orderList.Orders))

		for i, order := range orderList.Orders {
			fmt.Printf("%d. Order ID: %s\n", i+1, order.OrderID)
			fmt.Printf("   Status: %s\n", order.Status)
			fmt.Printf("   Amount: %s USDC\n", order.Amount)
			fmt.Printf("   Recipient: %s\n", order.RecipientAddress)
			fmt.Printf("   Network: %s\n", order.Network)
			fmt.Printf("   Created: %s\n", order.CreatedAt.Format("2006-01-02 15:04:05"))
			if order.TxHash != "" {
				fmt.Printf("   Tx Hash: %s\n", order.TxHash)
			}
			fmt.Println()
		}

		// Update local storage
		orderMgr, _ := storage.NewOrderStorageManager()
		for _, order := range orderList.Orders {
			orderMgr.SaveOrder(did, &order)
		}

		return nil
	},
}

// x402OrderShowCmd shows order details
var x402OrderShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show order details",
	Long: `Show detailed information about a specific order.

Example:
  ./did_helper x402 order show --did did:finai:users:0x123... --order-id <ORDER_ID>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if x402DID == "" || x402OrderID == "" {
			return fmt.Errorf("--did and --order-id are required")
		}

		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		// Validate prerequisites
		if err := validateX402Prerequisites(sm, x402DID); err != nil {
			return err
		}

		did, _ := sm.ResolveDID(x402DID)

		// Try to load from local storage first
		orderMgr, err := storage.NewOrderStorageManager()
		if err == nil {
			order, err := orderMgr.GetOrder(did, x402OrderID)
			if err == nil {
				displayOrderDetails(order)
				return nil
			}
		}

		// If not in local storage, fetch from API
		ticket, err := sm.LoadTicket(did)
		if err != nil {
			return fmt.Errorf("ticket not found or failed to load: %w", err)
		}

		config, err := loadX402Config(sm)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/api/x402WebClient/v1/orders/%s", config.X402API, x402OrderID)
		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.GetWithTicketAndDID(url, ticket.Ticket, did)
		if err != nil {
			return fmt.Errorf("failed to get order: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("failed to get order (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("query failed: %s", apiResp.Message)
		}

		var order models.OrderResponse
		dataBytes, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(dataBytes, &order); err != nil {
			return fmt.Errorf("failed to parse order: %w", err)
		}

		// Save to local storage
		if orderMgr != nil {
			orderMgr.SaveOrder(did, &order)
		}

		displayOrderDetails(&order)
		return nil
	},
}

func displayOrderDetails(order *models.OrderResponse) {
	fmt.Println("📋 Order Details")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Order ID: %s\n", order.OrderID)
	fmt.Printf("Status: %s\n", order.Status)
	fmt.Printf("Amount: %s USDC\n", order.Amount)
	fmt.Printf("Recipient: %s\n", order.RecipientAddress)
	if order.PayerAddress != "" {
		fmt.Printf("Payer: %s\n", order.PayerAddress)
	}
	fmt.Printf("Network: %s\n", order.Network)
	if order.TxHash != "" {
		fmt.Printf("Transaction Hash: %s\n", order.TxHash)
	}
	fmt.Printf("Created: %s\n", order.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", order.UpdatedAt.Format("2006-01-02 15:04:05"))
	if !order.ExpiresAt.IsZero() {
		fmt.Printf("Expires: %s\n", order.ExpiresAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println(strings.Repeat("=", 60))
}

// x402PayCmd processes payment with EIP-712 signature
var x402PayCmd = &cobra.Command{
	Use:   "pay",
	Short: "Process payment with EIP-712 signature",
	Long: `Process a payment by signing the order with EIP-712 and submitting to the network.

Example:
  ./did_helper x402 pay --did did:finai:users:0x123... --order-id <ORDER_ID>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if x402DID == "" || x402OrderID == "" {
			return fmt.Errorf("--did and --order-id are required")
		}

		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		// Validate prerequisites
		if err := validateX402Prerequisites(sm, x402DID); err != nil {
			return err
		}

		did, _ := sm.ResolveDID(x402DID)

		// Load config
		config, err := loadX402Config(sm)
		if err != nil {
			return err
		}

		// Load ticket
		ticket, err := sm.LoadTicket(did)
		if err != nil {
			return fmt.Errorf("ticket not found or failed to load: %w\n💡 Please run 'did_helper ticket challenge' and 'did_helper ticket verify' first", err)
		}

		// Get order details (from local storage or API)
		orderMgr, _ := storage.NewOrderStorageManager()
		var order *models.OrderResponse
		
		// Try local first
		if orderMgr != nil {
			order, err = orderMgr.GetOrder(did, x402OrderID)
		}
		
		// If not in local, fetch from API
		if order == nil {
			url := fmt.Sprintf("%s/api/x402WebClient/v1/orders/%s", config.X402API, x402OrderID)
			httpClient := client.NewHTTPClient()
			statusCode, res, err := httpClient.GetWithTicketAndDID(url, ticket.Ticket, did)
			if err != nil {
				return fmt.Errorf("failed to get order: %w", err)
			}

			if statusCode != 200 {
				return fmt.Errorf("failed to get order (status: %d): %s", statusCode, res)
			}

			var apiResp models.APIResponse
			if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if !apiResp.Success {
				return fmt.Errorf("query failed: %s", apiResp.Message)
			}

			order = &models.OrderResponse{}
			dataBytes, _ := json.Marshal(apiResp.Data)
			if err := json.Unmarshal(dataBytes, order); err != nil {
				return fmt.Errorf("failed to parse order: %w", err)
			}
		}

		// Check payment confirmation threshold
		if err := checkPaymentConfirmation(config, order.Amount, order.OrderID); err != nil {
			return err
		}

		fmt.Printf("🔐 Processing payment...\n")
		fmt.Printf("Order ID: %s\n", order.OrderID)
		fmt.Printf("Amount: %s USDC\n", order.Amount)
		fmt.Println()

		// Step 1: Get signing requirements
		fmt.Println("📋 Getting signing requirements...")
		signReqURL := fmt.Sprintf("%s/api/x402WebClient/v1/order/%s/signing-requirements", config.X402API, order.OrderID)
		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.GetWithTicketAndDID(signReqURL, ticket.Ticket, did)
		if err != nil {
			return fmt.Errorf("failed to get signing requirements: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("failed to get signing requirements (status: %d): %s", statusCode, res)
		}

		var signReqResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &signReqResp); err != nil {
			return fmt.Errorf("failed to parse signing requirements: %w", err)
		}

		if !signReqResp.Success {
			return fmt.Errorf("failed to get signing requirements: %s", signReqResp.Message)
		}

		var signRequirements models.SigningRequirements
		dataBytes, _ := json.Marshal(signReqResp.Data)
		if err := json.Unmarshal(dataBytes, &signRequirements); err != nil {
			return fmt.Errorf("failed to parse signing data: %w", err)
		}

		fmt.Println("✅ Signing requirements received")

		// Step 2: Load ETH private key
		fmt.Println("🔑 Loading ETH wallet...")
		importMgr, _ := storage.NewImportStorageManager()
		parts := strings.Split(did, ":")
		entityId := strings.ToLower(parts[3])

		// Load keystore and decrypt
		privateKeyHex, err := loadETHPrivateKeyForSigning(importMgr, entityId)
		if err != nil {
			return fmt.Errorf("failed to load private key: %w", err)
		}

		fmt.Println("✅ ETH wallet loaded")

		// Step 3: Sign with EIP-712
		fmt.Println("✍️  Signing with EIP-712...")
		signature, err := wallet.SignEIP712(
			privateKeyHex,
			signRequirements.Domain,
			signRequirements.Types,
			signRequirements.Message,
		)
		if err != nil {
			return fmt.Errorf("failed to sign with EIP-712: %w", err)
		}

		fmt.Printf("✅ Signature generated: %s...\n", signature[:20])

		// Step 4: Submit payment
		fmt.Println("\n📤 Submitting payment...")
		processReq := models.ProcessPaymentRequest{
			Address:   parts[3], // Use entity ID as address
			Signature: signature,
			Payload:   signRequirements,
			Network:   config.DefaultNetwork,
		}

		processURL := fmt.Sprintf("%s/api/x402WebClient/v1/order/%s/process-payment", config.X402API, order.OrderID)
		statusCode, res, err = httpClient.PostWithTicketAndDID(processURL, processReq, ticket.Ticket, did)
		if err != nil {
			return fmt.Errorf("failed to process payment: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("payment failed (status: %d): %s", statusCode, res)
		}

		var processResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &processResp); err != nil {
			return fmt.Errorf("failed to parse payment response: %w", err)
		}

		if !processResp.Success {
			return fmt.Errorf("payment failed: %s", processResp.Message)
		}

		// Parse result
		var txResult map[string]interface{}
		dataBytes, _ = json.Marshal(processResp.Data)
		json.Unmarshal(dataBytes, &txResult)

		txHash, _ := txResult["tx_hash"].(string)

		fmt.Println("\n✅ Payment processed successfully!")
		fmt.Println(strings.Repeat("=", 60))
		if txHash != "" {
			fmt.Printf("Transaction Hash: %s\n", txHash)
		}
		fmt.Printf("Message: %s\n", processResp.Message)
		fmt.Println(strings.Repeat("=", 60))

		// Update local order status
		if orderMgr != nil && txHash != "" {
			orderMgr.UpdateOrderStatus(did, order.OrderID, "paid", txHash)
		}

		fmt.Println("\n💡 You can check the transaction on the blockchain explorer")

		return nil
	},
}

// loadETHPrivateKeyForSigning loads and decrypts ETH private key for signing
func loadETHPrivateKeyForSigning(importMgr *storage.ImportStorageManager, entityId string) (string, error) {
	// Try method 1: Load keystore and decrypt with password
	importPath := importMgr.GetImportPath(entityId)
	ksPath := fmt.Sprintf("%s/keystore.json", importPath)
	
	keystoreJSON, err := os.ReadFile(ksPath)
	if err != nil {
		return "", fmt.Errorf("failed to read keystore file: %w", err)
	}

	// Read password if exists
	password := ""
	passPath := importMgr.GetPasswordPath(entityId)
	if passData, err := os.ReadFile(passPath); err == nil {
		password = strings.TrimSpace(string(passData))
	}

	if password == "" {
		return "", fmt.Errorf("password not found. Please ensure password.txt exists in the key directory")
	}

	// Decrypt private key from raw keystore JSON
	privateKeyHex, err := wallet.DecryptKeystoreFromRawJSON(keystoreJSON, password)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt keystore: %w", err)
	}

	return privateKeyHex, nil
}

// x402OrderStatusCmd checks order status
var x402OrderStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check order status",
	Long: `Check the current status of an order.

Example:
  ./did_helper x402 order status --did did:finai:users:0x123... --order-id <ORDER_ID>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if x402DID == "" || x402OrderID == "" {
			return fmt.Errorf("--did and --order-id are required")
		}

		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, _ := sm.ResolveDID(x402DID)

		// Load ticket
		ticket, err := sm.LoadTicket(did)
		if err != nil {
			return fmt.Errorf("ticket not found or failed to load: %w", err)
		}

		config, err := loadX402Config(sm)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/api/x402WebClient/v1/order/%s/status", config.X402API, x402OrderID)
		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.GetWithTicketAndDID(url, ticket.Ticket, did)
		if err != nil {
			return fmt.Errorf("failed to get order status: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("failed to get status (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("query failed: %s", apiResp.Message)
		}

		var status models.OrderStatus
		dataBytes, _ := json.Marshal(apiResp.Data)
		if err := json.Unmarshal(dataBytes, &status); err != nil {
			return fmt.Errorf("failed to parse status: %w", err)
		}

		fmt.Println("📊 Order Status")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("Order ID: %s\n", status.OrderID)
		fmt.Printf("Status: %s\n", status.Status)
		if status.TxHash != "" {
			fmt.Printf("Transaction Hash: %s\n", status.TxHash)
		}
		fmt.Printf("Updated: %s\n", status.UpdatedAt.Format("2006-01-02 15:04:05"))
		if status.Message != "" {
			fmt.Printf("Message: %s\n", status.Message)
		}
		fmt.Println(strings.Repeat("=", 60))

		return nil
	},
}

// x402OrderRetryCmd retries a failed order
var x402OrderRetryCmd = &cobra.Command{
	Use:   "retry",
	Short: "Retry a failed order",
	Long: `Retry a failed payment order.

Example:
  ./did_helper x402 order retry --did did:finai:users:0x123... --order-id <ORDER_ID>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if x402DID == "" || x402OrderID == "" {
			return fmt.Errorf("--did and --order-id are required")
		}

		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, _ := sm.ResolveDID(x402DID)

		// Load ticket
		ticket, err := sm.LoadTicket(did)
		if err != nil {
			return fmt.Errorf("ticket not found or failed to load: %w", err)
		}

		config, err := loadX402Config(sm)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/api/x402WebClient/v1/orders/%s/retry", config.X402API, x402OrderID)
		
		retryReq := map[string]interface{}{
			"reason": "User requested retry",
		}

		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.PostWithTicketAndDID(url, retryReq, ticket.Ticket, did)
		if err != nil {
			return fmt.Errorf("failed to retry order: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("retry failed (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("retry failed: %s", apiResp.Message)
		}

		fmt.Println("✅ Order retry initiated successfully!")
		fmt.Printf("Message: %s\n", apiResp.Message)

		return nil
	},
}

// x402OrderCancelCmd cancels an order
var x402OrderCancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel an order",
	Long: `Cancel a pending payment order.

Example:
  ./did_helper x402 order cancel --did did:finai:users:0x123... --order-id <ORDER_ID>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if x402DID == "" || x402OrderID == "" {
			return fmt.Errorf("--did and --order-id are required")
		}

		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		did, _ := sm.ResolveDID(x402DID)

		// Confirmation prompt
		if !x402Force {
			fmt.Printf("⚠️  WARNING: Cancelling order is IRREVERSIBLE!\n")
			fmt.Printf("Order ID: %s\n\n", x402OrderID)
			fmt.Print("Are you sure? Type 'yes' to confirm: ")

			var confirmation string
			fmt.Scanln(&confirmation)

			if strings.ToLower(confirmation) != "yes" {
				return fmt.Errorf("cancellation cancelled")
			}
		}

		// Load ticket
		ticket, err := sm.LoadTicket(did)
		if err != nil {
			return fmt.Errorf("ticket not found or failed to load: %w", err)
		}

		config, err := loadX402Config(sm)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/api/x402WebClient/v1/orders/%s/cancel", config.X402API, x402OrderID)
		
		cancelReq := map[string]interface{}{
			"reason": "User requested cancellation",
		}

		httpClient := client.NewHTTPClient()
		statusCode, res, err := httpClient.PostWithTicketAndDID(url, cancelReq, ticket.Ticket, did)
		if err != nil {
			return fmt.Errorf("failed to cancel order: %w", err)
		}

		if statusCode != 200 {
			return fmt.Errorf("cancellation failed (status: %d): %s", statusCode, res)
		}

		var apiResp models.APIResponse
		if err := json.Unmarshal([]byte(res), &apiResp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if !apiResp.Success {
			return fmt.Errorf("cancellation failed: %s", apiResp.Message)
		}

		fmt.Println("✅ Order cancelled successfully!")
		fmt.Printf("Message: %s\n", apiResp.Message)

		// Update local storage
		orderMgr, _ := storage.NewOrderStorageManager()
		if orderMgr != nil {
			orderMgr.UpdateOrderStatus(did, x402OrderID, "cancelled", "")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(x402Cmd)

	// Payment subcommand
	paymentCmd := &cobra.Command{
		Use:   "payment",
		Short: "Manage payments",
	}
	x402Cmd.AddCommand(paymentCmd)
	paymentCmd.AddCommand(x402PaymentCreateCmd)

	// Order subcommand
	orderCmd := &cobra.Command{
		Use:   "order",
		Short: "Manage orders",
	}
	x402Cmd.AddCommand(orderCmd)
	orderCmd.AddCommand(x402OrderListCmd)
	orderCmd.AddCommand(x402OrderShowCmd)
	orderCmd.AddCommand(x402OrderStatusCmd)
	orderCmd.AddCommand(x402OrderRetryCmd)
	orderCmd.AddCommand(x402OrderCancelCmd)

	// Pay command
	x402Cmd.AddCommand(x402PayCmd)

	// Flags for payment create
	x402PaymentCreateCmd.Flags().StringVarP(&x402DID, "did", "d", "", "DID identifier (required)")
	x402PaymentCreateCmd.Flags().StringVarP(&x402Amount, "amount", "a", "", "Payment amount in USDC (required)")
	x402PaymentCreateCmd.Flags().StringVarP(&x402Recipient, "recipient", "r", "", "Recipient address (required)")

	// Flags for order list
	x402OrderListCmd.Flags().StringVarP(&x402DID, "did", "d", "", "DID identifier (required)")
	x402OrderListCmd.Flags().StringVarP(&x402Status, "status", "s", "", "Filter by status (pending, paid, failed, cancelled, expired)")
	x402OrderListCmd.Flags().IntVarP(&x402Limit, "limit", "l", 10, "Maximum number of orders to return")

	// Flags for order show
	x402OrderShowCmd.Flags().StringVarP(&x402DID, "did", "d", "", "DID identifier (required)")
	x402OrderShowCmd.Flags().StringVarP(&x402OrderID, "order-id", "o", "", "Order ID (required)")

	// Flags for pay
	x402PayCmd.Flags().StringVarP(&x402DID, "did", "d", "", "DID identifier (required)")
	x402PayCmd.Flags().StringVarP(&x402OrderID, "order-id", "o", "", "Order ID (required)")

	// Flags for order status
	x402OrderStatusCmd.Flags().StringVarP(&x402DID, "did", "d", "", "DID identifier (required)")
	x402OrderStatusCmd.Flags().StringVarP(&x402OrderID, "order-id", "o", "", "Order ID (required)")

	// Flags for order retry
	x402OrderRetryCmd.Flags().StringVarP(&x402DID, "did", "d", "", "DID identifier (required)")
	x402OrderRetryCmd.Flags().StringVarP(&x402OrderID, "order-id", "o", "", "Order ID (required)")

	// Flags for order cancel
	x402OrderCancelCmd.Flags().StringVarP(&x402DID, "did", "d", "", "DID identifier (required)")
	x402OrderCancelCmd.Flags().StringVarP(&x402OrderID, "order-id", "o", "", "Order ID (required)")
	x402OrderCancelCmd.Flags().BoolVar(&x402Force, "force", false, "Skip confirmation prompt")
}
