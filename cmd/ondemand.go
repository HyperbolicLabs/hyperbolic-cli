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
	"sort"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// VirtualMachineOption represents a VM option from the API
type VirtualMachineOption struct {
	GPUCount    int     `json:"gpuCount"`
	CostPerHour float64 `json:"costPerHour"`
}

// VirtualMachineOptions represents the response from /v2/marketplace/virtual-machine-options
type VirtualMachineOptions []VirtualMachineOption

// BareMetalNetworkOption represents network configuration for bare metal instances
type BareMetalNetworkOption struct {
	GPUCount    int     `json:"gpuCount"`
	CostPerHour float64 `json:"costPerHour"`
}

// BareMetalOptions represents the response from /v2/marketplace/bare-metal-options
type BareMetalOptions struct {
	Ethernet   BareMetalNetworkOption `json:"ethernet"`
	Infiniband BareMetalNetworkOption `json:"infiniband"`
}



// ondemandCmd represents the ondemand command
var ondemandCmd = &cobra.Command{
	Use:   "ondemand",
	Short: "View available on-demand GPU instances",
	Long:  `View all available on-demand GPU instances with pricing information.`,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFormat, _ := cmd.Flags().GetBool("json")

		if jsonFormat {
			// If json flag is set, print raw JSON responses
			err := printRawJSON()
			if err != nil {
				fmt.Printf("Error fetching data: %v\n", err)
				return
			}
		} else {
			// Otherwise, format as a table
			err := printOnDemandTable()
			if err != nil {
				fmt.Printf("Error fetching data: %v\n", err)
				return
			}
		}
	},
}

func printRawJSON() error {
	vmOptions, bareMetalOptions, err := fetchOnDemandOptions()
	if err != nil {
		return err
	}

	response := map[string]interface{}{
		"virtualMachineOptions": vmOptions,
		"bareMetalOptions":      bareMetalOptions,
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func printOnDemandTable() error {
	vmOptions, bareMetalOptions, err := fetchOnDemandOptions()
	if err != nil {
		return err
	}

	gpuType := "H100-SXM5-80GB"

	// Virtual-machine: min/max GPUs
	if len(vmOptions) == 0 {
		return fmt.Errorf("no virtual machine options available")
	}
	
	// Sort VM options by GPU count for consistent display
	sort.Slice(vmOptions, func(i, j int) bool {
		return vmOptions[i].GPUCount < vmOptions[j].GPUCount
	})
	
	vmBase := vmOptions[0].CostPerHour

	// Bare-metal Ethernet: GPU count and price per GPU-hr
	ethMin, ethMax := 8, bareMetalOptions.Ethernet.GPUCount
	bmEthPrice := bareMetalOptions.Ethernet.CostPerHour

	// Bare-metal InfiniBand: GPU count and price per GPU-hr
	ibMin, ibMax := 8, bareMetalOptions.Infiniband.GPUCount
	bmIBPrice := bareMetalOptions.Infiniband.CostPerHour

	table := tablewriter.NewWriter(os.Stdout)
	table.Header(
		"GPU Type",
		"Instance Type",
		"Count",
		"Price/GPU/hr",
	)

	// VM row - show all available GPU counts
	var vmCountStrs []string
	for _, option := range vmOptions {
		vmCountStrs = append(vmCountStrs, strconv.Itoa(option.GPUCount))
	}
	vmCountStr := strings.Join(vmCountStrs, ", ")
	
	table.Append([]string{
		gpuType,
		"Virtual Machine",
		vmCountStr,
		fmt.Sprintf("$%.2f", vmBase),
	})

	// BM Ethernet row
	if ethMax > 0 {
		table.Append([]string{
			gpuType,
			"Bare Metal (Ethernet)",
			fmt.Sprintf("%d–%d (×8)", ethMin, ethMax),
			fmt.Sprintf("$%.2f", bmEthPrice),
		})
	}

	// BM InfiniBand row
	if ibMax > 0 {
		table.Append([]string{
			gpuType,
			"Bare Metal (InfiniBand)",
			fmt.Sprintf("%d–%d (×8)", ibMin, ibMax),
			fmt.Sprintf("$%.2f", bmIBPrice),
		})
	}

	table.Render()
	fmt.Println("\nBare Metal instances can be configured in multiples of 8 GPUs, subject to availability.")
	fmt.Printf("\nInfiniBand adds $0.50 to the base price of $%.2f/hr\n", bmEthPrice)

	fmt.Println("For rental options, run: `hyperbolic rent ondemand --help`")
	return nil
}

func fetchOnDemandOptions() (VirtualMachineOptions, BareMetalOptions, error) {
	// Get API key
	apiKey, err := GetAPIKey()
	if err != nil {
		return nil, BareMetalOptions{}, fmt.Errorf("failed to get API key: %v", err)
	}

	// Fetch both VM and bare metal options concurrently
	vmChan := make(chan VirtualMachineOptions, 1)
	bareMetalChan := make(chan BareMetalOptions, 1)
	errChan := make(chan error, 2)

	// Fetch VM options
	go func() {
		vmOptions, err := fetchVirtualMachineOptions(apiKey)
		if err != nil {
			errChan <- err
			return
		}
		vmChan <- vmOptions
	}()

	// Fetch bare metal options
	go func() {
		bareMetalOptions, err := fetchBareMetalOptions(apiKey)
		if err != nil {
			errChan <- err
			return
		}
		bareMetalChan <- bareMetalOptions
	}()

	// Wait for results
	var vmOptions VirtualMachineOptions
	var bareMetalOptions BareMetalOptions
	receivedCount := 0

	for receivedCount < 2 {
		select {
		case vm := <-vmChan:
			vmOptions = vm
			receivedCount++
		case bm := <-bareMetalChan:
			bareMetalOptions = bm
			receivedCount++
		case err := <-errChan:
			return nil, BareMetalOptions{}, err
		}
	}

	return vmOptions, bareMetalOptions, nil
}

func fetchVirtualMachineOptions(apiKey string) (VirtualMachineOptions, error) {
	url := "https://api.hyperbolic.xyz/v2/marketplace/virtual-machine-options"
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var vmOptions VirtualMachineOptions
	if err := json.Unmarshal(body, &vmOptions); err != nil {
		return nil, fmt.Errorf("error parsing VM options response: %v", err)
	}

	return vmOptions, nil
}

func fetchBareMetalOptions(apiKey string) (BareMetalOptions, error) {
	url := "https://api.hyperbolic.xyz/v2/marketplace/bare-metal-options"
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return BareMetalOptions{}, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return BareMetalOptions{}, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return BareMetalOptions{}, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return BareMetalOptions{}, fmt.Errorf("error reading response: %v", err)
	}

	var bareMetalOptions BareMetalOptions
	if err := json.Unmarshal(body, &bareMetalOptions); err != nil {
		return BareMetalOptions{}, fmt.Errorf("error parsing bare metal options response: %v", err)
	}

	return bareMetalOptions, nil
}



func init() {
	rootCmd.AddCommand(ondemandCmd)
	ondemandCmd.Flags().Bool("json", false, "Output raw JSON response")
} 