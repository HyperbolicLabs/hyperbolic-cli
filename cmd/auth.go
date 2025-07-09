/*
Copyright © 2025 Hyperbolic Labs
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth <api-key>",
	Short: "Authenticate with your Hyperbolic API key",
	Long: `Add your Hyperbolic API key for CLI usage. Create one at https://app.hyperbolic.ai/settings.`,
	Example: `hyperbolic auth your-hyperbolic-api-key`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := strings.TrimSpace(args[0])
		
		if apiKey == "" {
			fmt.Println("Error: API key cannot be empty")
			return
		}
		
		// Create config with the API key
		config := &Config{
			APIKey: apiKey,
		}
		
		// Save the config
		if err := SaveConfig(config); err != nil {
			fmt.Printf("Error saving configuration: %v\n", err)
			return
		}
		
		fmt.Println("✓ API key saved successfully!")
		fmt.Println("You can now use other commands like 'hyperbolic rent' without setting environment variables.")
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
} 