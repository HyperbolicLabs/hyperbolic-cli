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
	"os"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// Response structure for the Hyperbolic API
type MarketplaceResponse struct {
	Instances []Instance `json:"instances"`
}

type Instance struct {
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	Hardware     Hardware `json:"hardware"`
	GpusTotal    int      `json:"gpus_total"`
	GpusReserved int      `json:"gpus_reserved"`
	Location     Location `json:"location"`
	Pricing      Pricing  `json:"pricing"`
	ClusterName  string   `json:"cluster_name"`
	SupplierID   string   `json:"supplier_id"`
}

type Hardware struct {
	CPUs    []CPU     `json:"cpus"`
	GPUs    []GPU     `json:"gpus"`
	Storage []Storage `json:"storage"`
	RAM     []RAM     `json:"ram"`
}

type CPU struct {
	Model        string `json:"model"`
	VirtualCores int    `json:"virtual_cores"`
}

type GPU struct {
	Model     string `json:"model"`
	RAM       int    `json:"ram"`
	Interface string `json:"interface"`
}

type Storage struct {
	Capacity int `json:"capacity"`
}

type RAM struct {
	Capacity int `json:"capacity"`
}

type Location struct {
	Region string `json:"region"`
}

type Pricing struct {
	Price Price `json:"price"`
}

type Price struct {
	Amount int    `json:"amount"`
	Period string `json:"period"`
	Agent  string `json:"agent"`
}

// spotCmd represents the spot command
var spotCmd = &cobra.Command{
	Use:   "spot",
	Short: "View available spot compute resources.",
	Long:  `View all available spot compute resources.`,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFormat, _ := cmd.Flags().GetBool("json")
		showAll, _ := cmd.Flags().GetBool("all")

		response, err := callHyperbolicAPI()
		if err != nil {
			fmt.Printf("Error calling Hyperbolic API: %v\n", err)
			return
		}

		// Parse the JSON response
		var marketplaceData MarketplaceResponse
		err = json.Unmarshal([]byte(response), &marketplaceData)
		if err != nil {
			fmt.Printf("Error parsing API response: %v\n", err)
			return
		}

		if jsonFormat {
			// If json flag is set, print raw JSON response
			fmt.Println(response)
		} else {
			// Otherwise, format as a table
			printInstancesTable(marketplaceData.Instances, showAll)
		}
	},
}

func callHyperbolicAPI() (string, error) {
	url := "https://api.hyperbolic.xyz/v1/marketplace"

	// Create request payload
	payload := map[string]interface{}{  "filters": map[string]interface{}{} }

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling JSON: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

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

func printInstancesTable(instances []Instance, showAll bool) {
	// Filter instances to only those with available GPUs, unless showAll is true
	var filteredInstances []Instance
	for _, instance := range instances {
		availableGPUs := instance.GpusTotal - instance.GpusReserved
		if showAll || availableGPUs > 0 {
			filteredInstances = append(filteredInstances, instance)
		}
	}

	// Sort instances by cheapest price first, then by GPU model
	sort.Slice(filteredInstances, func(i, j int) bool {
		priceI := filteredInstances[i].Pricing.Price.Amount
		priceJ := filteredInstances[j].Pricing.Price.Amount

		// First sort by price (ascending - cheapest first)
		if priceI != priceJ {
			return priceI < priceJ
		}

		// If same price, sort by GPU model
		modelI := getGPUModel(filteredInstances[i])
		modelJ := getGPUModel(filteredInstances[j])
		return strings.Compare(modelI, modelJ) < 0
	})

	// Create a new table
	table := tablewriter.NewWriter(os.Stdout)
	
	// Set header
	table.Header(
		"GPU MODEL", "COUNT", "PRICE", "CLUSTER", "NODE",
		"CPU CORES", "RAM (GB)", "STORAGE (GB)", "REGION",
	)

	// Add rows
	for _, instance := range filteredInstances {
		var gpuModel string
		if len(instance.Hardware.GPUs) > 0 {
			gpuModel = instance.Hardware.GPUs[0].Model
		} else {
			gpuModel = "N/A"
		}

		var cpuCores int
		if len(instance.Hardware.CPUs) > 0 {
			cpuCores = instance.Hardware.CPUs[0].VirtualCores
		}

		var ramGB int
		if len(instance.Hardware.RAM) > 0 {
			ramGB = instance.Hardware.RAM[0].Capacity
		}

		var storageGB int
		if len(instance.Hardware.Storage) > 0 {
			storageGB = instance.Hardware.Storage[0].Capacity
		}

		availableGPUs := instance.GpusTotal - instance.GpusReserved

		// Convert cents to dollars
		dollarAmount := float64(instance.Pricing.Price.Amount) / 100.0

		table.Append(
			gpuModel,
			fmt.Sprintf("%d/%d", availableGPUs, instance.GpusTotal),
			fmt.Sprintf("$%.2f", dollarAmount),
			instance.ClusterName,
			instance.ID,
			fmt.Sprintf("%d", cpuCores),
			fmt.Sprintf("%d", ramGB),
			fmt.Sprintf("%d", storageGB),
			instance.Location.Region,
		)
	}
	fmt.Println("Prices are shown per GPU per hour in USD.")
	// Render the table
	table.Render()

	// Show count of available instances
	fmt.Printf("\nShowing %d instances with available GPUs.\n", len(filteredInstances))
	if !showAll && len(instances) > len(filteredInstances) {
		fmt.Printf("Use --all flag to show all %d instances.\n", len(instances))
	}
}

// getGPUModel returns the GPU model of an instance, or empty string if none
func getGPUModel(instance Instance) string {
	if len(instance.Hardware.GPUs) > 0 {
		return instance.Hardware.GPUs[0].Model
	}
	return ""
}

func init() {
	rootCmd.AddCommand(spotCmd)
	spotCmd.Flags().Bool("json", false, "Output raw JSON response")
	spotCmd.Flags().Bool("all", false, "Show all instances, including those with no available GPUs")
}
