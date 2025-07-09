/*
Copyright © 2025 Hyperbolic Labs
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// Balance response structure
type BalanceResponse struct {
	Credits int `json:"credits"`
}

type UserResponse struct {
	Email           string        `json:"email"`
	Picture         interface{}   `json:"picture"`
	Provider        string        `json:"provider"`
	EmailVerified   bool          `json:"email_verified"`
	Name            string        `json:"name"`
	PublicKey       string        `json:"public_key"`
	OnboardedAt     time.Time     `json:"onboarded_at"`
	OnboardedFor    string        `json:"onboarded_for"`
	Meta            interface{}   `json:"meta"`
	ReferralCode    string        `json:"referral_code"`
	ID              string        `json:"id"`
	IsActive        bool          `json:"is_active"`
	APIKey          string        `json:"api_key"`
	Role            string        `json:"role"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	CompletedPromos []interface{} `json:"completed_promos"`
	Roles           []interface{} `json:"roles"`
}

// accountCmd represents the account command
var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "View your account information and balance.",
	Long:  `View your Hyperbolic account information and balance.`,
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

		// Fetch user information
		user, err := fetchUser(apiKey)
		if err != nil {
			fmt.Printf("Error: failed to fetch user information: %v\n", err)
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
			accountInfo := map[string]interface{}{
				"user":    user,
				"balance": balance,
			}
			accountJSON, err := json.MarshalIndent(accountInfo, "", "  ")
			if err != nil {
				fmt.Printf("Error formatting JSON: %v\n", err)
				return
			}
			fmt.Println(string(accountJSON))
		} else {
			// Display formatted account info
			printAccountInfo(user, balance)
		}
	},
}

func fetchUser(apiKey string) (UserResponse, error) {
	url := "https://api.hyperbolic.xyz/users/me"
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return UserResponse{}, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return UserResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UserResponse{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return UserResponse{}, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var userResponse UserResponse
	err = json.Unmarshal(body, &userResponse)
	if err != nil {
		return UserResponse{}, fmt.Errorf("failed to parse user response: %v", err)
	}

	return userResponse, nil
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

func printAccountInfo(user UserResponse, balance BalanceResponse) {
	// Print email first
	fmt.Printf("Email: %s\n", user.Email)
	
	// Convert credits (stored in cents) to dollars
	dollars := float64(balance.Credits) / 100.0
	fmt.Printf("Balance: $%.2f\n", dollars)
}

func init() {
	rootCmd.AddCommand(accountCmd)
	accountCmd.Flags().Bool("json", false, "Output raw JSON response")
} 