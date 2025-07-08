/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

type TerminateRequest struct {
	ID string `json:"id"`
}

// terminateCmd represents the terminate command
var terminateCmd = &cobra.Command{
	Use:   "terminate [instance-id]",
	Short: "Terminate a rented instance.",
	Long:  `Terminate a rented instance by providing the --instance-id. Run 'hyperbolic instances' to see your active instances.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		instanceID := args[0]

		if instanceID == "" {
			fmt.Println("Error: Instance ID is required")
			fmt.Println("Usage: hyperbolic terminate [instance-id]")
			fmt.Println("Run 'hyperbolic instances' to see your active instances")
			return
		}

		response, err := terminateInstance(instanceID)
		if err != nil {
			fmt.Printf("Error terminating instance: %v\n", err)
			return
		}

		fmt.Printf("Successfully terminated instance: %s\n", instanceID)
		fmt.Println(response)
	},
}

func terminateInstance(instanceID string) (string, error) {
	url := "https://api.hyperbolic.xyz/v1/marketplace/instances/terminate"

	// Get API key from config file
	apiKey, err := GetAPIKey()
	if err != nil {
		return "", fmt.Errorf("authentication error: %v\nPlease run 'hyperbolic auth YOUR_API_KEY' to save your API key\n(Get your API key from https://app.hyperbolic.ai/settings)", err)
	}

	// Create request payload
	request := TerminateRequest{
		ID: instanceID,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("API error (status code %d): %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}

func init() {
	rootCmd.AddCommand(terminateCmd)
} 