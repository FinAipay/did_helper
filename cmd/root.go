package cmd

import (
	"fmt"
	"os"

	"did_helper/internal/storage"

	"github.com/spf13/cobra"
)

// rootCmd is the root command
var rootCmd = &cobra.Command{
	Use:   "did_helper",
	Short: "FinAI DID Assistant - CLI tool for managing DID documents, credentials, and wallets",
	Long: `FinAI DID Assistant is a powerful CLI tool for:
- Sending RESTful API requests (GET, POST, PUT, DELETE)
- Managing DID documents
- Managing tickets and API keys
- Generating and managing crypto wallets (ETH, Solana, X25519)
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize storage manager
		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		// Initialize default config if not exists
		if err := sm.InitializeDefaultConfig(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}

		return nil
	},
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Command execution failed: %v\n", err)
		os.Exit(1)
	}
}
