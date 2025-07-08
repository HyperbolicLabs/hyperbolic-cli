/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

// Balance response structure
type BalanceResponse struct {
	Credits int `json:"credits"`
}



// balanceCmd represents the balance command
var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "View your account balance.",
	Long:  `View your Hyperbolic account balance.`,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFormat, _ := cmd.Flags().GetBool("json")

		// Get API key from config file
		apiKey, err := GetAPIKey()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Printf("Please run 'hyperbolic auth YOUR_API_KEY' to save your API key\n")
			fmt.Printf("(Get your API key from https://app.hyperbolic.ai/settings)\n")
			return
		}

		// Fetch balance
		balance, err := fetchBalance(apiKey)
		if err != nil {
			fmt.Printf("Error: failed to fetch balance: %v\n", err)
			return
		}

		if jsonFormat {
			// Output as JSON
			balanceJSON, err := json.MarshalIndent(balance, "", "  ")
			if err != nil {
				fmt.Printf("Error formatting JSON: %v\n", err)
				return
			}
			fmt.Println(string(balanceJSON))
		} else {
			// Display formatted balance info
			printBalanceInfo(balance)
		}
	},
}



func fetchBalance(apiKey string) (BalanceResponse, error) {
	url := "https://api.hyperbolic.xyz/billing/get_current_balance"
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return BalanceResponse{}, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return BalanceResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return BalanceResponse{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return BalanceResponse{}, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var balanceResponse BalanceResponse
	err = json.Unmarshal(body, &balanceResponse)
	if err != nil {
		return BalanceResponse{}, fmt.Errorf("failed to parse balance response: %v", err)
	}

	return balanceResponse, nil
}

func printBalanceInfo(balance BalanceResponse) {
	// Convert credits (stored in cents) to dollars
	dollars := float64(balance.Credits) / 100.0
	fmt.Printf("Balance: $%.2f\n", dollars)
}

func init() {
	rootCmd.AddCommand(balanceCmd)
	balanceCmd.Flags().Bool("json", false, "Output raw JSON response")
} 