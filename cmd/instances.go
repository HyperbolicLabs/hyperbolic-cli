/*
Copyright © 2025 Hyperbolic Labs
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// Response structure for the Hyperbolic Instances API (spot instances)
type InstancesResponse struct {
	Instances []UserInstance `json:"instances"`
}

type UserInstance struct {
	ID           string                 `json:"id"`
	Start        string                 `json:"start"`
	End          *string                `json:"end"`
	Created      string                 `json:"created"`
	SSHCommand   string                 `json:"sshCommand"`
	PortMappings []PortMapping          `json:"portMappings"`
	Instance     UserInstanceDetails    `json:"instance"`
}

type PortMapping struct {
	Domain   string `json:"domain"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}

type UserInstanceDetails struct {
	ID       string                 `json:"id"`
	Status   string                 `json:"status"`
	Hardware UserInstanceHardware   `json:"hardware"`
	Pricing  UserInstancePricing    `json:"pricing"`
	GPUCount int                    `json:"gpu_count"`
}

type UserInstanceHardware struct {
	GPUs []UserInstanceGPU `json:"gpus"`
}

type UserInstanceGPU struct {
	Model string `json:"model"`
	RAM   int    `json:"ram"`
}

type UserInstancePricing struct {
	Price UserInstancePrice `json:"price"`
}

type UserInstancePrice struct {
	Amount float64 `json:"amount"`
	Period string  `json:"period"`
}

// On-demand instance structures
type OnDemandInstance struct {
	ID              int                    `json:"id"`
	CreatedAt       string                 `json:"createdAt"`
	UpdatedAt       *string                `json:"updatedAt"`
	DeletedAt       *string                `json:"deletedAt"`
	UserID          string                 `json:"userId"`
	StartedAt       string                 `json:"startedAt"`
	TerminatedAt    *string                `json:"terminatedAt"`
	ExternalID      string                 `json:"externalId"`
	RentalProvider  string                 `json:"rentalProvider"`
	CostPerHour     int                    `json:"costPerHour"`
	Status          string                 `json:"status"`
	Meta            OnDemandInstanceMeta   `json:"meta"`
}

type OnDemandInstanceMeta struct {
	Name            string                     `json:"name"`
	Tags            []string                   `json:"tags,omitempty"`
	Type            string                     `json:"type,omitempty"`
	PublicIP        string                     `json:"public_ip,omitempty"`
	GPUCount        int                        `json:"gpu_count,omitempty"`
	Resources       *OnDemandInstanceResources `json:"resources,omitempty"`
	HostnodeID      string                     `json:"hostnode_id,omitempty"`
	InternalIP      string                     `json:"internal_ip,omitempty"`
	RentalType      string                     `json:"rental_type"`
	SSHCommand      string                     `json:"ssh_command,omitempty"`
	PortForwards    []OnDemandPortForward      `json:"port_forwards,omitempty"`
	OperatingSystem string                     `json:"operating_system,omitempty"`
	TimestampCreation string                   `json:"timestamp_creation,omitempty"`
	// Bare metal specific fields
	SubOrder        *string                    `json:"sub_order,omitempty"`
	NodeCount       int                        `json:"node_count,omitempty"`
	NetworkType     string                     `json:"network_type,omitempty"`
	SpecsPerNode    *OnDemandInstanceSpecs     `json:"specs_per_node,omitempty"`
	Username        string                     `json:"username,omitempty"`
	NodeNetworking  []OnDemandNetworking       `json:"node_networking,omitempty"`
	CreationTimestamp string                   `json:"creation_timestamp,omitempty"`
}

type OnDemandInstanceResources struct {
	RAMGb     int                    `json:"ram_gb"`
	StorageGb int                    `json:"storage_gb"`
	VCPUCount int                    `json:"vcpu_count"`
	GPUs      map[string]OnDemandGPU `json:"gpus"`
}

type OnDemandGPU struct {
	Count int `json:"count"`
}

type OnDemandInstanceSpecs struct {
	RAMGb     int    `json:"ram_gb"`
	CPUCount  int    `json:"cpu_count"`
	CPUModel  string `json:"cpu_model"`
	GPUCount  int    `json:"gpu_count"`
	GPUModel  string `json:"gpu_model"`
	StorageGb int    `json:"storage_gb"`
}

type OnDemandNetworking struct {
	PublicIP  string `json:"public_ip"`
	PrivateIP string `json:"private_ip"`
}

type OnDemandPortForward struct {
	ExternalPort int `json:"external_port"`
	InternalPort int `json:"internal_port"`
}

// instancesCmd represents the instances command
var instancesCmd = &cobra.Command{
	Use:   "instances [instance-id]",
	Short: "View your active instances.",
	Long:  `View all your currently rented instances on Hyperbolic. This shows the status, SSH connection details, and pricing information for each instance. You can also specify an instance ID to get detailed information about a specific instance.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFormat, _ := cmd.Flags().GetBool("json")

		// Fetch both spot and on-demand instances
		spotResponse, err := callHyperbolicInstancesAPI()
		if err != nil {
			fmt.Printf("Error calling Hyperbolic API: %v\n", err)
			return
		}

		vmInstances, bmInstances, err := fetchOnDemandInstances()
		if err != nil {
			fmt.Printf("Error fetching on-demand instances: %v\n", err)
			return
		}

		// Parse the spot instances JSON response
		var spotInstancesData InstancesResponse
		err = json.Unmarshal([]byte(spotResponse), &spotInstancesData)
		if err != nil {
			fmt.Printf("Error parsing API response: %v\n", err)
			return
		}

		// If an instance ID is provided, show detailed info for that instance
		if len(args) > 0 {
			instanceID := args[0]
			showInstanceDetails(spotInstancesData.Instances, vmInstances, bmInstances, instanceID, jsonFormat)
			return
		}

		if jsonFormat {
			// If json flag is set, print raw JSON responses
			response := map[string]interface{}{
				"spot_instances": spotInstancesData,
				"vm_instances":   vmInstances,
				"bm_instances":   bmInstances,
			}
			jsonData, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				fmt.Printf("Error formatting JSON: %v\n", err)
				return
			}
			fmt.Println(string(jsonData))
		} else {
			// Otherwise, format as tables
			printInstancesTables(spotInstancesData.Instances, vmInstances, bmInstances)
		}
	},
}

func callHyperbolicInstancesAPI() (string, error) {
	url := "https://api.hyperbolic.xyz/v1/marketplace/instances"

	// Get API key from config file
	apiKey, err := GetAPIKey()
	if err != nil {
		return "", fmt.Errorf("authentication error: %v\nPlease run 'hyperbolic auth YOUR_API_KEY' to save your API key\n(Get your API key from https://app.hyperbolic.ai/settings)", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
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

	return string(body), nil
}

func fetchOnDemandInstances() ([]OnDemandInstance, []OnDemandInstance, error) {
	// Get API key from config file
	apiKey, err := GetAPIKey()
	if err != nil {
		return nil, nil, fmt.Errorf("authentication error: %v", err)
	}

	// Fetch VM instances
	vmInstances, err := fetchVMInstances(apiKey)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching VM instances: %v", err)
	}

	// Fetch bare-metal instances
	bmInstances, err := fetchBMInstances(apiKey)
	if err != nil {
		return nil, nil, fmt.Errorf("error fetching bare-metal instances: %v", err)
	}

	return vmInstances, bmInstances, nil
}

func fetchVMInstances(apiKey string) ([]OnDemandInstance, error) {
	url := "https://api.hyperbolic.xyz/v2/marketplace/virtual-machine-rentals"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var vmInstances []OnDemandInstance
	err = json.Unmarshal(body, &vmInstances)
	if err != nil {
		return nil, fmt.Errorf("error parsing VM instances response: %v", err)
	}

	return vmInstances, nil
}

func fetchBMInstances(apiKey string) ([]OnDemandInstance, error) {
	url := "https://api.hyperbolic.xyz/v2/marketplace/bare-metal-rentals"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var bmInstances []OnDemandInstance
	err = json.Unmarshal(body, &bmInstances)
	if err != nil {
		return nil, fmt.Errorf("error parsing bare-metal instances response: %v", err)
	}

	return bmInstances, nil
}

// calculateUptime calculates the uptime duration from start time to current time (or end time if available)
func calculateUptime(startTime string, endTime *string) string {
	// Try to parse the start time in different formats
	var start time.Time
	var err error
	
	// Format 1: RFC3339 (e.g., "2006-01-02T15:04:05Z07:00")
	start, err = time.Parse(time.RFC3339, startTime)
	if err != nil {
		// Format 2: Basic ISO format (e.g., "2006-01-02T15:04:05Z")
		start, err = time.Parse("2006-01-02T15:04:05Z", startTime)
		if err != nil {
			// Format 3: On-demand API format (e.g., "2025-07-08 21:53:35.367+00")
			start, err = time.Parse("2006-01-02 15:04:05.999+00", startTime)
			if err != nil {
				// Format 4: On-demand API format without microseconds (e.g., "2025-07-08 21:53:35+00")
				start, err = time.Parse("2006-01-02 15:04:05+00", startTime)
				if err != nil {
					return "N/A"
				}
			}
		}
	}

	var end time.Time
	if endTime != nil && *endTime != "" {
		// Try to parse the end time in the same formats
		end, err = time.Parse(time.RFC3339, *endTime)
		if err != nil {
			end, err = time.Parse("2006-01-02T15:04:05Z", *endTime)
			if err != nil {
				end, err = time.Parse("2006-01-02 15:04:05.999+00", *endTime)
				if err != nil {
					end, err = time.Parse("2006-01-02 15:04:05+00", *endTime)
					if err != nil {
						end = time.Now()
					}
				}
			}
		}
	} else {
		// If no end time, calculate duration until now
		end = time.Now()
	}

	duration := end.Sub(start)
	return formatDuration(duration)
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "N/A"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}

// formatPorts formats the port mappings into a comma-separated string or "None" if no ports
func formatPorts(portMappings []PortMapping) string {
	if len(portMappings) == 0 {
		return "None"
	}

	var ports []string
	for _, mapping := range portMappings {
		ports = append(ports, strconv.Itoa(mapping.Port))
	}

	return strings.Join(ports, ",")
}

// showInstanceDetails displays detailed information about a specific instance
func showInstanceDetails(spotInstances []UserInstance, vmInstances []OnDemandInstance, bmInstances []OnDemandInstance, instanceID string, jsonFormat bool) {
	// Check spot instances first
	for _, instance := range spotInstances {
		if instance.ID == instanceID {
			if jsonFormat {
				instanceJSON, err := json.MarshalIndent(instance, "", "  ")
				if err != nil {
					fmt.Printf("Error formatting instance JSON: %v\n", err)
					return
				}
				fmt.Println(string(instanceJSON))
			} else {
				printSpotInstanceDetails(instance)
			}
			return
		}
	}

	// Check VM instances
	for _, instance := range vmInstances {
		if strconv.Itoa(instance.ID) == instanceID {
			if jsonFormat {
				instanceJSON, err := json.MarshalIndent(instance, "", "  ")
				if err != nil {
					fmt.Printf("Error formatting instance JSON: %v\n", err)
					return
				}
				fmt.Println(string(instanceJSON))
			} else {
				printOnDemandInstanceDetails(instance, "VM")
			}
			return
		}
	}

	// Check bare-metal instances
	for _, instance := range bmInstances {
		if strconv.Itoa(instance.ID) == instanceID {
			if jsonFormat {
				instanceJSON, err := json.MarshalIndent(instance, "", "  ")
				if err != nil {
					fmt.Printf("Error formatting instance JSON: %v\n", err)
					return
				}
				fmt.Println(string(instanceJSON))
			} else {
				printOnDemandInstanceDetails(instance, "Bare Metal")
			}
			return
		}
	}

	fmt.Printf("Instance '%s' not found.\n", instanceID)
}

// printSpotInstanceDetails prints detailed information about a single spot instance in a formatted way
func printSpotInstanceDetails(instance UserInstance) {
	fmt.Printf("Instance ID: %s\n", instance.ID)

	fmt.Printf("Status: %s\n", instance.Instance.Status)
	fmt.Printf("Created: %s\n", instance.Created)
	
	if instance.Start != "" {
		fmt.Printf("Started: %s\n", instance.Start)
		uptime := calculateUptime(instance.Start, instance.End)
		fmt.Printf("Uptime: %s\n", uptime)
	}
	
	if instance.End != nil && *instance.End != "" {
		fmt.Printf("Ended: %s\n", *instance.End)
	}

	var gpuModel string
	if len(instance.Instance.Hardware.GPUs) > 0 {
		gpuModel = instance.Instance.Hardware.GPUs[0].Model
	} else {
		gpuModel = "N/A"
	}
	
	fmt.Printf("GPU Model: %s\n", gpuModel)
	fmt.Printf("GPU Count: %d\n", instance.Instance.GPUCount)
	
	if len(instance.Instance.Hardware.GPUs) > 0 {
		fmt.Printf("GPU RAM: %d GB\n", instance.Instance.Hardware.GPUs[0].RAM/1024)
	}

	dollarAmount := (instance.Instance.Pricing.Price.Amount / 100.0) * float64(instance.Instance.GPUCount)
	fmt.Printf("Price: $%.2f/hr\n", dollarAmount)

	fmt.Printf("SSH Command: %s\n", instance.SSHCommand)

	if len(instance.PortMappings) == 0 {
		fmt.Printf("No ports exposed\n")
	} else {
		for _, mapping := range instance.PortMappings {
			fmt.Printf("Public URL for port %d: %s://%s:%d\n", mapping.Port, mapping.Protocol, mapping.Domain, mapping.Port)
		}
	}
}

// printOnDemandInstanceDetails prints detailed information about a single on-demand instance
func printOnDemandInstanceDetails(instance OnDemandInstance, instanceType string) {
	fmt.Printf("Instance Details: %d\n", instance.ID)
	fmt.Printf("Type: %s\n", instanceType)
	fmt.Printf("Status: %s\n", instance.Status)
	fmt.Printf("Name: %s\n", instance.Meta.Name)
	
	// Timestamps
	if instance.CreatedAt != "" {
		fmt.Printf("Created: %s\n", instance.CreatedAt)
	}
	if instance.StartedAt != "" {
		fmt.Printf("Started: %s\n", instance.StartedAt)
		// Calculate uptime
		uptime := calculateUptime(instance.StartedAt, instance.TerminatedAt)
		fmt.Printf("Uptime: %s\n", uptime)
	}
	if instance.TerminatedAt != nil && *instance.TerminatedAt != "" {
		fmt.Printf("Terminated: %s\n", *instance.TerminatedAt)
	}
	
	// Get GPU count from the appropriate source
	var gpuCount int
	var totalGPUCount int
	if instance.Meta.SpecsPerNode != nil {
		// For bare-metal: store per-node count and calculate total
		gpuCount = instance.Meta.SpecsPerNode.GPUCount
		totalGPUCount = gpuCount * instance.Meta.NodeCount
	} else if instance.Meta.Resources != nil {
		// For VM instances, sum up GPU counts from resources
		for _, gpuInfo := range instance.Meta.Resources.GPUs {
			gpuCount += gpuInfo.Count
		}
		totalGPUCount = gpuCount
	}
	// Fallback to meta gpu_count if available
	if gpuCount == 0 {
		gpuCount = instance.Meta.GPUCount
		totalGPUCount = gpuCount
	}
	
	// Show GPU information
	if instance.Meta.NodeCount > 1 {
		fmt.Printf("Total GPUs: %d (%d×%d)\n", totalGPUCount, gpuCount, instance.Meta.NodeCount)
		fmt.Printf("Node Count: %d\n", instance.Meta.NodeCount)
		fmt.Printf("GPUs per Node: %d\n", gpuCount)
	} else {
		fmt.Printf("GPU Count: %d\n", totalGPUCount)
	}
	
	// Get GPU model
	var gpuModel string
	if instance.Meta.SpecsPerNode != nil {
		gpuModel = instance.Meta.SpecsPerNode.GPUModel
	} else if instance.Meta.Resources != nil {
		// Get the first GPU type from the resources
		for model := range instance.Meta.Resources.GPUs {
			gpuModel = model
			break
		}
	}
	
	if gpuModel != "" {
		// Clean up GPU model name
		gpuModel = strings.ReplaceAll(gpuModel, "NVIDIA-GeForce-", "")
		gpuModel = strings.ReplaceAll(gpuModel, "NVIDIA-", "")
		gpuModel = strings.ReplaceAll(gpuModel, "h100-sxm5-80gb", "H100-SXM5-80GB")
		fmt.Printf("GPU Model: %s\n", gpuModel)
	}
	
	// Hardware details
	if instance.Meta.Resources != nil {
		fmt.Printf("RAM: %d GB\n", instance.Meta.Resources.RAMGb)
		fmt.Printf("Storage: %d GB\n", instance.Meta.Resources.StorageGb)
		fmt.Printf("vCPU Count: %d\n", instance.Meta.Resources.VCPUCount)
	} else if instance.Meta.SpecsPerNode != nil {
		fmt.Printf("RAM: %d GB\n", instance.Meta.SpecsPerNode.RAMGb)
		fmt.Printf("Storage: %d GB\n", instance.Meta.SpecsPerNode.StorageGb)
		fmt.Printf("CPU Count: %d\n", instance.Meta.SpecsPerNode.CPUCount)
		if instance.Meta.SpecsPerNode.CPUModel != "" {
			fmt.Printf("CPU Model: %s\n", instance.Meta.SpecsPerNode.CPUModel)
		}
	}
	
	fmt.Printf("Price: $%.2f/hr\n", float64(instance.CostPerHour)/100.0)
	
	if instance.Meta.NetworkType != "" {
		fmt.Printf("Network Type: %s\n", instance.Meta.NetworkType)
	}
	
	if instance.Meta.OperatingSystem != "" {
		fmt.Printf("Operating System: %s\n", instance.Meta.OperatingSystem)
	}
	
	// SSH command(s)
	if instance.Meta.SSHCommand != "" {
		// VM with direct SSH command
		fmt.Printf("SSH Command: %s\n", instance.Meta.SSHCommand)
	} else if len(instance.Meta.NodeNetworking) > 0 && instance.Meta.Username != "" {
		// Bare-metal with node networking
		if len(instance.Meta.NodeNetworking) == 1 {
			// Single node
			publicIP := instance.Meta.NodeNetworking[0].PublicIP
			fmt.Printf("SSH Command: ssh %s@%s\n", instance.Meta.Username, publicIP)
		} else {
			// Multi-node
			fmt.Printf("SSH Commands:\n")
			for i, network := range instance.Meta.NodeNetworking {
				fmt.Printf("  Node %d: ssh %s@%s\n", i+1, instance.Meta.Username, network.PublicIP)
			}
		}
	} else if instance.Meta.PublicIP != "" {
		// For VMs, might have public IP directly in meta
		fmt.Printf("SSH Command: ssh user@%s\n", instance.Meta.PublicIP)
	} else {
		// Check if instance is still starting up
		status := strings.ToLower(instance.Status)
		if status == "pending" || status == "starting" || status == "provisioning" || status == "initializing" {
			fmt.Printf("SSH Command: Available when ready\n")
		} else {
			fmt.Printf("SSH Command: SSH details not available\n")
		}
	}
	
	// Port forwards for VMs
	if len(instance.Meta.PortForwards) > 0 {
		fmt.Printf("Port Forwards:\n")
		for _, portForward := range instance.Meta.PortForwards {
			fmt.Printf("  External Port %d → Internal Port %d\n", portForward.ExternalPort, portForward.InternalPort)
		}
	}
	
	// Network information
	if len(instance.Meta.NodeNetworking) > 0 {
		// Bare metal instances
		if len(instance.Meta.NodeNetworking) == 1 {
			fmt.Printf("Network Information:\n")
			network := instance.Meta.NodeNetworking[0]
			fmt.Printf("  Public IP: %s\n", network.PublicIP)
			fmt.Printf("  Private IP: %s\n", network.PrivateIP)
		} else {
			fmt.Printf("Network Information (%d nodes):\n", len(instance.Meta.NodeNetworking))
			for i, network := range instance.Meta.NodeNetworking {
				fmt.Printf("  Node %d: Public IP %s, Private IP %s\n", i+1, network.PublicIP, network.PrivateIP)
			}
		}
	} else if instance.Meta.PublicIP != "" || instance.Meta.InternalIP != "" {
		// VM instances
		fmt.Printf("Network Information:\n")
		if instance.Meta.PublicIP != "" {
			fmt.Printf("  Public IP: %s\n", instance.Meta.PublicIP)
		}
		if instance.Meta.InternalIP != "" {
			fmt.Printf("  Private IP: %s\n", instance.Meta.InternalIP)
		}
	}
}

func printInstancesTables(spotInstances []UserInstance, vmInstances []OnDemandInstance, bmInstances []OnDemandInstance) {
	// Print spot instances table if any exist
	if len(spotInstances) > 0 {
		fmt.Println("SPOT INSTANCES:")
		printSpotInstancesTable(spotInstances)
	}

	// Print on-demand instances table if any exist
	if len(vmInstances) > 0 || len(bmInstances) > 0 {
		if len(spotInstances) > 0 {
			fmt.Println()
		}
		fmt.Println("ON-DEMAND INSTANCES:")
		printOnDemandInstancesTable(vmInstances, bmInstances)
	}

	// Show overall message if no instances
	if len(spotInstances) == 0 && len(vmInstances) == 0 && len(bmInstances) == 0 {
		fmt.Println("No instances found.")
		return
	}
	
	// Show helpful message about instance details
	totalInstances := len(spotInstances) + len(vmInstances) + len(bmInstances)
	if totalInstances > 0 {
		fmt.Printf("\nRun 'hyperbolic instances instance-id' to view full instance information, port forwards, ip addresses, and more.\n")
	}
}

func printSpotInstancesTable(instances []UserInstance) {
	// Create a new table
	table := tablewriter.NewWriter(os.Stdout)
	
	// Set header
	table.Header([]string{
		"STATUS", "INSTANCE ID", "GPU MODEL", "COUNT", "SSH COMMAND", "PORTS", "PRICE", "UPTIME",
	})

	// Add rows
	for _, instance := range instances {
		var gpuModel string
		if len(instance.Instance.Hardware.GPUs) > 0 {
			gpuModel = instance.Instance.Hardware.GPUs[0].Model
		} else {
			gpuModel = "N/A"
		}

		// Convert cents to dollars and multiply by GPU count
		dollarAmount := (instance.Instance.Pricing.Price.Amount / 100.0) * float64(instance.Instance.GPUCount)

		// Calculate uptime
		uptime := calculateUptime(instance.Start, instance.End)

		// Use full SSH command
		sshCommand := instance.SSHCommand

		// Format ports
		ports := formatPorts(instance.PortMappings)

		table.Append([]string{
			instance.Instance.Status,
			instance.ID,
			gpuModel,
			fmt.Sprintf("%d", instance.Instance.GPUCount),
			sshCommand,
			ports,
			fmt.Sprintf("$%.2f/hr", dollarAmount),
			uptime,
		})
	}

	// Render the table
	table.Render()

}

func printOnDemandInstancesTable(vmInstances []OnDemandInstance, bmInstances []OnDemandInstance) {
	// Create a new table
	table := tablewriter.NewWriter(os.Stdout)
	
	// Set header - similar to spot table but with TYPE and NETWORKING instead of PORTS
	table.Header([]string{
		"STATUS", "TYPE", "INSTANCE ID", "GPU MODEL", "COUNT", "SSH COMMAND", "NETWORKING", "PRICE", "UPTIME",
	})

	// Add VM instances
	for _, instance := range vmInstances {
		var gpuModel string
		var gpuCount int
		
		// Get GPU info from resources
		if instance.Meta.Resources != nil {
			// Get the first GPU type from the resources
			for model, gpuInfo := range instance.Meta.Resources.GPUs {
				gpuModel = model
				gpuCount = gpuInfo.Count
				break
			}
		}
		
		// Fallback to meta gpu_count if available
		if gpuCount == 0 {
			gpuCount = instance.Meta.GPUCount
		}
		
		// Clean up GPU model name
		if gpuModel != "" {
			gpuModel = strings.ReplaceAll(gpuModel, "NVIDIA-GeForce-", "")
			gpuModel = strings.ReplaceAll(gpuModel, "NVIDIA-", "")
			gpuModel = strings.ReplaceAll(gpuModel, "h100-sxm5-80gb", "H100-SXM5-80GB")
		} else {
			gpuModel = "H100-SXM5-80GB" // Default for VMs
		}

		// SSH command
		sshCommand := instance.Meta.SSHCommand
		if sshCommand == "" {
			// Check if instance is still starting up
			status := strings.ToLower(instance.Status)
			if status == "pending" || status == "starting" || status == "provisioning" || status == "initializing" {
				sshCommand = "Available when ready"
			} else {
				sshCommand = "SSH details not available"
			}
		}

		// Networking info for VMs (inherently ethernet)
		networking := "Ethernet"

		// Calculate uptime
		uptime := calculateUptime(instance.StartedAt, instance.TerminatedAt)

		table.Append([]string{
			instance.Status,
			"Virtual Machine",
			strconv.Itoa(instance.ID),
			gpuModel,
			fmt.Sprintf("%d", gpuCount),
			sshCommand,
			networking,
			fmt.Sprintf("$%.2f/hr", float64(instance.CostPerHour)/100.0),
			uptime,
		})
	}

	// Add bare-metal instances
	for _, instance := range bmInstances {
		var gpuModel string
		var gpuCount int
		
		if instance.Meta.SpecsPerNode != nil {
			gpuModel = instance.Meta.SpecsPerNode.GPUModel
			gpuCount = instance.Meta.SpecsPerNode.GPUCount
		}
		
		// Clean up GPU model name
		if gpuModel != "" {
			gpuModel = strings.ReplaceAll(gpuModel, "NVIDIA-GeForce-", "")
			gpuModel = strings.ReplaceAll(gpuModel, "NVIDIA-", "")
			gpuModel = strings.ReplaceAll(gpuModel, "h100-sxm5-80gb", "H100-SXM5-80GB")
		} else {
			gpuModel = "H100-SXM5-80GB" // Default for bare-metal
		}

		// SSH command - handle multi-node
		var sshCommand string
		if len(instance.Meta.NodeNetworking) > 0 && instance.Meta.Username != "" {
			if len(instance.Meta.NodeNetworking) == 1 {
				// Single node
				publicIP := instance.Meta.NodeNetworking[0].PublicIP
				sshCommand = fmt.Sprintf("ssh %s@%s", instance.Meta.Username, publicIP)
			} else {
				// Multi-node - show all SSH commands on separate lines
				var sshCommands []string
				for _, network := range instance.Meta.NodeNetworking {
					sshCommands = append(sshCommands, fmt.Sprintf("ssh %s@%s", instance.Meta.Username, network.PublicIP))
				}
				sshCommand = strings.Join(sshCommands, "\n")
			}
		} else {
			// Check if instance is still starting up
			status := strings.ToLower(instance.Status)
			if status == "pending" || status == "starting" || status == "provisioning" || status == "initializing" {
				sshCommand = "Available when ready"
			} else {
				sshCommand = "SSH details not available"
			}
		}

		// Networking info for bare metal - show ethernet/infiniband
		var networking string
		if instance.Meta.NetworkType != "" {
			// Capitalize first letter and show network type
			networking = strings.Title(strings.ToLower(instance.Meta.NetworkType))
		} else {
			networking = "Standard"
		}

		// Calculate uptime
		uptime := calculateUptime(instance.StartedAt, instance.TerminatedAt)

		// Format GPU count with node information
		var gpuCountDisplay string
		if instance.Meta.NodeCount > 1 {
			gpuCountDisplay = fmt.Sprintf("%d×%d", gpuCount, instance.Meta.NodeCount)
		} else {
			gpuCountDisplay = fmt.Sprintf("%d", gpuCount)
		}

		table.Append([]string{
			instance.Status,
			"Bare Metal",
			strconv.Itoa(instance.ID),
			gpuModel,
			gpuCountDisplay,
			sshCommand,
			networking,
			fmt.Sprintf("$%.2f/hr", float64(instance.CostPerHour)/100.0),
			uptime,
		})
	}

	// Render the table
	table.Render()
}

func init() {
	rootCmd.AddCommand(instancesCmd)
	instancesCmd.Flags().Bool("json", false, "Output raw JSON response")
} 