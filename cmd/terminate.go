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
	"strconv"

	"github.com/spf13/cobra"
)

type TerminateRequest struct {
	ID string `json:"id"`
}

type OnDemandTerminateRequest struct {
	RentalID int `json:"rentalId"`
}

// Lightweight struct for termination - only need ID
type OnDemandInstanceForTerminate struct {
	ID int `json:"id"`
}

// terminateCmd represents the terminate command
var terminateCmd = &cobra.Command{
	Use:   "terminate [instance-id]",
	Short: "Terminate a rented instance.",
	Long:  `Terminate a rented spot or on-demand instance by providing the instance ID. Run 'hyperbolic instances' to see your active instances.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		instanceID := args[0]

		if instanceID == "" {
			fmt.Println("Error: Instance ID is required")
			fmt.Println("Usage: hyperbolic terminate [instance-id]")
			fmt.Println("Run 'hyperbolic instances' to see your active instances")
			return
		}

		err := terminateInstance(instanceID)
		if err != nil {
			fmt.Printf("Error terminating instance: %v\n", err)
			return
		}

		// Success message is handled within terminateOnDemandInstance for on-demand instances
		// Only show generic message for spot instances
		if _, parseErr := strconv.Atoi(instanceID); parseErr != nil {
			// This is a spot instance (string ID)
			fmt.Printf("Successfully terminated instance: %s\n", instanceID)
		}
	},
}

func terminateInstance(instanceID string) error {
	// Get API key from config file
	apiKey, err := GetAPIKey()
	if err != nil {
		return fmt.Errorf("authentication error: %v\nPlease run 'hyperbolic auth YOUR_API_KEY' to save your API key\n(Get your API key from https://app.hyperbolic.ai/settings)", err)
	}

	// Check if this is an on-demand instance (integer ID) or spot instance (string ID)
	if rentalID, err := strconv.Atoi(instanceID); err == nil {
		// This is an on-demand instance
		return terminateOnDemandInstance(rentalID, apiKey)
	} else {
		// This is a spot instance
		return terminateSpotInstance(instanceID, apiKey)
	}
}

func terminateSpotInstance(instanceID string, apiKey string) error {
	url := "https://api.hyperbolic.xyz/v1/marketplace/instances/terminate"

	// Create request payload
	request := TerminateRequest{
		ID: instanceID,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API error (status code %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func terminateOnDemandInstance(rentalID int, apiKey string) error {
	// First, we need to determine if this is a VM or bare-metal instance
	// by checking both endpoints to find the instance
	instanceType, err := findOnDemandInstanceType(rentalID, apiKey)
	if err != nil {
		return fmt.Errorf("failed to find instance: %v", err)
	}

	// Choose the correct endpoint based on instance type
	var endpoint string
	if instanceType == "vm" {
		endpoint = "https://api.hyperbolic.xyz/v2/marketplace/virtual-machine-rentals/terminate"
	} else {
		endpoint = "https://api.hyperbolic.xyz/v2/marketplace/bare-metal-rentals/terminate"
	}

	// Create request payload
	request := OnDemandTerminateRequest{
		RentalID: rentalID,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API error (status code %d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Successfully terminated %s instance with id %d\n", 
		map[string]string{"vm": "VM", "bare-metal": "Bare Metal"}[instanceType], 
		rentalID)

	return nil
}

func findOnDemandInstanceType(rentalID int, apiKey string) (string, error) {
	// Check VM instances first
	vmInstances, err := fetchVMInstancesForTerminate(apiKey)
	if err == nil {
		for _, instance := range vmInstances {
			if instance.ID == rentalID {
				return "vm", nil
			}
		}
	}

	// Check bare-metal instances
	bmInstances, err := fetchBMInstancesForTerminate(apiKey)
	if err == nil {
		for _, instance := range bmInstances {
			if instance.ID == rentalID {
				return "bare-metal", nil
			}
		}
	}

	return "", fmt.Errorf("instance with ID %d not found in either VM or bare-metal instances", rentalID)
}

func fetchVMInstancesForTerminate(apiKey string) ([]OnDemandInstanceForTerminate, error) {
	url := "https://api.hyperbolic.xyz/v2/marketplace/virtual-machine-rentals"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch VM instances")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var instances []OnDemandInstanceForTerminate
	err = json.Unmarshal(body, &instances)
	return instances, err
}

func fetchBMInstancesForTerminate(apiKey string) ([]OnDemandInstanceForTerminate, error) {
	url := "https://api.hyperbolic.xyz/v2/marketplace/bare-metal-rentals"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch bare-metal instances")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var instances []OnDemandInstanceForTerminate
	err = json.Unmarshal(body, &instances)
	return instances, err
}

func init() {
	rootCmd.AddCommand(terminateCmd)
} 