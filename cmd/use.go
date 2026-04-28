package cmd

import (
	"fmt"

	"did_helper/internal/storage"

	"github.com/spf13/cobra"
)

// useCmd switches the default DID
var useCmd = &cobra.Command{
	Use:   "use [DID]",
	Short: "Set the default DID",
	Long: `Set the specified DID as the default for subsequent commands.
The default DID is stored in config.json and used when no --did flag is provided.

Example:
  ./did_helper use did:finai:agents:0x1234...
  ./did_helper use did:finai:users:alice`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		did := args[0]

		sm, err := storage.NewDIDStorageManager()
		if err != nil {
			return err
		}

		// Check if DID exists
		if !sm.DIDExists(did) {
			return fmt.Errorf("DID not found: %s\n\nAvailable DIDs:\n%s", did, listAvailableDIDs(sm))
		}

		// Update default DID
		if err := sm.UseDID(did); err != nil {
			return fmt.Errorf("failed to set default DID: %w", err)
		}

		fmt.Printf("✓ Default DID set to: %s\n", did)
		fmt.Println("\nThis DID will be used by default for subsequent commands.")
		fmt.Println("You can override it with the --did flag anytime.")

		return nil
	},
}

func listAvailableDIDs(sm *storage.DIDStorageManager) string {
	dids, err := sm.ListDIDs()
	if err != nil || len(dids) == 0 {
		return "  (no DIDs found)"
	}

	result := ""
	for i, did := range dids {
		result += fmt.Sprintf("  %d. %s\n", i+1, did)
	}
	return result
}

func init() {
	rootCmd.AddCommand(useCmd)
}
