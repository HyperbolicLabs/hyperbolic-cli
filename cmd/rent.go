/*
Copyright © 2025 Hyperbolic Labs
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

type RentRequest struct {
	ClusterName string `json:"cluster_name"`
	NodeName    string `json:"node_name"`
	GpuCount    int    `json:"gpu_count"`
	Image       *Image `json:"image,omitempty"`
}

type Image struct {
	Name  string `json:"name"`
	Ports []int  `json:"ports,omitempty"`
}

// OnDemand request structures
type VirtualMachineRentalRequest struct {
	ConfigID string `json:"configId"`
	GPUCount string `json:"gpuCount"`
}

type BareMetalRentalRequest struct {
	ConfigID    string `json:"configId"`
	NetworkType string `json:"networkType"`
	GPUCount    int    `json:"gpuCount"`
}

// Response structures
type SpotRentResponse struct {
	InstanceID string `json:"instance_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

type OnDemandRentResponse struct {
	ID           int `json:"id"`
	ExternalID   string `json:"externalId"`
	CostPerHour  int `json:"costPerHour"`
	Meta         struct {
		Name        string `json:"name"`
		GPUCount    int    `json:"gpu_count"`
		RentalType  string `json:"rental_type"`
		NetworkType string `json:"network_type"`
	} `json:"meta"`
}

// rentCmd represents the rent command
var rentCmd = &cobra.Command{
	Use:   "rent",
	Short: "Rent an available GPU instance.",
	Long: `Rent a GPU instance from one of Hyperbolic's marketplaces. Supports both 'spot' and 'ondemand' marketplaces.

MARKETPLACES:
  spot       - Rent containerized H100s from $0.99/hr, A100s, 4090s, etc. subject to availability.
  ondemand   - Rent production-grade H100s from $1.49/hr in multi-node bare metal and virtual machine configurations.

For Marketplace-specific options, run:
'hyperbolic rent spot --help'
'hyperbolic rent ondemand --help'

To view available instances, run 'hyperbolic spot' or 'hyperbolic ondemand'.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default to spot if no subcommand is provided
		rentSpotInstance(cmd)
	},
}

// rentSpotCmd represents the spot subcommand
var rentSpotCmd = &cobra.Command{
	Use:   "spot",
	Short: "Rent a GPU instance from the spot marketplace",
	Long: `Rent containerized H100s from $0.99/hr, A100s, 4090s, etc. subject to availability.

REQUIRED FLAGS:
  --cluster-name    Cluster name for the instance
  --node-name       Node name for the instance

OPTIONAL FLAGS:
  --gpu-count       Number of GPUs to rent (default: 1)
  --ports           Ports to expose (up to 2 ports) 

EXAMPLE:
  hyperbolic rent spot --cluster-name cluster-1 --node-name node-1 --gpu-count 2 --ports 8080,3000

Use 'hyperbolic spot' to view available clusters and nodes.`,
	Run: func(cmd *cobra.Command, args []string) {
		rentSpotInstance(cmd)
	},
}

// rentOnDemandCmd represents the ondemand subcommand
var rentOnDemandCmd = &cobra.Command{
	Use:   "ondemand",
	Short: "Rent a GPU instance from the ondemand marketplace",
	Long: `Rent production-grade H100s from $1.49/hr in multi-node bare metal and virtual machine configurations.

REQUIRED FLAGS:
  --instance-type   Instance type: 'virtual-machine' or 'bare-metal'
  --gpu-count       Number of GPUs to rent

CONDITIONAL FLAGS:
  --network-type    Network type for bare-metal: 'ethernet' or 'infiniband' (required for bare-metal)

EXAMPLES:
  hyperbolic rent ondemand --instance-type virtual-machine --gpu-count 4

  hyperbolic rent ondemand --instance-type bare-metal --network-type infiniband --gpu-count 16

Use 'hyperbolic ondemand' to view available configurations and pricing.`,
	Run: func(cmd *cobra.Command, args []string) {
		rentOnDemandInstance(cmd)
	},
}

func rentSpotInstance(cmd *cobra.Command) {
	clusterName, _ := cmd.Flags().GetString("cluster-name")
	nodeName, _ := cmd.Flags().GetString("node-name")
	gpuCount, _ := cmd.Flags().GetInt("gpu-count")
	portStrings, _ := cmd.Flags().GetStringSlice("ports")

	// Validate and process ports
	var ports []int
	if len(portStrings) > 2 {
		fmt.Printf("Error: Maximum of 2 ports can be specified, but %d were provided\n", len(portStrings))
		return
	}

	for _, portStr := range portStrings {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			fmt.Printf("Error: Invalid port number '%s': %v\n", portStr, err)
			return
		}
		if port < 1 || port > 65535 {
			fmt.Printf("Error: Port number '%d' is out of valid range (1-65535)\n", port)
			return
		}
		ports = append(ports, port)
	}

	// Get API key from config file
	apiKey, err := GetAPIKey()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Please run 'hyperbolic auth YOUR_API_KEY' to save your API key\n")
		fmt.Printf("(Get your API key from https://app.hyperbolic.ai/settings)\n")
		return
	}

	request := RentRequest{
		ClusterName: clusterName,
		NodeName:    nodeName,
		GpuCount:    gpuCount,
	}

	// Only include image if ports are specified
	if len(ports) > 0 {
		request.Image = &Image{
			Name:  "ghcr.io/hyperboliclabs/hyper-dos/sshbox",
			Ports: ports,
		}
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	req, err := http.NewRequest("POST", "https://api.hyperbolic.xyz/v1/marketplace/instances/create", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		if resp.StatusCode == http.StatusInternalServerError {
			fmt.Printf("The server is temporarily experiencing issues. Please try again in a few moments.\n")
		} else {
			fmt.Printf("Error response from API (status code %d): %s\n", resp.StatusCode, string(body))
		}
		return
	}

	var spotResponse SpotRentResponse
	if err := json.Unmarshal(body, &spotResponse); err != nil {
		// Fallback to simple success message if parsing fails
		fmt.Printf("Successfully requested GPU instance.\n")
		fmt.Printf("Configuration: %s/%s with %d GPU(s)\n", clusterName, nodeName, gpuCount)
	} else {
		fmt.Printf("Successfully requested GPU instance: %s\n", spotResponse.InstanceID)
		fmt.Printf("Configuration: %s/%s with %d GPU(s)\n", clusterName, nodeName, gpuCount)
	}
	
	fmt.Println()
	fmt.Println("To view the status and get the SSH command, run:")
	fmt.Println("  hyperbolic instances")
	if len(ports) > 0 {
		fmt.Println()
		fmt.Println("To view public URLs for exposed ports, run:")
		fmt.Println("  hyperbolic instances <instance-id>")
	}
}

func rentOnDemandInstance(cmd *cobra.Command) {
	instanceType, _ := cmd.Flags().GetString("instance-type")
	gpuCount, _ := cmd.Flags().GetInt("gpu-count")
	networkType, _ := cmd.Flags().GetString("network-type")

	// Validate instance type
	if instanceType != "virtual-machine" && instanceType != "bare-metal" {
		fmt.Printf("Error: Invalid instance type '%s'. Must be 'virtual-machine' or 'bare-metal'\n", instanceType)
		return
	}

	// Validate network type for bare metal
	if instanceType == "bare-metal" {
		if networkType != "ethernet" && networkType != "infiniband" {
			fmt.Printf("Error: Invalid network type '%s' for bare-metal. Must be 'ethernet' or 'infiniband'\n", networkType)
			return
		}
	}

	// Get API key from config file
	apiKey, err := GetAPIKey()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Please run 'hyperbolic auth YOUR_API_KEY' to save your API key\n")
		fmt.Printf("(Get your API key from https://app.hyperbolic.ai/settings)\n")
		return
	}

	var endpoint string
	var requestBody []byte

	if instanceType == "virtual-machine" {
		endpoint = "https://api.hyperbolic.xyz/v2/marketplace/virtual-machine-rentals"
		request := VirtualMachineRentalRequest{
			ConfigID: "c6fd6253-cbb6-4ea8-a20c-47644b431f1c",
			GPUCount: strconv.Itoa(gpuCount),
		}
		requestBody, err = json.Marshal(request)
	} else { // bare-metal
		endpoint = "https://api.hyperbolic.xyz/v2/marketplace/bare-metal-rentals"
		request := BareMetalRentalRequest{
			ConfigID:    "a3111bd4-550a-47d0-838a-0a52bff2ae3f",
			NetworkType: networkType,
			GPUCount:    gpuCount,
		}
		requestBody, err = json.Marshal(request)
	}

	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		if resp.StatusCode == http.StatusInternalServerError {
			fmt.Printf("The server is temporarily experiencing issues. Please try again in a few moments.\n")
			fmt.Printf("If the problem persists, please contact support.\n")
		} else {
			fmt.Printf("Error response from API (status code %d): %s\n", resp.StatusCode, string(body))
		}
		return
	}

	var onDemandResponse OnDemandRentResponse
	if err := json.Unmarshal(body, &onDemandResponse); err != nil {
		// Fallback to simple success message if parsing fails
		fmt.Printf("Successfully requested on-demand GPU instance.\n")
		if instanceType == "bare-metal" {
			fmt.Printf("Configuration: %s with %d GPU(s), %s network\n", instanceType, gpuCount, networkType)
		} else {
			fmt.Printf("Configuration: %s with %d GPU(s)\n", instanceType, gpuCount)
		}
	} else {
		fmt.Printf("Successfully requested on-demand GPU instance with id: %d\n", onDemandResponse.ID)
		if instanceType == "bare-metal" {
			fmt.Printf("Configuration: %s with %d GPU(s), %s network\n", instanceType, gpuCount, networkType)
		} else {
			fmt.Printf("Configuration: %s with %d GPU(s)\n", instanceType, gpuCount)
		}
		fmt.Printf("Total cost: $%.2f/hour\n", float64(onDemandResponse.CostPerHour)/100)
	}
	
	fmt.Println()
	fmt.Println("To view the status and get the SSH command, run:")
	fmt.Println("  hyperbolic instances")
}

func init() {
	rootCmd.AddCommand(rentCmd)
	
	// Set custom help template to hide usage, flags, and commands
	rentCmd.SetHelpTemplate(`{{.Long}}

`)
	
	// Add subcommands
	rentCmd.AddCommand(rentSpotCmd)
	rentCmd.AddCommand(rentOnDemandCmd)

	// Default spot flags for backward compatibility (hidden from help)
	rentCmd.Flags().String("cluster-name", "", "Cluster name for the instance (required)")
	rentCmd.Flags().String("node-name", "", "Node name for the instance (required)")
	rentCmd.Flags().Int("gpu-count", 1, "Number of GPUs to rent")
	rentCmd.Flags().StringSlice("ports", []string{}, "Ports to expose (up to 2 ports, e.g., --ports 8080,3000 or --ports 8080 --ports 3000)")
	
	// Hide these flags from the main help
	rentCmd.Flags().MarkHidden("cluster-name")
	rentCmd.Flags().MarkHidden("node-name")
	rentCmd.Flags().MarkHidden("gpu-count")
	rentCmd.Flags().MarkHidden("ports")

	// Spot marketplace flags
	rentSpotCmd.Flags().String("cluster-name", "", "Cluster name for the instance (required)")
	rentSpotCmd.Flags().String("node-name", "", "Node name for the instance (required)")
	rentSpotCmd.Flags().Int("gpu-count", 1, "Number of GPUs to rent")
	rentSpotCmd.Flags().StringSlice("ports", []string{}, "Ports to expose (up to 2 ports, e.g., --ports 8080,3000 or --ports 8080 --ports 3000)")
	
	// Mark required flags for spot
	rentSpotCmd.MarkFlagRequired("cluster-name")
	rentSpotCmd.MarkFlagRequired("node-name")
	
	// OnDemand marketplace flags
	rentOnDemandCmd.Flags().String("instance-type", "", "Instance type: 'virtual-machine' or 'bare-metal' (required)")
	rentOnDemandCmd.Flags().String("network-type", "", "Network type for bare-metal instances: 'ethernet' or 'infiniband' (required for bare-metal)")
	rentOnDemandCmd.Flags().Int("gpu-count", 1, "Number of GPUs to rent")
	
	// Mark required flags for ondemand
	rentOnDemandCmd.MarkFlagRequired("instance-type")
}
