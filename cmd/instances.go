/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
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

// Response structure for the Hyperbolic Instances API
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

// instancesCmd represents the instances command
var instancesCmd = &cobra.Command{
	Use:   "instances [instance-id]",
	Short: "View your active instances.",
	Long:  `View all your currently rented instances on Hyperbolic. This shows the status, SSH connection details, and pricing information for each instance. You can also specify an instance ID to get detailed information about a specific instance.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFormat, _ := cmd.Flags().GetBool("json")

		response, err := callHyperbolicInstancesAPI()
		if err != nil {
			fmt.Printf("Error calling Hyperbolic API: %v\n", err)
			return
		}

		// Parse the JSON response
		var instancesData InstancesResponse
		err = json.Unmarshal([]byte(response), &instancesData)
		if err != nil {
			fmt.Printf("Error parsing API response: %v\n", err)
			return
		}

		// If an instance ID is provided, show detailed info for that instance
		if len(args) > 0 {
			instanceID := args[0]
			showInstanceDetails(instancesData.Instances, instanceID, jsonFormat)
			return
		}

		if jsonFormat {
			// If json flag is set, print raw JSON response
			fmt.Println(response)
		} else {
			// Otherwise, format as a table
			printUserInstancesTable(instancesData.Instances)
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

// calculateUptime calculates the uptime duration from start time to current time (or end time if available)
func calculateUptime(startTime string, endTime *string) string {
	// Try to parse the start time in ISO 8601 format
	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		// Try alternative formats if RFC3339 fails
		start, err = time.Parse("2006-01-02T15:04:05Z", startTime)
		if err != nil {
			return "N/A"
		}
	}

	var end time.Time
	if endTime != nil && *endTime != "" {
		// If there's an end time, calculate duration until then
		end, err = time.Parse(time.RFC3339, *endTime)
		if err != nil {
			end, err = time.Parse("2006-01-02T15:04:05Z", *endTime)
			if err != nil {
				end = time.Now()
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
func showInstanceDetails(instances []UserInstance, instanceID string, jsonFormat bool) {
	// Find the instance by ID
	var foundInstance *UserInstance
	for _, instance := range instances {
		if instance.ID == instanceID {
			foundInstance = &instance
			break
		}
	}

	if foundInstance == nil {
		fmt.Printf("Instance '%s' not found.\n", instanceID)
		return
	}

	if jsonFormat {
		// Output the instance as JSON
		instanceJSON, err := json.MarshalIndent(foundInstance, "", "  ")
		if err != nil {
			fmt.Printf("Error formatting instance JSON: %v\n", err)
			return
		}
		fmt.Println(string(instanceJSON))
	} else {
		// Display formatted instance details
		printInstanceDetails(*foundInstance)
	}
}

// printInstanceDetails prints detailed information about a single instance in a formatted way
func printInstanceDetails(instance UserInstance) {
	fmt.Printf("Instance Details: %s\n", instance.ID)

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
			fmt.Printf("Port %d: %s://%s:%d\n", mapping.Port, mapping.Protocol, mapping.Domain, mapping.Port)
		}
	}
}

func printUserInstancesTable(instances []UserInstance) {
	if len(instances) == 0 {
		fmt.Println("No instances found.")
		return
	}

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

	// Show count of instances
	fmt.Printf("\nShowing %d instances.\n", len(instances))
	fmt.Printf("Run 'hyperbolic instances instance-id' to view full instance information.\n")
}

func init() {
	rootCmd.AddCommand(instancesCmd)
	instancesCmd.Flags().Bool("json", false, "Output raw JSON response")
} 